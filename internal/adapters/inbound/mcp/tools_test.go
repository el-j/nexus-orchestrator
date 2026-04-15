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
