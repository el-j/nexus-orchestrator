package mcp_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/mcp"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

type toolHarnessOrch struct {
	mockOrch
	promoteResult       ports.PromoteResult
	registeredSession   domain.AISession
	registerErr         error
	claimResult         domain.Task
	claimErr            error
	updateStatusResult  domain.Task
	updateStatusErr     error
	discoveredPlans     []domain.DiscoveredPlanFile
	discoveredPlansPath string
	delegateInstruction string
	delegateErr         error

	// provider config stubs
	providerConfigs   []domain.ProviderConfig
	addProviderResult domain.ProviderConfig
	addProviderErr    error
	updProviderResult domain.ProviderConfig
	updProviderErr    error
	rmProviderErr     error
	listProviderErr   error

	// session lifecycle stubs
	heartbeatSessionErr  error
	deregisterSessionErr error
	purgedCount          int
	purgeErr             error

	// discovery stubs
	discoveredAgents []domain.DiscoveredAgent
	agentsErr        error
}

func (m *toolHarnessOrch) PromoteTask(_ string) (ports.PromoteResult, error) {
	if m.promoteResult.Warning != "" || m.promoteResult.Promoted {
		return m.promoteResult, m.promoteErr
	}
	return ports.PromoteResult{Promoted: m.promoteErr == nil}, m.promoteErr
}

func (m *toolHarnessOrch) RegisterAISession(_ context.Context, s domain.AISession) (domain.AISession, error) {
	if m.registerErr != nil {
		return domain.AISession{}, m.registerErr
	}
	if m.registeredSession.ID == "" {
		m.registeredSession = s
		m.registeredSession.ID = "session-1"
	}
	return m.registeredSession, nil
}

func (m *toolHarnessOrch) ClaimTask(_ context.Context, _, _ string) (domain.Task, error) {
	return m.claimResult, m.claimErr
}

func (m *toolHarnessOrch) UpdateTaskStatus(_ context.Context, _, _ string, _ domain.TaskStatus, _ string) (domain.Task, error) {
	return m.updateStatusResult, m.updateStatusErr
}

func (m *toolHarnessOrch) GetDiscoveredPlanFiles(_ context.Context, projectPath string) ([]domain.DiscoveredPlanFile, error) {
	m.discoveredPlansPath = projectPath
	return m.discoveredPlans, nil
}

func (m *toolHarnessOrch) DelegateToNexus(_ context.Context, _ string) (string, error) {
	return m.delegateInstruction, m.delegateErr
}

func (m *toolHarnessOrch) GetTasksBySessionID(_ string) ([]domain.Task, error) {
	return nil, nil
}

func (m *toolHarnessOrch) AddProviderConfig(_ context.Context, _ domain.ProviderConfig) (domain.ProviderConfig, error) {
	return m.addProviderResult, m.addProviderErr
}

func (m *toolHarnessOrch) UpdateProviderConfig(_ context.Context, _ domain.ProviderConfig) (domain.ProviderConfig, error) {
	return m.updProviderResult, m.updProviderErr
}

func (m *toolHarnessOrch) RemoveProviderConfig(_ context.Context, _ string) error {
	return m.rmProviderErr
}

func (m *toolHarnessOrch) ListProviderConfigs(_ context.Context) ([]domain.ProviderConfig, error) {
	return m.providerConfigs, m.listProviderErr
}

func (m *toolHarnessOrch) HeartbeatAISession(_ context.Context, _ string) error {
	return m.heartbeatSessionErr
}

func (m *toolHarnessOrch) DeregisterAISession(_ context.Context, _ string) error {
	return m.deregisterSessionErr
}

func (m *toolHarnessOrch) PurgeDisconnectedSessions(_ context.Context) (int, error) {
	return m.purgedCount, m.purgeErr
}

func (m *toolHarnessOrch) GetDiscoveredAgents(_ context.Context) ([]domain.DiscoveredAgent, error) {
	return m.discoveredAgents, m.agentsErr
}

