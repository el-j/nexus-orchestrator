package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/mcp"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// --- Mock orchestrator ---

type mockOrch struct {
	submitID   string
	submitErr  error
	submitTask domain.Task // captures the most recent SubmitTask argument
	getTask    domain.Task
	getErr     error
	queue      []domain.Task
	queueErr   error
	cancelErr  error
	providers  []ports.ProviderInfo
	provErr    error

	createDraftID  string
	createDraftErr error
	backlogTasks   []domain.Task
	backlogErr     error
	promoteErr     error
	updateResult   domain.Task
	updateErr      error

	triggerScanResult  []domain.DiscoveredProvider
	triggerScanErr     error
	promoteProviderErr error
}

func (m *mockOrch) SubmitTask(t domain.Task) (string, error) {
	m.submitTask = t
	return m.submitID, m.submitErr
}
func (m *mockOrch) GetTask(_ string) (domain.Task, error) { return m.getTask, m.getErr }
func (m *mockOrch) GetQueue() ([]domain.Task, error)      { return m.queue, m.queueErr }
func (m *mockOrch) GetAllTasks() ([]domain.Task, error)   { return m.queue, m.queueErr }
func (m *mockOrch) CancelTask(_ string) error             { return m.cancelErr }
func (m *mockOrch) GetProviders() ([]ports.ProviderInfo, error) {
	return m.providers, m.provErr
}
func (m *mockOrch) RegisterCloudProvider(_ domain.ProviderConfig) error { return nil }
func (m *mockOrch) RemoveProvider(_ string) error                       { return nil }
func (m *mockOrch) GetProviderModels(_ string) ([]string, error)        { return nil, nil }
func (m *mockOrch) AddProviderConfig(_ context.Context, cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	return cfg, nil
}
func (m *mockOrch) UpdateProviderConfig(_ context.Context, cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	return cfg, nil
}
func (m *mockOrch) RemoveProviderConfig(_ context.Context, _ string) error { return nil }
func (m *mockOrch) ListProviderConfigs(_ context.Context) ([]domain.ProviderConfig, error) {
	return nil, nil
}
func (m *mockOrch) GetDiscoveredProviders() ([]domain.DiscoveredProvider, error) {
	return nil, nil
}
func (m *mockOrch) TriggerScan(_ context.Context) ([]domain.DiscoveredProvider, error) {
	return m.triggerScanResult, m.triggerScanErr
}
func (m *mockOrch) PromoteProvider(_ context.Context, _ string) error { return m.promoteProviderErr }
func (m *mockOrch) CreateDraft(_ domain.Task) (string, error) {
	return m.createDraftID, m.createDraftErr
}
func (m *mockOrch) GetBacklog(_ string) ([]domain.Task, error) { return m.backlogTasks, m.backlogErr }
func (m *mockOrch) PromoteTask(_ string) error                 { return m.promoteErr }
func (m *mockOrch) UpdateTask(_ string, _ domain.Task) (domain.Task, error) {
	return m.updateResult, m.updateErr
}
func (m *mockOrch) RegisterAISession(_ context.Context, s domain.AISession) (domain.AISession, error) {
	return s, nil
}
func (m *mockOrch) ListAISessions(_ context.Context) ([]domain.AISession, error) { return nil, nil }
func (m *mockOrch) DeregisterAISession(_ context.Context, _ string) error        { return nil }
func (m *mockOrch) HeartbeatAISession(_ context.Context, _ string) error         { return nil }

// --- Helpers ---

type rpcResp struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func postRPC(t *testing.T, srv *httptest.Server, body any) rpcResp {
	t.Helper()
	b, _ := json.Marshal(body)
	resp, err := http.Post(srv.URL+"/mcp", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST /mcp: %v", err)
	}
	defer resp.Body.Close()
	var r rpcResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return r
}

func newServer(t *testing.T, orch *mockOrch) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(mcp.NewServer(orch))
	t.Cleanup(srv.Close)
	return srv
}

// --- Tests ---

func TestMCP_Health(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: want 200, got %d", resp.StatusCode)
	}
}

func TestMCP_Initialize(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
	})
	if r.Error != nil {
		t.Fatalf("expected no error, got %+v", r.Error)
	}
	var result struct {
		ProtocolVersion string `json:"protocolVersion"`
		ServerInfo      struct {
			Name string `json:"name"`
		} `json:"serverInfo"`
	}
	if err := json.Unmarshal(r.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result.ProtocolVersion != "2024-11-05" {
		t.Errorf("protocolVersion: want 2024-11-05, got %q", result.ProtocolVersion)
	}
	if result.ServerInfo.Name != "nexusOrchestrator" {
		t.Errorf("serverInfo.name: want nexusOrchestrator, got %q", result.ServerInfo.Name)
	}
}

