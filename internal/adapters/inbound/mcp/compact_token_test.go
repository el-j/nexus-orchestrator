package mcp_test

import (
	"encoding/json"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

func TestMCP_CompactToken_GetTask(t *testing.T) {
	longLogs := strings.Repeat("A", 100)
	orch := &mockOrch{
		getTask: domain.Task{
			ID:          "task-123",
			Status:      domain.StatusCompleted,
			TargetFile:  "pkg/runner/runner.go",
			Instruction: "This is a detailed instruction for the task that should be preserved or summarized.",
			Role:        "coder",
			Priority:    1,
			Command:     domain.CommandExecute,
			Logs:        longLogs,
		},
	}
	srv := newServer(t, orch)
	defer srv.Close()

	// 1. Compact mode
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "get_task",
			"arguments": map[string]any{
				"id":      "task-123",
				"compact": true,
			},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var res struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	_ = json.Unmarshal(r.Result, &res)
	if len(res.Content) == 0 {
		t.Fatal("empty content")
	}
	var sum map[string]any
	_ = json.Unmarshal([]byte(res.Content[0].Text), &sum)
	if sum["id"] != "task-123" || sum["status"] != "COMPLETED" || sum["hasLogs"] != true {
		t.Errorf("unexpected summary: %+v", sum)
	}
	if _, hasFullLogs := sum["logs"]; hasFullLogs {
		t.Errorf("compact summary should not contain full logs: %+v", sum)
	}

	// 2. Truncate logs
	r2 := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "get_task",
			"arguments": map[string]any{
				"id":           "task-123",
				"truncateLogs": 20,
			},
		},
	})
	if r2.Error != nil {
		t.Fatalf("unexpected error: %+v", r2.Error)
	}
	var res2 struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	_ = json.Unmarshal(r2.Result, &res2)
	var task2 domain.Task
	_ = json.Unmarshal([]byte(res2.Content[0].Text), &task2)
	if !strings.Contains(task2.Logs, "[truncated 80 chars for token economy]") {
		t.Errorf("logs were not truncated: %q", task2.Logs)
	}
}

func TestMCP_CompactToken_GetQueue(t *testing.T) {
	orch := &mockOrch{
		queue: []domain.Task{
			{ID: "q1", Status: domain.StatusQueued, Instruction: "Task 1", Logs: "huge log 1"},
			{ID: "q2", Status: domain.StatusProcessing, Instruction: "Task 2", Logs: "huge log 2"},
			{ID: "q3", Status: domain.StatusQueued, Instruction: "Task 3", Logs: "huge log 3"},
		},
	}
	srv := newServer(t, orch)
	defer srv.Close()

	// Compact mode with limit 2
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "get_queue",
			"arguments": map[string]any{
				"compact": true,
				"limit":   2,
			},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var res struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	_ = json.Unmarshal(r.Result, &res)
	var items []map[string]any
	_ = json.Unmarshal([]byte(res.Content[0].Text), &items)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0]["id"] != "q1" || items[1]["id"] != "q2" {
		t.Errorf("unexpected items: %+v", items)
	}
	if _, hasLogs := items[0]["logs"]; hasLogs {
		t.Errorf("compact items must omit full logs: %+v", items[0])
	}
}

func TestMCP_CompactToken_GetAllTasks(t *testing.T) {
	orch := &mockOrch{
		queue: []domain.Task{
			{ID: "a1", Status: domain.StatusCompleted, Instruction: "Task A"},
			{ID: "a2", Status: domain.StatusFailed, Instruction: "Task B"},
		},
	}
	srv := newServer(t, orch)
	defer srv.Close()

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "get_all_tasks",
			"arguments": map[string]any{
				"compact": true,
				"limit":   1,
			},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var res struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	_ = json.Unmarshal(r.Result, &res)
	var items []map[string]any
	_ = json.Unmarshal([]byte(res.Content[0].Text), &items)
	if len(items) != 1 || items[0]["id"] != "a1" {
		t.Fatalf("expected 1 item with id a1, got %+v", items)
	}
}

func TestMCP_CompactToken_GetBacklog(t *testing.T) {
	orch := &mockOrch{
		backlogTasks: []domain.Task{
			{ID: "b1", Status: domain.StatusDraft, Instruction: "Draft 1"},
			{ID: "b2", Status: domain.StatusDraft, Instruction: "Draft 2"},
		},
	}
	srv := newServer(t, orch)
	defer srv.Close()

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "get_backlog",
			"arguments": map[string]any{
				"projectPath": "/project",
				"compact":     true,
				"limit":       1,
			},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var res struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	_ = json.Unmarshal(r.Result, &res)
	var items []map[string]any
	_ = json.Unmarshal([]byte(res.Content[0].Text), &items)
	if len(items) != 1 || items[0]["id"] != "b1" {
		t.Fatalf("expected 1 item with id b1, got %+v", items)
	}
}