func decodeToolText(t *testing.T, raw json.RawMessage, dest any) {
	t.Helper()
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal tool result envelope: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected tool result content")
	}
	if err := json.Unmarshal([]byte(result.Content[0].Text), dest); err != nil {
		t.Fatalf("unmarshal tool text payload: %v", err)
	}
}

func newToolServer(t *testing.T, orch *toolHarnessOrch) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(mcp.NewMcpServer(orch, nil))
	t.Cleanup(srv.Close)
	return srv
}

func TestToolPromoteTask_ReturnsWarningPayload(t *testing.T) {
	orch := &toolHarnessOrch{promoteResult: ports.PromoteResult{Promoted: true, Warning: "no active provider configured"}}
	srv := newToolServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "promote_task",
			"arguments": map[string]any{
				"id": "draft-1",
			},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var payload map[string]any
	decodeToolText(t, r.Result, &payload)
	if payload["promoted"] != true {
		t.Fatalf("expected promoted=true, got %#v", payload["promoted"])
	}
	if payload["warning"] != "no active provider configured" {
		t.Fatalf("unexpected warning payload: %#v", payload)
	}
}

func TestToolRegisterSession_RequiresAgentName(t *testing.T) {
	srv := newToolServer(t, &toolHarnessOrch{})

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "register_session",
			"arguments": map[string]any{},
		},
	})
	if r.Error == nil {
		t.Fatal("expected invalid params error")
	}
	if r.Error.Code != -32602 {
		t.Fatalf("expected invalid params code, got %+v", r.Error)
	}
	if !strings.Contains(r.Error.Message, "agent_name is required") {
		t.Fatalf("unexpected error message: %+v", r.Error)
	}
}

func TestToolClaimTask_RequiresTaskAndSessionID(t *testing.T) {
	srv := newToolServer(t, &toolHarnessOrch{})

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "claim_task",
			"arguments": map[string]any{"task_id": ""},
		},
	})
	if r.Error == nil {
		t.Fatal("expected invalid params error")
	}
	if r.Error.Code != -32602 {
		t.Fatalf("expected invalid params code, got %+v", r.Error)
	}
}

func TestToolUpdateTaskStatus_RejectsUnsupportedStatus(t *testing.T) {
	srv := newToolServer(t, &toolHarnessOrch{})

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "update_task_status",
			"arguments": map[string]any{
				"task_id":    "task-1",
				"session_id": "session-1",
				"status":     "PROCESSING",
			},
		},
	})
	if r.Error == nil {
		t.Fatal("expected invalid params error")
	}
	if !strings.Contains(r.Error.Message, "status must be COMPLETED or FAILED") {
		t.Fatalf("unexpected error message: %+v", r.Error)
	}
}

func TestToolGetDiscoveredPlans_PassesProjectPath(t *testing.T) {
	orch := &toolHarnessOrch{discoveredPlans: []domain.DiscoveredPlanFile{{
		ID:          "plan-1",
		Path:        "/repo/.claude/plans/PLAN-1.md",
		Kind:        domain.PlanFileKindMarkdown,
		Format:      "markdown",
		ProjectPath: "/repo",
		IsActive:    true,
	}}}
	srv := newToolServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      5,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "get_discovered_plans",
			"arguments": map[string]any{
				"projectPath": "/repo",
			},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	if orch.discoveredPlansPath != "/repo" {
		t.Fatalf("expected projectPath to be forwarded, got %q", orch.discoveredPlansPath)
	}
	var payload struct {
		ProjectPath string                      `json:"projectPath"`
		FileCount   int                         `json:"fileCount"`
		ByKind      map[string]int              `json:"byKind"`
		ActiveTool  string                      `json:"activeTool"`
		Files       []domain.DiscoveredPlanFile `json:"files"`
	}
	decodeToolText(t, r.Result, &payload)
	if payload.ProjectPath != "/repo" {
		t.Fatalf("expected projectPath=/repo, got %q", payload.ProjectPath)
	}
	if payload.FileCount != 1 {
		t.Fatalf("expected fileCount=1, got %d", payload.FileCount)
	}
	if payload.ByKind["markdown"] != 1 {
		t.Fatalf("expected byKind[markdown]=1, got %v", payload.ByKind)
	}
	if payload.ActiveTool != "markdown" {
		t.Fatalf("expected activeTool=markdown, got %q", payload.ActiveTool)
	}
	if len(payload.Files) != 1 || payload.Files[0].ID != "plan-1" {
		t.Fatalf("unexpected files payload: %+v", payload.Files)
	}
}

