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

	"nexus-orchestrator/internal/core/domain"
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
}

type capabilities struct {
	Tools map[string]any `json:"tools"`
}

type initializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    capabilities `json:"capabilities"`
	ServerInfo      serverInfo   `json:"serverInfo"`
}

type property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
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
		s.handleToolCall(w, req)
	default:
		writeError(w, req.ID, codeMethodNotFound, fmt.Sprintf("method not found: %s", req.Method))
	}
}

func (s *Server) handleInitialize(w http.ResponseWriter, req rpcRequest) {
	result := initializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities:    capabilities{Tools: map[string]any{}},
		ServerInfo:      serverInfo{Name: "nexusOrchestrator", Version: "1.0.0"},
	}
	resp := rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result}
	_ = json.NewEncoder(w).Encode(resp)
}

// ----- Tool dispatch -----

type callToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

func (s *Server) handleToolCall(w http.ResponseWriter, req rpcRequest) {
	var p callToolParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		writeError(w, req.ID, codeInvalidParams, "invalid params")
		return
	}

	var (
		result callToolResult
		err    error
	)

	switch p.Name {
	case "submit_task":
		result, err = s.toolSubmitTask(p.Arguments)
	case "get_task":
		result, err = s.toolGetTask(p.Arguments)
	case "get_queue":
		result, err = s.toolGetQueue()
	case "cancel_task":
		result, err = s.toolCancelTask(p.Arguments)
	case "get_providers":
		result, err = s.toolGetProviders()
	case "health":
		result, err = s.toolHealth()
	default:
		writeError(w, req.ID, codeMethodNotFound, fmt.Sprintf("unknown tool: %s", p.Name))
		return
	}

	if err != nil {
		writeError(w, req.ID, codeInternalError, err.Error())
		return
	}

	resp := rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result}
	_ = json.NewEncoder(w).Encode(resp)
}

// ----- Individual tool handlers -----

func (s *Server) toolSubmitTask(args json.RawMessage) (callToolResult, error) {
	var p struct {
		ProjectPath  string   `json:"projectPath"`
		TargetFile   string   `json:"targetFile"`
		Instruction  string   `json:"instruction"`
		ContextFiles []string `json:"contextFiles"`
		Command      string   `json:"command"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: submit_task: invalid arguments: %w", err)
	}
	t := domain.Task{
		ProjectPath:  p.ProjectPath,
		TargetFile:   p.TargetFile,
		Instruction:  p.Instruction,
		ContextFiles: p.ContextFiles,
		Command:      domain.CommandType(p.Command),
	}
	id, err := s.orch.SubmitTask(t)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: submit_task: %w", err)
	}
	b, _ := json.Marshal(map[string]string{"id": id})
	return textResult(string(b)), nil
}

func (s *Server) toolGetTask(args json.RawMessage) (callToolResult, error) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_task: invalid arguments: %w", err)
	}
	task, err := s.orch.GetTask(p.ID)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_task: %w", err)
	}
	b, err := json.Marshal(task)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_task: marshal: %w", err)
	}
	return textResult(string(b)), nil
}

func (s *Server) toolGetQueue() (callToolResult, error) {
	tasks, err := s.orch.GetQueue()
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_queue: %w", err)
	}
	b, err := json.Marshal(tasks)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_queue: marshal: %w", err)
	}
	return textResult(string(b)), nil
}

func (s *Server) toolCancelTask(args json.RawMessage) (callToolResult, error) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: cancel_task: invalid arguments: %w", err)
	}
	if err := s.orch.CancelTask(p.ID); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: cancel_task: %w", err)
	}
	b, _ := json.Marshal(map[string]bool{"cancelled": true})
	return textResult(string(b)), nil
}

func (s *Server) toolGetProviders() (callToolResult, error) {
	providers, err := s.orch.GetProviders()
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_providers: %w", err)
	}
	b, err := json.Marshal(providers)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_providers: marshal: %w", err)
	}
	return textResult(string(b)), nil
}

func (s *Server) toolHealth() (callToolResult, error) {
	b, _ := json.Marshal(map[string]string{"status": "ok"})
	return textResult(string(b)), nil
}

// ----- Helpers -----

func textResult(text string) callToolResult {
	return callToolResult{Content: []contentItem{{Type: "text", Text: text}}}
}

func writeError(w http.ResponseWriter, id json.RawMessage, code int, msg string) {
	resp := rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: msg},
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func toolList() []toolDef {
	return []toolDef{
		{
			Name:        "submit_task",
			Description: "Submit a new code-generation task to the orchestrator.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"projectPath":  {Type: "string", Description: "Absolute path to the project root."},
					"targetFile":   {Type: "string", Description: "Relative path of the file to generate or modify."},
					"instruction":  {Type: "string", Description: "Natural-language instruction for the LLM."},
					"contextFiles": {Type: "array", Description: "Optional list of relative file paths to include as context."},
					"command":      {Type: "string", Description: "Task type: plan, execute, or auto (default: auto)."},
				},
				Required: []string{"projectPath", "targetFile", "instruction"},
			},
		},
		{
			Name:        "get_task",
			Description: "Get the current status and output of a task by ID.",
			InputSchema: inputSchema{
				Type:       "object",
				Properties: map[string]property{"id": {Type: "string", Description: "Task ID returned by submit_task."}},
				Required:   []string{"id"},
			},
		},
		{
			Name:        "get_queue",
			Description: "List all tasks currently in the queue.",
			InputSchema: inputSchema{Type: "object", Properties: map[string]property{}},
		},
		{
			Name:        "cancel_task",
			Description: "Cancel a pending task by ID.",
			InputSchema: inputSchema{
				Type:       "object",
				Properties: map[string]property{"id": {Type: "string", Description: "Task ID to cancel."}},
				Required:   []string{"id"},
			},
		},
		{
			Name:        "get_providers",
			Description: "List available LLM providers and their models.",
			InputSchema: inputSchema{Type: "object", Properties: map[string]property{}},
		},
		{
			Name:        "health",
			Description: "Check that the nexusOrchestrator daemon is reachable.",
			InputSchema: inputSchema{Type: "object", Properties: map[string]property{}},
		},
	}
}
