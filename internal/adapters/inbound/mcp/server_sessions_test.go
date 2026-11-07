package mcp_test

import (
	"encoding/json"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

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

func TestMCP_ToolCall_DeregisterAISession_Success(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      57,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "deregister_ai_session",
			"arguments": map[string]any{"session_id": "sess-1"},
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
	var payload map[string]bool
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if !payload["ok"] {
		t.Error("ok: want true, got false")
	}
}

func TestMCP_ToolCall_DeregisterAISession_MissingSessionID(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      58,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "deregister_ai_session",
			"arguments": map[string]any{},
		},
	})
	if r.Error == nil {
		t.Fatal("expected error for missing session_id")
	}
	if r.Error.Code != -32602 {
		t.Errorf("code: want -32602, got %d", r.Error.Code)
	}
}

func TestMCP_ToolCall_HeartbeatAISession_Success(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      59,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "heartbeat_ai_session",
			"arguments": map[string]any{"session_id": "sess-2"},
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
	var payload map[string]bool
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if !payload["ok"] {
		t.Error("ok: want true, got false")
	}
}

func TestMCP_ToolCall_HeartbeatAISession_MissingSessionID(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      60,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "heartbeat_ai_session",
			"arguments": map[string]any{},
		},
	})
	if r.Error == nil {
		t.Fatal("expected error for missing session_id")
	}
	if r.Error.Code != -32602 {
		t.Errorf("code: want -32602, got %d", r.Error.Code)
	}
}

func TestMCP_ToolCall_PurgeDisconnectedSessions(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      61,
		"method":  "tools/call",
		"params":  map[string]any{"name": "purge_disconnected_sessions", "arguments": map[string]any{}},
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
	var payload map[string]int
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if _, ok := payload["purged"]; !ok {
		t.Error("result must contain purged key")
	}
}

func TestMCP_ToolCall_GetDiscoveredAgents(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      62,
		"method":  "tools/call",
		"params":  map[string]any{"name": "get_discovered_agents", "arguments": map[string]any{}},
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
	// mock returns nil; valid JSON null or array
	var agents []domain.DiscoveredAgent
	if err := json.Unmarshal([]byte(result.Content[0].Text), &agents); err != nil {
		t.Fatalf("result text is not a valid JSON agent list: %v", err)
	}
}

func TestMCP_ToolCall_DelegateToNexus_Success(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      63,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "delegate_to_nexus",
			"arguments": map[string]any{"session_id": "sess-3"},
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
		t.Fatalf("unmarshal payload: %v", err)
	}
	if _, ok := payload["instruction"]; !ok {
		t.Error("result must contain instruction key")
	}
}

func TestMCP_ToolCall_DelegateToNexus_MissingSessionID(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      64,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "delegate_to_nexus",
			"arguments": map[string]any{},
		},
	})
	if r.Error == nil {
		t.Fatal("expected error for missing session_id")
	}
	if r.Error.Code != -32602 {
		t.Errorf("code: want -32602, got %d", r.Error.Code)
	}
}
