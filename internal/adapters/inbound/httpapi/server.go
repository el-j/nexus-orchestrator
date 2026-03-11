// Package httpapi provides the REST API inbound adapter for nexusOrchestrator.
// It serves task management endpoints under /api/tasks, provider discovery,
// and a Server-Sent Events stream for real-time updates.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server holds the HTTP API dependencies.
type Server struct {
	orch ports.Orchestrator
	hub  *Hub
}

// NewServer constructs a Server. hub may be nil to disable SSE.
func NewServer(orch ports.Orchestrator, hub *Hub) *Server {
	return &Server{orch: orch, hub: hub}
}

// broadcasterSetter is satisfied by *services.OrchestratorService. Defined here
// as a local interface to avoid importing the services package from an inbound adapter.
type broadcasterSetter interface {
	SetBroadcaster(ports.EventBroadcaster)
}

// Handler returns the fully configured chi router.
// maxBodySize limits request bodies to 1 MB.
func maxBodySize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		next.ServeHTTP(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(maxBodySize)
	r.Use(securityHeaders)

	// Redirect root to dashboard
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui", http.StatusFound)
	})
	r.Get("/ui", s.handleUI)

	// Task endpoints
	r.Post("/api/tasks", s.handleCreateTask)
	r.Get("/api/tasks", s.handleListTasks)
	r.Get("/api/tasks/{id}", s.handleGetTask)
	r.Delete("/api/tasks/{id}", s.handleCancelTask)

	// Provider + health
	r.Get("/api/providers", s.handleProviders)
	r.Post("/api/providers", s.handleRegisterProvider)
	r.Delete("/api/providers/{name}", s.handleRemoveProvider)
	r.Get("/api/providers/{name}/models", s.handleProviderModels)

	// Provider config CRUD (persistent, with API-key masking in responses)
	r.Post("/api/providers/config", s.handleAddProviderConfig)
	r.Get("/api/providers/config", s.handleListProviderConfigs)
	r.Put("/api/providers/config/{id}", s.handleUpdateProviderConfig)
	r.Delete("/api/providers/config/{id}", s.handleRemoveProviderConfig)

	r.Get("/api/health", s.handleHealth)

	// GET /api/events — SSE stream for task lifecycle events
	r.Get("/api/events", func(w http.ResponseWriter, r *http.Request) {
		if s.hub == nil {
			http.Error(w, "SSE not configured", http.StatusServiceUnavailable)
			return
		}
		s.hub.ServeSSE(w, r)
	})

	return r
}

// StartServer starts the HTTP API on addr and blocks until ctx is cancelled.
// Signature unchanged — existing callers (main.go, cmd/nexus-daemon/main.go) work without modification.
func StartServer(ctx context.Context, orch ports.Orchestrator, addr string) error {
	hub := NewHub()
	// Wire broadcaster if orch exposes SetBroadcaster (avoids importing services).
	if bs, ok := orch.(broadcasterSetter); ok {
		bs.SetBroadcaster(hub)
	}
	s := NewServer(orch, hub)
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // no write timeout — required for long-lived SSE connections
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			log.Printf("httpapi: shutdown: %v", err)
		}
	}()

	log.Printf("httpapi: listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req domain.Task
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	taskID, err := s.orch.SubmitTask(req)
	if err != nil {
		if errors.Is(err, domain.ErrNoPlan) {
			http.Error(w, "planning required before execution; submit a 'plan' task first", http.StatusUnprocessableEntity)
			return
		}
		log.Printf("httpapi: create task: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"task_id": taskID,
		"status":  string(domain.StatusQueued),
	})
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.orch.GetQueue()
	if err != nil {
		log.Printf("httpapi: list tasks: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if tasks == nil {
		tasks = []domain.Task{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tasks)
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	task, err := s.orch.GetTask(id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		log.Printf("httpapi: get task %s: %v", id, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(task)
}

func (s *Server) handleCancelTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.orch.CancelTask(id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		log.Printf("httpapi: cancel task %s: %v", id, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := s.orch.GetProviders()
	if err != nil {
		log.Printf("httpapi: get providers: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if providers == nil {
		providers = []ports.ProviderInfo{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(providers)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "nexus-orchestrator",
	})
}

func (s *Server) handleRegisterProvider(w http.ResponseWriter, r *http.Request) {
	var cfg domain.ProviderConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if cfg.Name == "" || cfg.Kind == "" {
		http.Error(w, "name and kind are required", http.StatusBadRequest)
		return
	}
	if err := s.orch.RegisterCloudProvider(cfg); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"name": cfg.Name, "kind": string(cfg.Kind)})
}

func (s *Server) handleRemoveProvider(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := s.orch.RemoveProvider(name); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "provider not found", http.StatusNotFound)
			return
		}
		log.Printf("httpapi: remove provider %s: %v", name, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleProviderModels(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	models, err := s.orch.GetProviderModels(name)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "provider not found", http.StatusNotFound)
			return
		}
		log.Printf("httpapi: provider models %s: %v", name, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if models == nil {
		models = []string{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(models)
}

// maskAPIKey returns a masked representation of an API key:
// if longer than 4 characters, the last 4 are preserved; otherwise "****".
func maskAPIKey(key string) string {
	if len(key) > 4 {
		return "****" + key[len(key)-4:]
	}
	return "****"
}

// maskedProviderConfig returns a copy of cfg with the APIKey field masked.
func maskedProviderConfig(cfg domain.ProviderConfig) domain.ProviderConfig {
	if cfg.APIKey != "" {
		cfg.APIKey = maskAPIKey(cfg.APIKey)
	}
	return cfg
}

func (s *Server) handleAddProviderConfig(w http.ResponseWriter, r *http.Request) {
	var cfg domain.ProviderConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if cfg.Name == "" || cfg.Kind == "" {
		http.Error(w, "name and kind are required", http.StatusBadRequest)
		return
	}
	created, err := s.orch.AddProviderConfig(r.Context(), cfg)
	if err != nil {
		log.Printf("httpapi: add provider config: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(maskedProviderConfig(created))
}

func (s *Server) handleListProviderConfigs(w http.ResponseWriter, r *http.Request) {
	cfgs, err := s.orch.ListProviderConfigs(r.Context())
	if err != nil {
		log.Printf("httpapi: list provider configs: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	masked := make([]domain.ProviderConfig, len(cfgs))
	for i, c := range cfgs {
		masked[i] = maskedProviderConfig(c)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(masked)
}

func (s *Server) handleUpdateProviderConfig(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var cfg domain.ProviderConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	cfg.ID = id
	updated, err := s.orch.UpdateProviderConfig(r.Context(), cfg)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "provider config not found", http.StatusNotFound)
			return
		}
		log.Printf("httpapi: update provider config %s: %v", id, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(maskedProviderConfig(updated))
}

func (s *Server) handleRemoveProviderConfig(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.orch.RemoveProviderConfig(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "provider config not found", http.StatusNotFound)
			return
		}
		log.Printf("httpapi: remove provider config %s: %v", id, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
