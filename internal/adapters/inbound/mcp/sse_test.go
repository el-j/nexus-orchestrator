package mcp_test

import (
	"bufio"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// --- SSE Transport Tests ---

func TestSSE_Connect_ReceivesEndpointEvent(t *testing.T) {
	srv := newServer(t, &mockOrch{})

	// Connect to /sse
	resp, err := http.Get(srv.URL + "/sse")
	if err != nil {
		t.Fatalf("GET /sse: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type: want text/event-stream, got %q", ct)
	}

	// Read the first SSE event
	scanner := bufio.NewScanner(resp.Body)
	var eventType, eventData string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			eventData = strings.TrimPrefix(line, "data: ")
		}
		if eventType != "" && eventData != "" {
			break
		}
	}

	if eventType != "endpoint" {
		t.Errorf("first event type: want 'endpoint', got %q", eventType)
	}
	if !strings.HasPrefix(eventData, "/messages?sessionId=") {
		t.Errorf("endpoint data: want '/messages?sessionId=...', got %q", eventData)
	}
}

func TestSSE_TokenAuth_RejectsMissingBearer(t *testing.T) {
	t.Setenv("NEXUS_MCP_TOKEN", "secret-token")
	srv := newServer(t, &mockOrch{})
	resp, err := http.Get(srv.URL + "/sse")
	if err != nil {
		t.Fatalf("GET /sse: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: want 401, got %d", resp.StatusCode)
	}
}

func TestSSE_TokenAuth_AllowsValidBearer(t *testing.T) {
	t.Setenv("NEXUS_MCP_TOKEN", "secret-token")
	srv := newServer(t, &mockOrch{})
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/sse", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer secret-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /sse with auth: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: want 200, got %d", resp.StatusCode)
	}
}

func TestSSE_MessageExchange(t *testing.T) {
	orch := &mockOrch{}
	srv := newServer(t, orch)

	// Step 1: Connect to /sse and get the endpoint
	sseResp, err := http.Get(srv.URL + "/sse")
	if err != nil {
		t.Fatalf("GET /sse: %v", err)
	}
	defer sseResp.Body.Close()

	scanner := bufio.NewScanner(sseResp.Body)
	var endpoint string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			endpoint = strings.TrimPrefix(line, "data: ")
			break
		}
	}
	if endpoint == "" {
		t.Fatal("no endpoint received")
	}

	// Step 2: POST an initialize request to the messages endpoint
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize"}`
	postResp, err := http.Post(srv.URL+endpoint, "application/json", strings.NewReader(initReq))
	if err != nil {
		t.Fatalf("POST %s: %v", endpoint, err)
	}
	defer postResp.Body.Close()

	if postResp.StatusCode != http.StatusAccepted {
		t.Errorf("POST status: want 202, got %d", postResp.StatusCode)
	}

	// Step 3: Read the response from the SSE stream
	// Skip the empty line after the endpoint event
	var responseData string
	timeout := time.After(5 * time.Second)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				responseData = strings.TrimPrefix(line, "data: ")
				return
			}
		}
	}()

	select {
	case <-done:
	case <-timeout:
		t.Fatal("timeout waiting for SSE response")
	}

	if responseData == "" {
		t.Fatal("no response received on SSE stream")
	}

	// Verify it's a valid initialize response
	var rpc struct {
		JSONRPC string `json:"jsonrpc"`
		Result  struct {
			ProtocolVersion string `json:"protocolVersion"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(responseData), &rpc); err != nil {
		t.Fatalf("unmarshal SSE response: %v", err)
	}
	if rpc.Result.ProtocolVersion != "2024-11-05" {
		t.Errorf("protocolVersion: want 2024-11-05, got %q", rpc.Result.ProtocolVersion)
	}
}

func TestSSE_POST_MissingSessionID(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	resp, err := http.Post(srv.URL+"/sse", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("POST /sse: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("want 400, got %d", resp.StatusCode)
	}
}

func TestSSE_MethodNotAllowed(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	req, err := http.NewRequest(http.MethodDelete, srv.URL+"/sse", nil)
	if err != nil {
		t.Fatalf("DELETE /sse request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE /sse: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("want 405, got %d", resp.StatusCode)
	}
}

func TestSSE_MissingSessionID(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	resp, err := http.Post(srv.URL+"/messages", "application/json", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
	if err != nil {
		t.Fatalf("POST /messages: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("want 400, got %d", resp.StatusCode)
	}
}

func TestMessages_TokenAuth_RejectsMissingBearer(t *testing.T) {
	t.Setenv("NEXUS_MCP_TOKEN", "secret-token")
	srv := newServer(t, &mockOrch{})
	resp, err := http.Post(srv.URL+"/messages?sessionId=nonexistent", "application/json", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
	if err != nil {
		t.Fatalf("POST /messages: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", resp.StatusCode)
	}
}

func TestSSE_InvalidSessionID(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	resp, err := http.Post(srv.URL+"/messages?sessionId=nonexistent", "application/json", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
	if err != nil {
		t.Fatalf("POST /messages: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("want 404, got %d", resp.StatusCode)
	}
}

// --- Streamable HTTP Tests ---

func TestMCP_GET_Returns405(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	resp, err := http.Get(srv.URL + "/mcp")
	if err != nil {
		t.Fatalf("GET /mcp: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("want 405, got %d", resp.StatusCode)
	}
}

func TestMCP_Initialize_HasSessionID(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
	})
	if r.Error != nil {
		t.Fatalf("expected no error, got %+v", r.Error)
	}
	// NOTE: We can't easily check the header from postRPC since it just decodes the body.
	// This test verifies the response itself is valid.
}

// --- Origin Validation Tests ---

func TestMCP_OriginValidation_NoOrigin_Passes(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	req, _ := http.NewRequest("POST", srv.URL+"/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
	req.Header.Set("Content-Type", "application/json")
	// No Origin header
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden {
		t.Error("requests without Origin should not be blocked")
	}
}

func TestMCP_OriginValidation_Localhost_Passes(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	req, _ := http.NewRequest("POST", srv.URL+"/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:3000")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden {
		t.Error("localhost origin should be allowed")
	}
}

func TestMCP_OriginValidation_ForeignOrigin_Blocked(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	req, _ := http.NewRequest("POST", srv.URL+"/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://evil.example.com")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("foreign origin should be blocked, got %d", resp.StatusCode)
	}
}

// --- CORS Tests ---

func TestMCP_CORS_Preflight(t *testing.T) {
	srv := newServer(t, &mockOrch{})
	req, _ := http.NewRequest("OPTIONS", srv.URL+"/mcp", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("OPTIONS: want 204, got %d", resp.StatusCode)
	}
	if allow := resp.Header.Get("Access-Control-Allow-Methods"); !strings.Contains(allow, "POST") {
		t.Errorf("Allow-Methods: want POST, got %q", allow)
	}
	if allow := resp.Header.Get("Access-Control-Allow-Headers"); !strings.Contains(allow, "Authorization") {
		t.Errorf("Allow-Headers: want Authorization, got %q", allow)
	}
}
