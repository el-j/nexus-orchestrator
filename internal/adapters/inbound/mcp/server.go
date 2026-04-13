// Package mcp provides a JSON-RPC 2.0 Model Context Protocol server as an
// inbound adapter. It exposes task management tools compatible with Claude
// Desktop and other MCP clients.
package mcp

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"nexus-orchestrator/internal/core/ports"
)

// JSON-RPC 2.0 error codes.
const (
	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternalError  = -32603
)

// ----- JSON-RPC 2.0 envelope types -----

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ----- MCP protocol types -----

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	// Instructions is surfaced by MCP clients (e.g. Claude Desktop, Cursor) as a
	// system-level hint that tells the AI how to start working with this server.
	Instructions string `json:"instructions,omitempty"`
}

type capabilities struct {
	Tools map[string]any `json:"tools"`
}

type initializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    capabilities `json:"capabilities"`
	ServerInfo      serverInfo   `json:"serverInfo"`
}

type propertyItems struct {
	Type string `json:"type"`
}

type property struct {
	Type        string         `json:"type"`
	Description string         `json:"description"`
	Items       *propertyItems `json:"items,omitempty"` // required by JSON Schema when type=="array"
}

type inputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]property `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

type toolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema inputSchema `json:"inputSchema"`
}

type contentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type callToolResult struct {
	Content []contentItem `json:"content"`
}

// ----- Server -----

// Server is the MCP inbound adapter.
type Server struct {
	orch           ports.Orchestrator
	brain          ports.BrainService
	mux            *http.ServeMux
	sse            *sseManager
	allowedOrigins map[string]bool
	// authToken is the env-configured token (NEXUS_MCP_TOKEN). When set it
	// takes precedence over persisted runtime config.
	authToken string
}

// NewMcpServer creates a Server and registers its HTTP handlers.
func NewMcpServer(orch ports.Orchestrator, brain ports.BrainService) *Server {
	s := &Server{
		orch:  orch,
		brain: brain,
		mux:   http.NewServeMux(),
		sse:   &sseManager{sessions: make(map[string]*sseSession)},
		allowedOrigins: map[string]bool{
			"http://localhost":  true,
			"http://127.0.0.1":  true,
			"https://localhost": true,
			"https://127.0.0.1": true,
		},
		authToken: strings.TrimSpace(os.Getenv("NEXUS_MCP_TOKEN")),
	}
	s.mux.HandleFunc("/mcp", s.handleRPC)
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/sse", s.handleSSE)
	s.mux.HandleFunc("/messages", s.handleSSEMessage)
	return s
}

// ServeHTTP implements http.Handler so *Server can be passed to httptest.NewServer.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// CORS preflight.
	if r.Method == http.MethodOptions {
		s.writeCORSHeaders(w, r)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Origin validation per MCP spec security requirements.
	origin := r.Header.Get("Origin")
	if origin != "" && !s.isAllowedOrigin(origin) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if s.requiresAuth(r.Context(), r.URL.Path) && !s.isAuthorized(r.Context(), r.Header.Get("Authorization")) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	s.writeCORSHeaders(w, r)
	s.mux.ServeHTTP(w, r)
}

func (s *Server) isAllowedOrigin(origin string) bool {
	if s.allowedOrigins[origin] {
		return true
	}
	// Allow origins with any port on localhost/127.0.0.1.
	for allowed := range s.allowedOrigins {
		if strings.HasPrefix(origin, allowed+":") {
			return true
		}
	}
	return false
}

func (s *Server) writeCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin != "" && s.isAllowedOrigin(origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Mcp-Session-Id, MCP-Protocol-Version")
	w.Header().Set("Access-Control-Expose-Headers", "Mcp-Session-Id")
}

func (s *Server) requiresAuth(ctx context.Context, path string) bool {
	if s.effectiveAuthToken(ctx) == "" {
		return false
	}
	switch path {
	case "/mcp", "/sse", "/messages":
		return true
	default:
		return false
	}
}

func (s *Server) isAuthorized(ctx context.Context, authHeader string) bool {
	expected := s.effectiveAuthToken(ctx)
	if expected == "" {
		return true
	}
	if len(authHeader) < len("Bearer ")+1 {
		return false
	}
	if !strings.EqualFold(authHeader[:len("Bearer ")], "Bearer ") {
		return false
	}
	token := strings.TrimSpace(authHeader[len("Bearer "):])
	if token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1
}

func (s *Server) effectiveAuthToken(ctx context.Context) string {
	if strings.TrimSpace(s.authToken) != "" {
		return strings.TrimSpace(s.authToken)
	}
	if s.orch == nil {
		return ""
	}
	cfg, err := s.orch.GetRuntimeConfig(ctx)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cfg.MCPToken)
}

// StartMCPServer runs an HTTP server serving the MCP JSON-RPC 2.0 endpoint.
// It blocks until ctx is cancelled, then shuts down gracefully.
func StartMCPServer(ctx context.Context, orch ports.Orchestrator, brain ports.BrainService, addr string) error {
	handler := NewMcpServer(orch, brain)
	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
		// ReadTimeout is deliberately 0 so that long-lived SSE connections are not
		// killed after an idle period. Individual RPC handlers impose their own
		// context deadlines where needed.
		ReadTimeout: 0,
		IdleTimeout: 120 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		log.Printf("mcp: listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("mcp: listen: %w", err)
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			return fmt.Errorf("mcp: shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		return err
	}
}

// ----- HTTP handlers -----

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleRPC(w http.ResponseWriter, r *http.Request) {
	// Streamable HTTP: POST for messages, GET returns 405 (no standalone SSE stream).
	if r.Method == http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, nil, codeParseError, "parse error")
		return
	}
	if req.JSONRPC != "2.0" {
		writeError(w, req.ID, codeInvalidRequest, `invalid request: jsonrpc must be "2.0"`)
		return
	}
	s.processRPC(w, r, req)
}

// processRPC handles a single JSON-RPC request and writes the response to w.
// Used by both the direct HTTP transport and the SSE transport.
func (s *Server) processRPC(w http.ResponseWriter, r *http.Request, req rpcRequest) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(w, req)
	case "notifications/initialized":
		w.WriteHeader(http.StatusNoContent)
	case "tools/list":
		resp := rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"tools": toolList()}}
		_ = json.NewEncoder(w).Encode(resp)
	case "tools/call":
		s.handleToolCall(w, r, req)
	case "ping":
		resp := rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}
		_ = json.NewEncoder(w).Encode(resp)
	case "resources/list":
		resp := rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"resources": []any{}}}
		_ = json.NewEncoder(w).Encode(resp)
	case "prompts/list":
		resp := rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"prompts": []any{}}}
		_ = json.NewEncoder(w).Encode(resp)
	default:
		writeError(w, req.ID, codeMethodNotFound, fmt.Sprintf("method not found: %s", req.Method))
	}
}

func (s *Server) handleInitialize(w http.ResponseWriter, req rpcRequest) {
	result := initializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities:    capabilities{Tools: map[string]any{}},
		ServerInfo: serverInfo{
			Name:    "nexusOrchestrator",
			Version: "1.0.0",
			Instructions: "You are connected to nexusOrchestrator — a multi-LLM AI task " +
				"orchestration server. Call 'howto_brief' first if you have a small context " +
				"window (< 64K tokens), or 'howto' for the full guide. " +
				"Use 'register_session' to identify yourself, " +
				"'get_queue' to see available tasks, and 'claim_task' to start working.",
		},
	}
	// Streamable HTTP session management (MCP 2025-03-26+).
	w.Header().Set("Mcp-Session-Id", fmt.Sprintf("nexus-%d", time.Now().UnixNano()))
	resp := rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result}
	_ = json.NewEncoder(w).Encode(resp)
}

// ----- Helpers -----

func writeError(w http.ResponseWriter, id json.RawMessage, code int, msg string) {
	resp := rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: msg},
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// mcpError is a sentinel error that carries an MCP error code so that
// handleToolCall can write the correct JSON-RPC error instead of -32603.
type mcpError struct {
	code int
	msg  string
}

func (e *mcpError) Error() string { return e.msg }
