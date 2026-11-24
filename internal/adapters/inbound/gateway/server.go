package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Server is the HTTP server providing an OpenAI-compatible API reverse proxy.
type Server struct {
	cfg Config
	gw  *Gateway
	mux *http.ServeMux
}

// NewServer creates a new OpenAI-compatible HTTP server handler.
func NewServer(cfg Config, gw *Gateway) *Server {
	s := &Server{
		cfg: cfg,
		gw:  gw,
		mux: http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

// Routes returns the bare routes, without CORS or key check: for mounting in the daemon, whose own
// origin, host and token guards stand in front of it.
func (s *Server) Routes() http.Handler { return s.mux }

// projectFor is the project a request is about: X-Nexus-Project, else the one the server was started with.
func (s *Server) projectFor(r *http.Request) string {
	if p := strings.TrimSpace(r.Header.Get("X-Nexus-Project")); p != "" {
		return p
	}
	return s.cfg.ProjectPath
}

// Handler returns the underlying http.Handler with middleware applied.
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// API Key authentication if configured
		if s.cfg.APIKey != "" {
			authHeader := r.Header.Get("Authorization")
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if strings.TrimSpace(token) != s.cfg.APIKey {
				s.writeError(w, http.StatusUnauthorized, "Invalid API key", "invalid_request_error")
				return
			}
		}

		s.mux.ServeHTTP(w, r)
	})
}

// registerRoutes sets up the supported OpenAI-compatible routes.
func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/v1/chat/completions", s.handleChatCompletions)
	s.mux.HandleFunc("/chat/completions", s.handleChatCompletions)
	s.mux.HandleFunc("/v1/models", s.handleModels)
	s.mux.HandleFunc("/models", s.handleModels)
	s.mux.HandleFunc("/v1/health", s.handleHealth)
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/", s.handleNotFound)
}

// handleChatCompletions handles both streaming and non-streaming chat completion requests.
func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error")
		return
	}

	var req ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON payload: %v", err), "invalid_request_error")
		return
	}

	if len(req.Messages) == 0 {
		s.writeError(w, http.StatusBadRequest, "messages array cannot be empty", "invalid_request_error")
		return
	}

	if req.HistoryBudget <= 0 {
		if h := r.Header.Get("X-Nexus-History-Budget"); h != "" {
			if b, err := strconv.Atoi(strings.TrimSpace(h)); err == nil && b > 0 {
				req.HistoryBudget = b
			}
		} else if q := r.URL.Query().Get("history_budget"); q != "" {
			if b, err := strconv.Atoi(strings.TrimSpace(q)); err == nil && b > 0 {
				req.HistoryBudget = b
			}
		}
	}

	if req.Stream {
		s.handleStreamingChat(w, r, req)
		return
	}

	resp, err := s.gw.ExecuteChat(r.Context(), req, s.projectFor(r))
	if err != nil {
		s.writeError(w, http.StatusBadGateway, err.Error(), "nexus_router_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// handleStreamingChat delivers responses chunk-by-chunk using Server-Sent Events.
func (s *Server) handleStreamingChat(w http.ResponseWriter, r *http.Request, req ChatCompletionRequest) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeError(w, http.StatusInternalServerError, "streaming unsupported by server", "server_error")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	id := "chatcmpl-" + uuid.NewString()
	created := time.Now().Unix()
	model := req.Model
	if model == "" {
		model = s.cfg.DefaultRole
	}

	// 1. Initial role delta chunk
	initChunk := ChatCompletionChunk{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   model,
		Choices: []ChatChunkChoice{
			{
				Index: 0,
				Delta: ChatChunkDelta{
					Role: "assistant",
				},
			},
		},
	}
	s.writeSSEChunk(w, flusher, initChunk)

	// 2. Stream content deltas
	onChunk := func(piece string) error {
		chunk := ChatCompletionChunk{
			ID:      id,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   model,
			Choices: []ChatChunkChoice{
				{
					Index: 0,
					Delta: ChatChunkDelta{
						Content: piece,
					},
				},
			},
		}
		s.writeSSEChunk(w, flusher, chunk)
		return nil
	}

	_, err := s.gw.StreamChat(r.Context(), req, s.projectFor(r), onChunk)
	if err != nil {
		errChunk := ChatCompletionChunk{
			ID:      id,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   model,
			Choices: []ChatChunkChoice{
				{
					Index: 0,
					Delta: ChatChunkDelta{
						Content: fmt.Sprintf("\n[stream error: %v]", err),
					},
				},
			},
		}
		s.writeSSEChunk(w, flusher, errChunk)
	}

	// 3. Final stop chunk & [DONE]
	stop := "stop"
	stopChunk := ChatCompletionChunk{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   model,
		Choices: []ChatChunkChoice{
			{
				Index:        0,
				Delta:        ChatChunkDelta{},
				FinishReason: &stop,
			},
		},
	}
	s.writeSSEChunk(w, flusher, stopChunk)
	_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()
}

// writeSSEChunk writes a single serialized SSE chunk to the response writer.
func (s *Server) writeSSEChunk(w http.ResponseWriter, flusher http.Flusher, chunk ChatCompletionChunk) {
	data, err := json.Marshal(chunk)
	if err == nil {
		_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}

// handleModels returns all active models and supported virtual roles.
func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error")
		return
	}

	items := s.gw.ListModels(r.Context())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(ModelList{
		Object: "list",
		Data:   items,
	})
}

// handleHealth returns basic service liveness.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","service":"nexus-router"}`))
}

// handleNotFound returns standard OpenAI 404 error payload.
func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	s.writeError(w, http.StatusNotFound, fmt.Sprintf("unrecognized path %s", r.URL.Path), "invalid_request_error")
}

// writeError serializes and writes an OpenAI-style JSON error response.
func (s *Server) writeError(w http.ResponseWriter, status int, msg, errType string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIError{
		Error: ErrorDetails{
			Message: msg,
			Type:    errType,
		},
	})
}
