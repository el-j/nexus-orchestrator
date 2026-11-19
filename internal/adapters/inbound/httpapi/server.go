// Package httpapi provides the REST API inbound adapter for nexusOrchestrator.
// It serves task management endpoints under /api/tasks, provider discovery,
// and a Server-Sent Events stream for real-time updates.
package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"nexus-orchestrator/internal/adapters/inbound/httpguard"
	"nexus-orchestrator/internal/adapters/inbound/webui"
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
	brain       ports.BrainService
	guard       *httpguard.Policy
	mcpAddr     string       // listen address of the companion MCP server, advertised in discovery
	ui          http.Handler // the full frontend, served under /app/ (nil: only the built-in dashboard)
}

// NewServer constructs a Server. hub may be nil to disable SSE.
func NewServer(orch ports.Orchestrator, brain ports.BrainService, hub *Hub) *Server {
	return &Server{orch: orch, brain: brain, hub: hub}
}

// WithGuard sets the Origin/Host policy applied to every request. Without it
// the handler still rejects foreign browser origins but does not check Host.
func (s *Server) WithGuard(g *httpguard.Policy) *Server {
	s.guard = g
	return s
}

// WithUI serves the full frontend under /app/ and makes the root point there. h is expected to
// answer the whole /app/ subtree (see package webui); nil keeps only the built-in dashboard.
func (s *Server) WithUI(h http.Handler) *Server {
	s.ui = h
	return s
}

// WithMCPAddr records the listen address of the companion MCP server so the
// discovery document can advertise where it actually is.
func (s *Server) WithMCPAddr(addr string) *Server {
	s.mcpAddr = addr
	return s
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

// securityHeaders sets the headers that keep the dashboard from being framed or sniffed.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware allows requests from the Wails WebView (wails://wails on macOS and Linux,
// http://wails.localhost on Windows) and the local browser.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "wails://wails" || origin == "wails://wails.localhost" || origin == "http://wails.localhost" || strings.HasPrefix(origin, "http://localhost:") {
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

// tokenAuthMiddleware requires the bearer token when one is set in the environment or the settings;
// without one the API is open on the loopback.
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
		// Constant-time comparison so response timing does not leak the token.
		given := strings.TrimSpace(strings.TrimPrefix(auth, prefix))
		if !strings.HasPrefix(auth, prefix) || subtle.ConstantTimeCompare([]byte(given), []byte(token)) != 1 {
			writeJSONError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Handler returns the server's root http.Handler, with every middleware applied.
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(accessLog)
	r.Use(middleware.Recoverer)
	guard := s.guard
	if guard == nil {
		guard = httpguard.New("")
	}
	r.Use(guard.Middleware)
	r.Use(corsMiddleware)
	r.Use(s.tokenAuthMiddleware)
	r.Use(maxBodySize)
	r.Use(securityHeaders)

	s.routeUI(r)
	s.routeTasks(r)
	s.routeProviders(r)
	s.routeSessions(r)
	s.routeSystem(r)
	s.routeBrain(r)

	return r
}

// shutdownGrace is how long in-flight requests get to finish on shutdown before
// the remaining connections are closed.
const shutdownGrace = 2 * time.Second

// StartServer starts the HTTP API on addr and blocks until ctx is cancelled.
// An optional *LogHub may be passed as the final argument to capture log output via SSE.
func StartServer(ctx context.Context, orch ports.Orchestrator, brain ports.BrainService, addr string, logHub ...*LogHub) error {
	hub := NewHub()
	// Wire broadcaster if orch exposes SetBroadcaster (avoids importing services).
	if bs, ok := orch.(broadcasterSetter); ok {
		bs.SetBroadcaster(hub)
	}
	s := NewServer(orch, brain, hub).WithGuard(httpguard.New(addr)).WithUI(webui.Embedded("/app/"))
	if len(logHub) > 0 && logHub[0] != nil {
		s.WithLogHub(logHub[0])
	}
	return runHTTP(ctx, addr, s.Handler(), shutdownGrace)
}

// runHTTP serves handler on addr until ctx is cancelled, then gives in-flight requests grace to
// finish before closing the rest.
func runHTTP(ctx context.Context, addr string, handler http.Handler, grace time.Duration) error {
	// Cancelled when shutdown starts so long-lived /api/events streams end
	// promptly (http.Server.Shutdown does not cancel request contexts itself).
	baseCtx, cancelBase := context.WithCancel(context.Background())
	defer cancelBase()
	srv := &http.Server{
		Addr:         addr,
		BaseContext:  func(net.Listener) context.Context { return baseCtx },
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // no write timeout — required for long-lived SSE connections
		IdleTimeout:  60 * time.Second,
	}

	drained := make(chan struct{})
	go func() {
		defer close(drained)
		<-ctx.Done()
		cancelBase()
		shutCtx, cancel := context.WithTimeout(context.Background(), grace)
		defer cancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			log.Printf("httpapi: shutdown grace period elapsed, closing remaining connections: %v", err)
			_ = srv.Close()
		}
	}()

	log.Printf("httpapi: listening on %s", addr)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err // the address could not be served; nothing is shutting down
	}
	<-drained // ListenAndServe returns as soon as Shutdown starts: wait until the in-flight requests are done
	return nil
}

