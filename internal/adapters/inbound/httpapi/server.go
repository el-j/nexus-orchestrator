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
	"os"
	"strings"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server holds the HTTP API dependencies.
type Server struct {
	orch        ports.Orchestrator
	hub         *Hub
	logHub      *LogHub
	activitySvc activityQuerier
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

// WithActivityService injects an activity querier for the activity endpoints.
func (s *Server) WithActivityService(svc activityQuerier) *Server {
	s.activitySvc = svc
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
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) tokenAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := os.Getenv("NEXUS_API_TOKEN")
		if token == "" && s.orch != nil {
			if cfg, err := s.orch.GetRuntimeConfig(r.Context()); err == nil {
				token = cfg.APIToken
			}
		}

		// If no token is configured, allow requests (backward compatible default).
		if token == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Only guard /api/* endpoints. UI routes remain public so the app can load.
		path := r.URL.Path
		if !strings.HasPrefix(path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		// Allow unauthenticated health and discovery docs.
		if path == "/api/health" || path == "/api/howto" || path == "/.well-known/nexus.json" {
			next.ServeHTTP(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(auth, prefix) || strings.TrimSpace(strings.TrimPrefix(auth, prefix)) != token {
			writeJSONError(w, "unauthorized", http.StatusUnauthorized)
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
	r.Use(s.tokenAuthMiddleware)
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
	r.Delete("/api/ai-sessions", s.handlePurgeDisconnectedSessions)
	r.Get("/api/ai-sessions/discovered", s.handleGetDiscoveredAgents)
	r.Delete("/api/ai-sessions/{id}", s.handleDeregisterAISession)
	r.Post("/api/ai-sessions/{id}/heartbeat", s.handleHeartbeatAISession)
	r.Post("/api/ai-sessions/{id}/terminate", s.handleTerminateAISession)
	r.Get("/api/ai-sessions/{id}/tasks", s.handleGetSessionTasks)
	r.Post("/api/ai-sessions/{id}/delegate", s.handleDelegateToNexus)

	// Task claim + external status update
	r.Post("/api/tasks/{id}/claim", s.handleClaimTask)
	r.Put("/api/tasks/{id}/status", s.handleUpdateTaskStatus)
	r.Post("/api/tasks/{id}/heartbeat", s.handleHeartbeatTask)

	r.Get("/api/health", s.handleHealth)
	r.Get("/api/logs", s.handleGetLogs)
	// Runtime config
	r.Get("/api/config", s.handleGetConfig)
	r.Put("/api/config", s.handlePutConfig)

	// GET /api/events — SSE stream for task lifecycle and log events
	r.Get("/api/events", s.handleEvents)
	// Activity observatory endpoints
	r.Get("/api/activities", s.handleListActivities)
	r.Get("/api/activities/timeline", s.handleActivityTimeline)

	// Plan file discovery endpoints
	r.Get("/api/plans/discovered", s.handleGetDiscoveredPlanFiles)
	r.Post("/api/plans/discovered/scan", s.handleScanPlanFiles)

	// Discovery + how-to
	r.Get("/api/howto", s.handleHowto)
	r.Get("/.well-known/nexus.json", s.handleWellKnownNexus)

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

// activityBroadcasterSetter is satisfied by *services.ActivityService.
// Defined as a local interface to avoid importing the services package from an inbound adapter.
type activityBroadcasterSetter interface {
	SetBroadcaster(ports.ActivityBroadcaster)
}

// StartServerFull is like StartServer but also wires an activity service for
// the activity observatory endpoints and SSE broadcasting of activity events.
func StartServerFull(ctx context.Context, orch ports.Orchestrator, addr string, actSvc activityQuerier, logHub ...*LogHub) error {
	hub := NewHub()
	if bs, ok := orch.(broadcasterSetter); ok {
		bs.SetBroadcaster(hub)
	}
	if actSvc != nil {
		if abs, ok := actSvc.(activityBroadcasterSetter); ok {
			abs.SetBroadcaster(hub)
		}
	}
	s := NewServer(orch, hub)
	if len(logHub) > 0 && logHub[0] != nil {
		s.WithLogHub(logHub[0])
	}
	if actSvc != nil {
		s.WithActivityService(actSvc)
	}
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // no write timeout — required for long-lived SSE connections
		IdleTimeout:  60 * time.Second,
	}

	// Start periodic background plan discovery scans.
	StartPlanScanWorker(ctx, orch, 5*time.Minute)

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

// writeJSON sets Content-Type to application/json, writes the given HTTP status
// code, and encodes v as the response body.
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "nexus-orchestrator",
	})
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
