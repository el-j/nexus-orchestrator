package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"nexus-ai/internal/core/domain"
	"nexus-ai/internal/core/ports"
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
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(task)
}

func (s *Server) handleCancelTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.orch.CancelTask(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := s.orch.GetProviders()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		"service": "nexus-ai",
	})
}