// activityBroadcasterSetter is satisfied by *services.ActivityService.
// Defined as a local interface to avoid importing the services package from an inbound adapter.
type activityBroadcasterSetter interface {
	SetBroadcaster(ports.ActivityBroadcaster)
}

// StartServerFull is like StartServer but also wires an activity service for
// the activity observatory endpoints and SSE broadcasting of activity events.
func StartServerFull(ctx context.Context, orch ports.Orchestrator, brain ports.BrainService, addr string, actSvc activityQuerier, logHub ...*LogHub) error {
	extras := Extras{Activity: actSvc}
	if len(logHub) > 0 {
		extras.LogHub = logHub[0]
	}
	return StartServerWith(ctx, orch, brain, addr, extras)
}

// Extras are the optional collaborators of StartServerWith.
type Extras struct {
	Activity activityQuerier // activity timeline service; nil disables /api/activities
	LogHub   *LogHub         // log capture for /api/logs; nil disables it
	MCPAddr  string          // companion MCP server address, advertised by /.well-known/nexus.json
}

// StartServerWith starts the REST API with every optional collaborator in
// Extras. It blocks until ctx is cancelled.
func StartServerWith(ctx context.Context, orch ports.Orchestrator, brain ports.BrainService, addr string, extras Extras) error {
	actSvc, logHub := extras.Activity, extras.LogHub
	hub := NewHub()
	if bs, ok := orch.(broadcasterSetter); ok {
		bs.SetBroadcaster(hub)
	}
	if actSvc != nil {
		if abs, ok := actSvc.(activityBroadcasterSetter); ok {
			abs.SetBroadcaster(hub)
		}
	}
	s := NewServer(orch, brain, hub).WithGuard(httpguard.New(addr)).WithUI(webui.Embedded("/app/")).WithMCPAddr(extras.MCPAddr)
	if logHub != nil {
		s.WithLogHub(logHub)
	}
	if actSvc != nil {
		s.WithActivityService(actSvc)
	}
	// Start periodic background plan discovery scans.
	StartPlanScanWorker(ctx, orch, 5*time.Minute)
	return runHTTP(ctx, addr, s.Handler(), shutdownGrace)
}

// writeJSONError sets Content-Type to application/json, writes the HTTP status
// code, and encodes {"error":"<msg>"} as the response body.
func writeJSONError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": msg}); err != nil {
		log.Printf("httpapi: json encode: %v", err)
	}
}

// writeQueueFull answers 429 with a Retry-After hint: a full queue is a
// transient condition clients should back off from, not a server fault.
func writeQueueFull(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "5")
	writeJSONError(w, "queue is full; retry later", http.StatusTooManyRequests)
}

// writeJSON sets Content-Type to application/json, writes the given HTTP status
// code, and encodes v as the response body.
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("httpapi: json encode: %v", err)
	}
}

// handleHealth answers the liveness probe with the service name.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
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
	writeJSON(w, http.StatusOK, entries)
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