func TestToolGetDiscoveredPlans_NexusTakesPriority(t *testing.T) {
	orch := &toolHarnessOrch{discoveredPlans: []domain.DiscoveredPlanFile{
		{ID: "n1", Kind: domain.PlanFileKindNexus, Format: "json"},
		{ID: "m1", Kind: domain.PlanFileKindMarkdown, Format: "md"},
		{ID: "m2", Kind: domain.PlanFileKindMarkdown, Format: "md"},
	}}
	srv := newToolServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      50,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "get_discovered_plans",
			"arguments": map[string]any{"projectPath": "/repo"},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var payload struct {
		ActiveTool string         `json:"activeTool"`
		FileCount  int            `json:"fileCount"`
		ByKind     map[string]int `json:"byKind"`
	}
	decodeToolText(t, r.Result, &payload)
	if payload.ActiveTool != "nexus" {
		t.Fatalf("expected activeTool=nexus (priority), got %q", payload.ActiveTool)
	}
	if payload.FileCount != 3 {
		t.Fatalf("expected fileCount=3, got %d", payload.FileCount)
	}
}

func TestToolGetDiscoveredPlans_EmptyReturnsUnknown(t *testing.T) {
	orch := &toolHarnessOrch{discoveredPlans: []domain.DiscoveredPlanFile{}}
	srv := newToolServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      51,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "get_discovered_plans",
			"arguments": map[string]any{"projectPath": "/empty"},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var payload struct {
		ActiveTool string `json:"activeTool"`
		FileCount  int    `json:"fileCount"`
	}
	decodeToolText(t, r.Result, &payload)
	if payload.ActiveTool != "unknown" {
		t.Fatalf("expected activeTool=unknown for empty result, got %q", payload.ActiveTool)
	}
	if payload.FileCount != 0 {
		t.Fatalf("expected fileCount=0, got %d", payload.FileCount)
	}
}

