package mcp_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

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

func TestMCP_TokenAuth_RequiresBearer(t *testing.T) {
	t.Setenv("NEXUS_MCP_TOKEN", "mcp-token")
	srv := newServer(t, &mockOrch{})

	status, _ := postRPCRaw(t, srv, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "ping",
	}, "")
	if status != http.StatusUnauthorized {
		t.Fatalf("no auth: want 401, got %d", status)
	}

	status, _ = postRPCRaw(t, srv, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "ping",
	}, "Bearer wrong")
	if status != http.StatusUnauthorized {
		t.Fatalf("wrong auth: want 401, got %d", status)
	}

	status, body := postRPCRaw(t, srv, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "ping",
	}, "Bearer mcp-token")
	if status != http.StatusOK {
		t.Fatalf("correct auth: want 200, got %d (%s)", status, body)
	}
}

func TestMCP_TokenAuth_RejectsMissingBearer(t *testing.T) {
	t.Setenv("NEXUS_MCP_TOKEN", "secret-token")
	srv := newServer(t, &mockOrch{})
	status, _ := postRPCRaw(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "ping",
	}, "")
	if status != http.StatusUnauthorized {
		t.Fatalf("status: want 401, got %d", status)
	}
}

func TestMCP_TokenAuth_RejectsWrongBearer(t *testing.T) {
	t.Setenv("NEXUS_MCP_TOKEN", "secret-token")
	srv := newServer(t, &mockOrch{})
	status, _ := postRPCRaw(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "ping",
	}, "Bearer wrong-token")
	if status != http.StatusUnauthorized {
		t.Fatalf("status: want 401, got %d", status)
	}
}

func TestMCP_TokenAuth_AllowsValidBearer(t *testing.T) {
	t.Setenv("NEXUS_MCP_TOKEN", "secret-token")
	srv := newServer(t, &mockOrch{})
	status, body := postRPCRaw(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "ping",
	}, "Bearer secret-token")
	if status != http.StatusOK {
		t.Fatalf("status: want 200, got %d body=%s", status, body)
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

func TestMCP_ToolsList_Returns44Tools(t *testing.T) {
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
	if len(result.Tools) != 44 {
		t.Errorf("expected 44 tools, got %d", len(result.Tools))
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

func TestMCP_Ping(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0", "id": 42, "method": "ping",
	})
	if r.Error != nil {
		t.Fatalf("expected no error, got %+v", r.Error)
	}
	var result map[string]any
	if err := json.Unmarshal(r.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

func TestMCP_ResourcesList(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0", "id": 43, "method": "resources/list",
	})
	if r.Error != nil {
		t.Fatalf("expected no error, got %+v", r.Error)
	}
	var result struct {
		Resources []any `json:"resources"`
	}
	if err := json.Unmarshal(r.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result.Resources == nil {
		t.Fatal("expected resources key in result")
	}
	if len(result.Resources) != 0 {
		t.Errorf("expected empty resources, got %v", result.Resources)
	}
}

func TestMCP_PromptsList(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0", "id": 44, "method": "prompts/list",
	})
	if r.Error != nil {
		t.Fatalf("expected no error, got %+v", r.Error)
	}
	var result struct {
		Prompts []any `json:"prompts"`
	}
	if err := json.Unmarshal(r.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result.Prompts == nil {
		t.Fatal("expected prompts key in result")
	}
	if len(result.Prompts) != 0 {
		t.Errorf("expected empty prompts, got %v", result.Prompts)
	}
}