func TestMCP_CompactToken_GetProviders(t *testing.T) {
	orch := &mockOrch{
		providers: []ports.ProviderInfo{
			{Name: "Ollama", Active: true, ActiveModel: "llama3", Models: []string{"llama3", "codellama"}},
			{Name: "OpenAI", Active: false, Models: []string{"gpt-4o"}},
		},
	}
	srv := newServer(t, orch)
	defer srv.Close()

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "get_providers",
			"arguments": map[string]any{
				"compact": true,
			},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var res struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	_ = json.Unmarshal(r.Result, &res)
	var items []map[string]any
	_ = json.Unmarshal([]byte(res.Content[0].Text), &items)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0]["name"] != "Ollama" || items[0]["active"] != true {
		t.Errorf("unexpected item 0: %+v", items[0])
	}
}

func TestMCP_CompactToken_GetAISessions(t *testing.T) {
	orch := &mockOrch{
		aiSessions: []domain.AISession{
			{ID: "sess-1", AgentName: "Claude", ModelID: "claude-3-5-sonnet", Status: domain.SessionStatusActive},
		},
	}
	srv := newServer(t, orch)
	defer srv.Close()

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "get_ai_sessions",
			"arguments": map[string]any{
				"compact": true,
			},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var res struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	_ = json.Unmarshal(r.Result, &res)
	var items []map[string]any
	_ = json.Unmarshal([]byte(res.Content[0].Text), &items)
	if len(items) != 1 || items[0]["agentName"] != "Claude" {
		t.Fatalf("unexpected session summary: %+v", items)
	}
}

func TestMCP_CompactToken_TaskSummaryHelpers(t *testing.T) {
	longInstr := strings.Repeat("W", 150)
	longLogs := strings.Repeat("L", 100)
	task := domain.Task{
		ID:          "long-task",
		Instruction: longInstr,
		Logs:        longLogs,
	}

	// We can test that get_task compact mode truncates instruction snippet and logs preview properly
	orch := &mockOrch{getTask: task}
	srv := newServer(t, orch)
	defer srv.Close()

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "get_task",
			"arguments": map[string]any{
				"id":      "long-task",
				"compact": true,
			},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var res struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	_ = json.Unmarshal(r.Result, &res)
	var sum map[string]any
	_ = json.Unmarshal([]byte(res.Content[0].Text), &sum)
	snip, _ := sum["instructionSnippet"].(string)
	if !strings.HasSuffix(snip, "...") || len(snip) != 120 {
		t.Errorf("instruction snippet not truncated to 120: len=%d %q", len(snip), snip)
	}
	preview, _ := sum["logsPreview"].(string)
	if !strings.HasSuffix(preview, "...") || len(preview) != 80 {
		t.Errorf("logs preview not truncated to 80: len=%d %q", len(preview), preview)
	}

	// Short instruction & short logs
	taskShort := domain.Task{
		ID:          "short-task",
		Instruction: "short",
		Logs:        "short logs",
	}
	orch.getTask = taskShort
	rShort := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "get_task",
			"arguments": map[string]any{
				"id":      "short-task",
				"compact": true,
			},
		},
	})
	if rShort.Error != nil {
		t.Fatalf("unexpected error: %+v", rShort.Error)
	}
	_ = json.Unmarshal(rShort.Result, &res)
	_ = json.Unmarshal([]byte(res.Content[0].Text), &sum)
	if sum["instructionSnippet"] != "short" || sum["logsPreview"] != "short logs" || sum["hasLogs"] != true {
		t.Errorf("unexpected short task summary: %+v", sum)
	}

	// Task with empty logs
	taskEmpty := domain.Task{
		ID:          "empty-logs",
		Instruction: "do something",
	}
	orch.getTask = taskEmpty
	rEmpty := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "get_task",
			"arguments": map[string]any{
				"id":      "empty-logs",
				"compact": true,
			},
		},
	})
	if rEmpty.Error != nil {
		t.Fatalf("unexpected error: %+v", rEmpty.Error)
	}
	var resEmpty struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	_ = json.Unmarshal(rEmpty.Result, &resEmpty)
	var sumEmpty map[string]any
	_ = json.Unmarshal([]byte(resEmpty.Content[0].Text), &sumEmpty)
	if sumEmpty["hasLogs"] != false || sumEmpty["logsPreview"] != nil {
		t.Errorf("expected hasLogs=false and nil logsPreview, got: %+v", sumEmpty)
	}
}
