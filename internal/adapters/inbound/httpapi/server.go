// Package httpapi provides the REST API inbound adapter for nexusOrchestrator.
// It serves task management endpoints under /api/tasks, provider discovery,
// and a Server-Sent Events stream for real-time updates.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server holds the HTTP API dependencies.
type Server struct {
	orch   ports.Orchestrator
	hub    *Hub
	logHub *LogHub
}

// NewServer constructs a Server. hub may be nil to disable SSE.
func NewServer(orch ports.Orchestrator, hub *Hub) *Server {
	return &Server{orch: orch, hub: hub}
}

// WithLogHub configures the Server to capture and stream log entries via SSE.
func (s *Server) WithLogHub(h *LogHub) *Server {
	s.logHub = h
	return s
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

// corsMiddleware allows requests from the Wails WebView and local browser.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "wails://wails.localhost" || strings.HasPrefix(origin, "http://localhost:") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)
	r.Use(maxBodySize)
	r.Use(securityHeaders)

	// Redirect root to dashboard
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui", http.StatusFound)
	})
	r.Get("/ui", s.handleUI)

	// Task endpoints — literal segments must be registered before wildcard {id}
	r.Post("/api/tasks", s.handleCreateTask)
	r.Get("/api/tasks", s.handleListTasks)
	r.Get("/api/tasks/all", s.handleGetAllTasks)
	r.Post("/api/tasks/draft", s.handleCreateDraft)
	r.Get("/api/tasks/backlog", s.handleGetBacklog)
	r.Get("/api/tasks/{id}", s.handleGetTask)
	r.Delete("/api/tasks/{id}", s.handleCancelTask)
	r.Post("/api/tasks/{id}/promote", s.handlePromoteTask)
	r.Put("/api/tasks/{id}", s.handleUpdateTask)

	// Provider + health
	r.Get("/api/providers", s.handleProviders)
	r.Post("/api/providers", s.handleRegisterProvider)

	// Provider discovery — literal segments registered before wildcard {name}
	r.Get("/api/providers/discovered", s.handleGetDiscoveredProviders)
	r.Post("/api/providers/discovered/scan", s.handleTriggerScan)
	r.Post("/api/providers/promote/{id}", s.handlePromoteProvider)

	r.Delete("/api/providers/{name}", s.handleRemoveProvider)
	r.Get("/api/providers/{name}/models", s.handleProviderModels)

	// Provider config CRUD (persistent, with API-key masking in responses)
	r.Post("/api/providers/config", s.handleAddProviderConfig)
	r.Get("/api/providers/config", s.handleListProviderConfigs)
	r.Put("/api/providers/config/{id}", s.handleUpdateProviderConfig)
	r.Delete("/api/providers/config/{id}", s.handleRemoveProviderConfig)

	// AI session endpoints — literal segment before wildcard {id}
	r.Post("/api/ai-sessions", s.handleRegisterAISession)
	r.Get("/api/ai-sessions", s.handleListAISessions)
	r.Delete("/api/ai-sessions/{id}", s.handleDeregisterAISession)
	r.Post("/api/ai-sessions/{id}/heartbeat", s.handleHeartbeatAISession)
	r.Get("/api/ai-sessions/{id}/tasks", s.handleGetSessionTasks)

	// Task claim + external status update
	r.Post("/api/tasks/{id}/claim", s.handleClaimTask)
	r.Put("/api/tasks/{id}/status", s.handleUpdateTaskStatus)

	r.Get("/api/health", s.handleHealth)
	r.Get("/api/logs", s.handleGetLogs)

	// GET /api/events — SSE stream for task lifecycle and log events
	r.Get("/api/events", s.handleEvents)

	return r
}