func TestMCP_ToolsList_Returns15Tools(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	})
	if r.Error != nil {
		t.Fatalf("expected no error, got %+v", r.Error)
	}
	var result struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(r.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(result.Tools) != 15 {
		t.Errorf("expected 15 tools, got %d", len(result.Tools))
	}
}

func TestMCP_ToolCall_SubmitTask(t *testing.T) {
	orch := &mockOrch{submitID: "task-abc"}
	srv := newServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "submit_task",
			"arguments": map[string]any{
				"projectPath": "/project",
				"targetFile":  "main.go",
				"instruction": "write hello world",
			},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(r.Result, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal text payload: %v", err)
	}
	if payload["id"] != "task-abc" {
		t.Errorf("id: want task-abc, got %q", payload["id"])
	}
}

func TestMCP_ToolCall_GetQueue(t *testing.T) {
	orch := &mockOrch{queue: []domain.Task{{ID: "t1"}, {ID: "t2"}}}
	srv := newServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "tools/call",
		"params":  map[string]any{"name": "get_queue", "arguments": map[string]any{}},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(r.Result, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var tasks []domain.Task
	if err := json.Unmarshal([]byte(result.Content[0].Text), &tasks); err != nil {
		t.Fatalf("unmarshal tasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestMCP_ToolCall_GetAllTasks(t *testing.T) {
	orch := &mockOrch{queue: []domain.Task{{ID: "t1"}, {ID: "t2"}, {ID: "t3"}}}
	srv := newServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      41,
		"method":  "tools/call",
		"params":  map[string]any{"name": "get_all_tasks", "arguments": map[string]any{}},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(r.Result, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var tasks []domain.Task
	if err := json.Unmarshal([]byte(result.Content[0].Text), &tasks); err != nil {
		t.Fatalf("unmarshal tasks: %v", err)
	}
	if len(tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(tasks))
	}
}

func TestMCP_ToolCall_CancelTask(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      5,
		"method":  "tools/call",
		"params":  map[string]any{"name": "cancel_task", "arguments": map[string]any{"id": "t1"}},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
}

func TestMCP_ToolCall_Health(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      6,
		"method":  "tools/call",
		"params":  map[string]any{"name": "health", "arguments": map[string]any{}},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
}

func TestMCP_UnknownMethod_ReturnsMethodNotFound(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      7,
		"method":  "no/such/method",
	})
	if r.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if r.Error.Code != -32601 {
		t.Errorf("code: want -32601, got %d", r.Error.Code)
	}
}

func TestMCP_InvalidJSON_ReturnsParseError(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	resp, err := http.Post(srv.URL+"/mcp", "application/json", bytes.NewReader([]byte("not-json")))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	var r rpcResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if r.Error == nil || r.Error.Code != -32700 {
		t.Errorf("expected parse error (-32700), got %+v", r.Error)
	}
}

func TestMCP_SubmitTask_PropagatesOrchestratorError(t *testing.T) {
	orch := &mockOrch{submitErr: errors.New("queue full")}
	srv := newServer(t, orch)
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      8,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "submit_task",
			"arguments": map[string]any{"projectPath": "/p", "targetFile": "f.go", "instruction": "x"},
		},
	})
	if r.Error == nil {
		t.Fatal("expected error from orchestrator")
	}
	if r.Error.Code != -32603 {
		t.Errorf("code: want -32603, got %d", r.Error.Code)
	}
}

func TestMCP_SubmitTask_WithCommand(t *testing.T) {
	orch := &mockOrch{submitID: "task-cmd"}
	srv := newServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      9,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "submit_task",
			"arguments": map[string]any{
				"projectPath": "/project",
				"targetFile":  "main.go",
				"instruction": "plan this",
				"command":     "plan",
			},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}

	// Verify the command was passed through to the orchestrator
	if orch.submitTask.Command != domain.CommandPlan {
		t.Errorf("expected command %q, got %q", domain.CommandPlan, orch.submitTask.Command)
	}
}

func TestMCP_SubmitTask_ErrNoPlan_PropagatesError(t *testing.T) {
	orch := &mockOrch{submitErr: domain.ErrNoPlan}
	srv := newServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      10,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "submit_task",
			"arguments": map[string]any{
				"projectPath": "/project",
				"targetFile":  "main.go",
				"instruction": "execute now",
				"command":     "execute",
			},
		},
	})
	if r.Error == nil {
		t.Fatal("expected error for ErrNoPlan")
	}
	if r.Error.Code != -32603 {
		t.Errorf("code: want -32603, got %d", r.Error.Code)
	}
}

// --- Backlog lifecycle MCP tests ---------------------------------------------

func TestMCP_CreateDraft_ReturnsTaskID(t *testing.T) {
	orch := &mockOrch{createDraftID: "draft-abc"}
	srv := newServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      20,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "create_draft",
			"arguments": map[string]any{
				"projectPath": "/proj/mcp",
				"instruction": "build a thing",
			},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(r.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal text payload: %v", err)
	}
	if payload["id"] != "draft-abc" {
		t.Errorf("id: want draft-abc, got %q", payload["id"])
	}
	if payload["status"] != string(domain.StatusDraft) {
		t.Errorf("status: want DRAFT, got %q", payload["status"])
	}
}