// TestToolGetQueue_WithProjectPath verifies that get_queue scopes results to the
// supplied projectPath, preventing agents from different projects seeing each other's work.
func TestToolGetQueue_WithProjectPath(t *testing.T) {
	orch := &toolHarnessOrch{}
	orch.queue = []domain.Task{
		{ID: "ta1", ProjectPath: "/proj/a", Status: domain.StatusQueued},
		{ID: "tb1", ProjectPath: "/proj/b", Status: domain.StatusQueued},
		{ID: "ta2", ProjectPath: "/proj/a", Status: domain.StatusProcessing},
	}
	srv := newToolServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0", "id": 10,
		"method": "tools/call",
		"params": map[string]any{
			"name":      "get_queue",
			"arguments": map[string]any{"projectPath": "/proj/a"},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var tasks []domain.Task
	decodeToolText(t, r.Result, &tasks)
	if len(tasks) != 2 {
		t.Fatalf("want 2 tasks for /proj/a, got %d", len(tasks))
	}
	for _, task := range tasks {
		if task.ProjectPath != "/proj/a" {
			t.Errorf("got task from wrong project: %s (id=%s)", task.ProjectPath, task.ID)
		}
	}
}

// TestToolGetQueue_NoProjectPath verifies that get_queue without projectPath returns all tasks
// (admin/global view — allowed but caller is responsible for not confusing projects).
func TestToolGetQueue_NoProjectPath(t *testing.T) {
	orch := &toolHarnessOrch{}
	orch.queue = []domain.Task{
		{ID: "ta1", ProjectPath: "/proj/a", Status: domain.StatusQueued},
		{ID: "tb1", ProjectPath: "/proj/b", Status: domain.StatusQueued},
	}
	srv := newToolServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0", "id": 11,
		"method": "tools/call",
		"params": map[string]any{
			"name":      "get_queue",
			"arguments": map[string]any{},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var tasks []domain.Task
	decodeToolText(t, r.Result, &tasks)
	if len(tasks) != 2 {
		t.Fatalf("want 2 tasks (all projects), got %d", len(tasks))
	}
}

// TestToolGetAllTasks_WithProjectPath verifies that get_all_tasks filters by project.
func TestToolGetAllTasks_WithProjectPath(t *testing.T) {
	orch := &toolHarnessOrch{}
	orch.queue = []domain.Task{
		{ID: "ta1", ProjectPath: "/proj/a", Status: domain.StatusCompleted},
		{ID: "tb1", ProjectPath: "/proj/b", Status: domain.StatusCompleted},
		{ID: "ta2", ProjectPath: "/proj/a", Status: domain.StatusFailed},
	}
	srv := newToolServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0", "id": 12,
		"method": "tools/call",
		"params": map[string]any{
			"name":      "get_all_tasks",
			"arguments": map[string]any{"projectPath": "/proj/b"},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var tasks []domain.Task
	decodeToolText(t, r.Result, &tasks)
	if len(tasks) != 1 {
		t.Fatalf("want 1 task for /proj/b, got %d", len(tasks))
	}
	if tasks[0].ID != "tb1" {
		t.Errorf("expected task tb1, got %s", tasks[0].ID)
	}
}

func TestToolListProviderConfigs_ReturnsAll(t *testing.T) {
	orch := &toolHarnessOrch{
		providerConfigs: []domain.ProviderConfig{
			{ID: "cfg-1", Name: "lmstudio", Kind: domain.ProviderKindLMStudio},
			{ID: "cfg-2", Name: "ollama", Kind: domain.ProviderKindOllama},
		},
	}
	srv := newToolServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0", "id": 10, "method": "tools/call",
		"params": map[string]any{"name": "list_provider_configs", "arguments": map[string]any{}},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var cfgs []domain.ProviderConfig
	decodeToolText(t, r.Result, &cfgs)
	if len(cfgs) != 2 {
		t.Errorf("expected 2 configs, got %d", len(cfgs))
	}
}

func TestToolAddProviderConfig_RequiresKindAndName(t *testing.T) {
	orch := &toolHarnessOrch{}
	srv := newToolServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0", "id": 11, "method": "tools/call",
		"params": map[string]any{
			"name":      "add_provider_config",
			"arguments": map[string]any{"base_url": "http://localhost:1234"},
		},
	})
	if r.Error == nil {
		t.Fatal("expected -32602 error for missing kind/name")
	}
	if r.Error.Code != -32602 {
		t.Errorf("expected code -32602, got %d", r.Error.Code)
	}
}

func TestToolUpdateProviderConfig_RequiresID(t *testing.T) {
	orch := &toolHarnessOrch{}
	srv := newToolServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0", "id": 12, "method": "tools/call",
		"params": map[string]any{
			"name":      "update_provider_config",
			"arguments": map[string]any{"name": "new-name"},
		},
	})
	if r.Error == nil {
		t.Fatal("expected -32602 error for missing id")
	}
	if r.Error.Code != -32602 {
		t.Errorf("expected code -32602, got %d", r.Error.Code)
	}
}