// StartServer starts the HTTP API on addr and blocks until ctx is cancelled.
// An optional *LogHub may be passed as the final argument to capture log output via SSE.
func StartServer(ctx context.Context, orch ports.Orchestrator, addr string, logHub ...*LogHub) error {
	hub := NewHub()
	// Wire broadcaster if orch exposes SetBroadcaster (avoids importing services).
	if bs, ok := orch.(broadcasterSetter); ok {
		bs.SetBroadcaster(hub)
	}
	s := NewServer(orch, hub)
	if len(logHub) > 0 && logHub[0] != nil {
		s.WithLogHub(logHub[0])
	}
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

// writeJSONError sets Content-Type to application/json, writes the HTTP status
// code, and encodes {"error":"<msg>"} as the response body.
func writeJSONError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req domain.Task
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	taskID, err := s.orch.SubmitTask(req)
	if err != nil {
		if errors.Is(err, domain.ErrNoPlan) {
			writeJSONError(w, "planning required before execution; submit a 'plan' task first", http.StatusUnprocessableEntity)
			return
		}
		log.Printf("httpapi: create task: %v", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
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
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if tasks == nil {
		tasks = []domain.Task{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tasks)
}

func (s *Server) handleGetAllTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.orch.GetAllTasks()
	if err != nil {
		log.Printf("httpapi: get all tasks: %v", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
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
			writeJSONError(w, "task not found", http.StatusNotFound)
			return
		}
		log.Printf("httpapi: get task %s: %v", id, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(task)
}

func (s *Server) handleCancelTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.orch.CancelTask(id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "task not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "cannot cancel task with status") {
			writeJSONError(w, err.Error(), http.StatusConflict)
			return
		}
		log.Printf("httpapi: cancel task %s: %v", id, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCreateDraft(w http.ResponseWriter, r *http.Request) {
	var task domain.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		writeJSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	id, err := s.orch.CreateDraft(task)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"id": id, "status": "DRAFT"})
}

func (s *Server) handleGetBacklog(w http.ResponseWriter, r *http.Request) {
	projectPath := r.URL.Query().Get("project")
	tasks, err := s.orch.GetBacklog(projectPath)
	if err != nil {
		log.Printf("httpapi: get backlog: %v", err)
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tasks == nil {
		tasks = []domain.Task{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tasks)
}

func (s *Server) handlePromoteTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.orch.PromoteTask(id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "task not found", http.StatusNotFound)
			return
		}
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var updates domain.Task
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeJSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	updated, err := s.orch.UpdateTask(id, updates)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "task not found", http.StatusNotFound)
			return
		}
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(updated)
}

func (s *Server) handleProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := s.orch.GetProviders()
	if err != nil {
		log.Printf("httpapi: get providers: %v", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
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
		writeJSONError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if cfg.Name == "" || cfg.Kind == "" {
		writeJSONError(w, "name and kind are required", http.StatusBadRequest)
		return
	}
	if err := s.orch.RegisterCloudProvider(cfg); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, err.Error(), http.StatusConflict)
			return
		}
		writeJSONError(w, err.Error(), http.StatusUnprocessableEntity)
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
			writeJSONError(w, "provider not found", http.StatusNotFound)
			return
		}
		log.Printf("httpapi: remove provider %s: %v", name, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleProviderModels(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	models, err := s.orch.GetProviderModels(name)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "provider not found", http.StatusNotFound)
			return
		}
		log.Printf("httpapi: provider models %s: %v", name, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
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
		writeJSONError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if cfg.Name == "" || cfg.Kind == "" {
		writeJSONError(w, "name and kind are required", http.StatusBadRequest)
		return
	}
	created, err := s.orch.AddProviderConfig(r.Context(), cfg)
	if err != nil {
		log.Printf("httpapi: add provider config: %v", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
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
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
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
		writeJSONError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	cfg.ID = id
	updated, err := s.orch.UpdateProviderConfig(r.Context(), cfg)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "provider config not found", http.StatusNotFound)
			return
		}
		log.Printf("httpapi: update provider config %s: %v", id, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(maskedProviderConfig(updated))
}

func (s *Server) handleRemoveProviderConfig(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.orch.RemoveProviderConfig(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "provider config not found", http.StatusNotFound)
			return
		}
		log.Printf("httpapi: remove provider config %s: %v", id, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetDiscoveredProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := s.orch.GetDiscoveredProviders()
	if err != nil {
		writeJSONError(w, "failed to get discovered providers", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(providers)
}

func (s *Server) handleTriggerScan(w http.ResponseWriter, r *http.Request) {
	providers, err := s.orch.TriggerScan(r.Context())
	if err != nil {
		writeJSONError(w, "scan failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(providers)
}

func (s *Server) handlePromoteProvider(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	err := s.orch.PromoteProvider(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "provider not found", http.StatusNotFound)
			return
		}
		writeJSONError(w, "failed to promote provider", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleGetLogs returns a JSON array of buffered log entries from the ring buffer.
func (s *Server) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	if s.logHub == nil {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
		return
	}
	entries := s.logHub.Buffer()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(entries)
}

func (s *Server) handleRegisterAISession(w http.ResponseWriter, r *http.Request) {
	var session domain.AISession
	if err := json.NewDecoder(r.Body).Decode(&session); err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if session.AgentName == "" {
		writeJSONError(w, "agentName is required", http.StatusBadRequest)
		return
	}
	if session.Source != domain.SessionSourceMCP && session.Source != domain.SessionSourceVSCode && session.Source != domain.SessionSourceHTTP {
		writeJSONError(w, "source must be one of: mcp, vscode, http", http.StatusBadRequest)
		return
	}
	created, err := s.orch.RegisterAISession(r.Context(), session)
	if err != nil {
		log.Printf("httpapi: register ai session: %v", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (s *Server) handleListAISessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.orch.ListAISessions(r.Context())
	if err != nil {
		log.Printf("httpapi: list ai sessions: %v", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if sessions == nil {
		sessions = []domain.AISession{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(sessions)
}

func (s *Server) handleDeregisterAISession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.orch.DeregisterAISession(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "ai session not found", http.StatusNotFound)
			return
		}
		log.Printf("httpapi: deregister ai session %s: %v", id, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleHeartbeatAISession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.orch.HeartbeatAISession(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "ai session not found", http.StatusNotFound)
			return
		}
		log.Printf("httpapi: heartbeat ai session %s: %v", id, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleClaimTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.SessionID == "" {
		writeJSONError(w, "sessionId is required", http.StatusBadRequest)
		return
	}
	task, err := s.orch.ClaimTask(r.Context(), id, body.SessionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "task or session not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "not QUEUED") || strings.Contains(err.Error(), "is disconnected") {
			writeJSONError(w, err.Error(), http.StatusConflict)
			return
		}
		log.Printf("httpapi: claim task %s: %v", id, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(task)
}

func (s *Server) handleUpdateTaskStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		SessionID string `json:"sessionId"`
		Status    string `json:"status"`
		Logs      string `json:"logs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.SessionID == "" || body.Status == "" {
		writeJSONError(w, "sessionId and status are required", http.StatusBadRequest)
		return
	}
	if body.Status != "COMPLETED" && body.Status != "FAILED" {
		writeJSONError(w, "status must be COMPLETED or FAILED", http.StatusBadRequest)
		return
	}
	task, err := s.orch.UpdateTaskStatus(r.Context(), id, body.SessionID, domain.TaskStatus(body.Status), body.Logs)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "task not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "does not own") {
			writeJSONError(w, err.Error(), http.StatusForbidden)
			return
		}
		if strings.Contains(err.Error(), "not PROCESSING") || strings.Contains(err.Error(), "invalid target status") {
			writeJSONError(w, err.Error(), http.StatusConflict)
			return
		}
		log.Printf("httpapi: update task status %s: %v", id, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(task)
}

func (s *Server) handleGetSessionTasks(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// We use GetAllTasks and filter by AISessionID since there's no direct port method on Orchestrator
	// for session-scoped task query. The repo method is on TaskRepository, not Orchestrator.
	allTasks, err := s.orch.GetAllTasks()
	if err != nil {
		log.Printf("httpapi: get session tasks %s: %v", id, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	var sessionTasks []domain.Task
	for _, t := range allTasks {
		if t.AISessionID == id {
			sessionTasks = append(sessionTasks, t)
		}
	}
	if sessionTasks == nil {
		sessionTasks = []domain.Task{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(sessionTasks)
}

// handleEvents serves a Server-Sent Events stream that multiplexes task lifecycle
// events (default event type) and log entries (event: log).
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if s.hub == nil {
		writeJSONError(w, "SSE not configured", http.StatusServiceUnavailable)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONError(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	taskCh := s.hub.Subscribe()
	defer s.hub.Unsubscribe(taskCh)

	var logCh chan domain.LogEntry
	if s.logHub != nil {
		logCh = s.logHub.Subscribe()
		defer s.logHub.Unsubscribe(logCh)
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	fmt.Fprint(w, "data: {\"type\":\"connected\"}\n\n")
	flusher.Flush()

	for {
		select {
		case msg, ok := <-taskCh:
			if !ok {
				return
			}
			if _, err := w.Write(msg); err != nil {
				return
			}
			flusher.Flush()
		case entry, ok := <-logCh:
			if !ok {
				return
			}
			data, err := json.Marshal(entry)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: log\ndata: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
