// Package mcp provides a JSON-RPC 2.0 Model Context Protocol server as an
// inbound adapter. It exposes task management tools compatible with Claude
// Desktop and other MCP clients.
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
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
	orch ports.Orchestrator
	mux  *http.ServeMux
}

// NewServer creates a Server and registers its HTTP handlers.
func NewServer(orch ports.Orchestrator) *Server {
	s := &Server{
		orch: orch,
		mux:  http.NewServeMux(),
	}
	s.mux.HandleFunc("/mcp", s.handleRPC)
	s.mux.HandleFunc("/health", s.handleHealth)
	return s
}

// ServeHTTP implements http.Handler so *Server can be passed to httptest.NewServer.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// StartMCPServer runs an HTTP server serving the MCP JSON-RPC 2.0 endpoint.
// It blocks until ctx is cancelled, then shuts down gracefully.
func StartMCPServer(ctx context.Context, orch ports.Orchestrator, addr string) error {
	srv := &http.Server{
		Addr:         addr,
		Handler:      NewServer(orch).mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
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
				"orchestration server. Call the 'howto' tool first to receive a complete " +
				"integration guide. Use 'register_session' to identify yourself, " +
				"'get_queue' to see available tasks, and 'claim_task' to start working.",
		},
	}
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