func TestToolRemoveProviderConfig_ForwardsID(t *testing.T) {
	orch := &toolHarnessOrch{}
	srv := newToolServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0", "id": 13, "method": "tools/call",
		"params": map[string]any{
			"name":      "remove_provider_config",
			"arguments": map[string]any{"id": "cfg-1"},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
}

func TestToolHeartbeatAISession_RequiresSessionID(t *testing.T) {
	orch := &toolHarnessOrch{}
	srv := newToolServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0", "id": 14, "method": "tools/call",
		"params": map[string]any{
			"name":      "heartbeat_ai_session",
			"arguments": map[string]any{},
		},
	})
	if r.Error == nil {
		t.Fatal("expected -32602 error for missing session_id")
	}
	if r.Error.Code != -32602 {
		t.Errorf("expected code -32602, got %d", r.Error.Code)
	}
}

func TestToolDeregisterAISession_RequiresSessionID(t *testing.T) {
	orch := &toolHarnessOrch{}
	srv := newToolServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0", "id": 15, "method": "tools/call",
		"params": map[string]any{
			"name":      "deregister_ai_session",
			"arguments": map[string]any{},
		},
	})
	if r.Error == nil {
		t.Fatal("expected -32602 error for missing session_id")
	}
	if r.Error.Code != -32602 {
		t.Errorf("expected code -32602, got %d", r.Error.Code)
	}
}

func TestToolPurgeDisconnectedSessions_ReturnsCount(t *testing.T) {
	orch := &toolHarnessOrch{purgedCount: 3}
	srv := newToolServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0", "id": 16, "method": "tools/call",
		"params": map[string]any{"name": "purge_disconnected_sessions", "arguments": map[string]any{}},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var payload map[string]any
	decodeToolText(t, r.Result, &payload)
	if int(payload["purged"].(float64)) != 3 {
		t.Errorf("expected purged=3, got %v", payload["purged"])
	}
}

func TestToolGetDiscoveredAgents_ReturnsAgents(t *testing.T) {
	orch := &toolHarnessOrch{
		discoveredAgents: []domain.DiscoveredAgent{
			{ID: "agent-1", Name: "Claude CLI", Kind: "claude-cli"},
		},
	}
	srv := newToolServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0", "id": 17, "method": "tools/call",
		"params": map[string]any{"name": "get_discovered_agents", "arguments": map[string]any{}},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var agents []domain.DiscoveredAgent
	decodeToolText(t, r.Result, &agents)
	if len(agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(agents))
	}
}

func TestToolHowto_ContainsRegisterSessionStep(t *testing.T) {
	orch := &toolHarnessOrch{}
	srv := newToolServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0", "id": 18, "method": "tools/call",
		"params": map[string]any{"name": "howto", "arguments": map[string]any{}},
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
		t.Fatalf("unmarshal howto result: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty howto response")
	}
	text := result.Content[0].Text
	if !strings.Contains(text, "register_session") {
		t.Errorf("howto response should contain 'register_session', got: %s", text[:min(200, len(text))])
	}
}

func TestToolHowtoBrief_ContainsKeywords(t *testing.T) {
	orch := &toolHarnessOrch{}
	srv := newToolServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0", "id": 19, "method": "tools/call",
		"params": map[string]any{"name": "howto_brief", "arguments": map[string]any{}},
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
		t.Fatalf("unmarshal howto_brief result: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty howto_brief response")
	}
	text := result.Content[0].Text
	if !strings.Contains(text, "claim_task") {
		t.Errorf("howto_brief should contain 'claim_task', got: %s", text[:min(200, len(text))])
	}
}

func TestToolDelegateToNexus_ReturnsInstruction(t *testing.T) {
	orch := &toolHarnessOrch{delegateInstruction: "claim task and report back"}
	srv := newToolServer(t, orch)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      6,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "delegate_to_nexus",
			"arguments": map[string]any{
				"session_id": "session-1",
			},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var payload map[string]string
	decodeToolText(t, r.Result, &payload)
	if payload["instruction"] != "claim task and report back" {
		t.Fatalf("unexpected delegate payload: %+v", payload)
	}
}