func TestMCP_GetBacklog_ReturnsTasks(t *testing.T) {
	orch := &mockOrch{
		backlogTasks: []domain.Task{
			{ID: "d1", Status: domain.StatusDraft},
			{ID: "b1", Status: domain.StatusBacklog},
		},
	}
	srv := newServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      21,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "get_backlog",
			"arguments": map[string]any{"projectPath": "/proj/mcp"},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(r.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	var tasks []domain.Task
	if err := json.Unmarshal([]byte(result.Content[0].Text), &tasks); err != nil {
		t.Fatalf("unmarshal tasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("want 2 tasks, got %d", len(tasks))
	}
}

func TestMCP_PromoteTask_ReturnsBool(t *testing.T) {
	srv := newServer(t, &mockOrch{})

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      22,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "promote_task",
			"arguments": map[string]any{"id": "task-to-promote"},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(r.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	var payload map[string]bool
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if !payload["promoted"] {
		t.Error("promoted: want true, got false")
	}
}

// --- Provider discovery MCP tests -------------------------------------------

func TestMCP_DiscoverProviders_ReturnsResults(t *testing.T) {
	orch := &mockOrch{}
	orch.triggerScanResult = []domain.DiscoveredProvider{
		{ID: "dp-1", Name: "LM Studio", Kind: domain.ProviderKindLMStudio},
	}
	srv := newServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      30,
		"method":  "tools/call",
		"params":  map[string]any{"name": "discover_providers", "arguments": map[string]any{}},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(r.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	var providers []domain.DiscoveredProvider
	if err := json.Unmarshal([]byte(result.Content[0].Text), &providers); err != nil {
		t.Fatalf("unmarshal providers: %v", err)
	}
	if len(providers) != 1 || providers[0].ID != "dp-1" {
		t.Errorf("unexpected providers: %+v", providers)
	}
}

func TestMCP_PromoteProvider_Success(t *testing.T) {
	srv := newServer(t, &mockOrch{})

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      31,
		"method":  "tools/call",
		"params":  map[string]any{"name": "promote_provider", "arguments": map[string]any{"id": "dp-1"}},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(r.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload["promoted"] != true {
		t.Errorf("promoted: want true, got %v", payload["promoted"])
	}
	if payload["id"] != "dp-1" {
		t.Errorf("id: want dp-1, got %v", payload["id"])
	}
}

func TestMCP_PromoteProvider_NotFound_Returns32602(t *testing.T) {
	orch := &mockOrch{promoteProviderErr: fmt.Errorf("not found: %w", domain.ErrNotFound)}
	srv := newServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      32,
		"method":  "tools/call",
		"params":  map[string]any{"name": "promote_provider", "arguments": map[string]any{"id": "missing"}},
	})
	if r.Error == nil {
		t.Fatal("expected error for not found")
	}
	if r.Error.Code != -32602 {
		t.Errorf("code: want -32602, got %d", r.Error.Code)
	}
}

// --- AI session MCP tests ----------------------------------------------------

func TestMCP_RegisterSession_ReturnsSessionID(t *testing.T) {
	srv := newServer(t, &mockOrch{})

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      40,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "register_session",
			"arguments": map[string]any{
				"agent_name":   "Claude Desktop",
				"project_path": "/my/project",
			},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(r.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal text payload: %v", err)
	}
	if _, ok := payload["session_id"]; !ok {
		t.Error("result must contain session_id key")
	}
	if payload["status"] != "registered" {
		t.Errorf("status: want \"registered\", got %v", payload["status"])
	}
}

func TestMCP_RegisterSession_MissingAgentName_Returns32602(t *testing.T) {
	srv := newServer(t, &mockOrch{})

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      41,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "register_session",
			"arguments": map[string]any{},
		},
	})
	if r.Error == nil {
		t.Fatal("expected error for missing agent_name")
	}
	if r.Error.Code != -32602 {
		t.Errorf("code: want -32602, got %d", r.Error.Code)
	}
}

func TestMCP_GetAISessions_ReturnsList(t *testing.T) {
	srv := newServer(t, &mockOrch{})

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      42,
		"method":  "tools/call",
		"params":  map[string]any{"name": "get_ai_sessions", "arguments": map[string]any{}},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(r.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}
	// The mock returns nil sessions; the marshalled form is a JSON null or empty array.
	// Either way the response must be valid JSON.
	var sessions []domain.AISession
	if err := json.Unmarshal([]byte(result.Content[0].Text), &sessions); err != nil {
		t.Fatalf("result text is not a valid JSON session list: %v", err)
	}
}
