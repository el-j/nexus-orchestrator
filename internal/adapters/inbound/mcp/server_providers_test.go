package mcp_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

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

func TestMCP_ToolCall_ListProviderConfigs(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      50,
		"method":  "tools/call",
		"params":  map[string]any{"name": "list_provider_configs", "arguments": map[string]any{}},
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
	var cfgs []domain.ProviderConfig
	if err := json.Unmarshal([]byte(result.Content[0].Text), &cfgs); err != nil {
		t.Fatalf("result text is not a valid JSON provider config list: %v", err)
	}
}

func TestMCP_ToolCall_AddProviderConfig_Success(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      51,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "add_provider_config",
			"arguments": map[string]any{
				"kind":    "lmstudio",
				"name":    "My LM Studio",
				"enabled": true,
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
	var cfg domain.ProviderConfig
	if err := json.Unmarshal([]byte(result.Content[0].Text), &cfg); err != nil {
		t.Fatalf("unmarshal provider config: %v", err)
	}
	if cfg.Name != "My LM Studio" {
		t.Errorf("name: want My LM Studio, got %q", cfg.Name)
	}
}

func TestMCP_ToolCall_AddProviderConfig_MissingKind(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      52,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "add_provider_config",
			"arguments": map[string]any{"name": "No Kind"},
		},
	})
	if r.Error == nil {
		t.Fatal("expected error for missing kind")
	}
	if r.Error.Code != -32602 {
		t.Errorf("code: want -32602, got %d", r.Error.Code)
	}
}

func TestMCP_ToolCall_UpdateProviderConfig_Success(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      53,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "update_provider_config",
			"arguments": map[string]any{
				"id":      "cfg-1",
				"kind":    "lmstudio",
				"name":    "Updated Name",
				"enabled": true,
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
	var cfg domain.ProviderConfig
	if err := json.Unmarshal([]byte(result.Content[0].Text), &cfg); err != nil {
		t.Fatalf("unmarshal provider config: %v", err)
	}
	if cfg.Name != "Updated Name" {
		t.Errorf("name: want Updated Name, got %q", cfg.Name)
	}
}

func TestMCP_ToolCall_UpdateProviderConfig_MissingID(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      54,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "update_provider_config",
			"arguments": map[string]any{"kind": "lmstudio", "name": "No ID"},
		},
	})
	if r.Error == nil {
		t.Fatal("expected error for missing id")
	}
	if r.Error.Code != -32602 {
		t.Errorf("code: want -32602, got %d", r.Error.Code)
	}
}

func TestMCP_ToolCall_RemoveProviderConfig_Success(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      55,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "remove_provider_config",
			"arguments": map[string]any{"id": "cfg-1"},
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

func TestMCP_ToolCall_RemoveProviderConfig_MissingID(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      56,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "remove_provider_config",
			"arguments": map[string]any{},
		},
	})
	if r.Error == nil {
		t.Fatal("expected error for missing id")
	}
	if r.Error.Code != -32602 {
		t.Errorf("code: want -32602, got %d", r.Error.Code)
	}
}
