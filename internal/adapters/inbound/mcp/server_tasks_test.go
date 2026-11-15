package mcp_test

import (
	"encoding/json"
	"errors"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

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
	var payload map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if promoted, _ := payload["promoted"].(bool); !promoted {
		t.Error("promoted: want true, got false")
	}
}
