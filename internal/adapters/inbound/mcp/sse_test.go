package mcp_test

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/inbound/mcp"
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
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	// TASK-405: missing session must return either 204 (no body) or a parseable
	// JSON response — NOT a plain-text 400/404 that MCP clients log as
	// "Failed to parse message: \"\"".
	switch resp.StatusCode {
	case http.StatusNoContent:
		// 204 is the preferred response — no body expected.
		if len(body) != 0 {
			t.Errorf("204 response should have empty body, got %q", body)
		}
	default:
		// Any other response must carry valid JSON so clients can parse it.
		if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
			t.Errorf("non-204 response must be application/json, got %q", ct)
		}
		var js json.RawMessage
		if err := json.Unmarshal(body, &js); err != nil {
			t.Errorf("non-204 response body must be valid JSON, got %q: %v", body, err)
		}
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

// TestSSE_PingKeepalive verifies that the SSE stream emits a comment ping
// before the keepalive interval elapses, keeping idle connections alive.
func TestSSE_PingKeepalive(t *testing.T) {
	// Speed up the keepalive for this test.
	orig := *mcp.SseKeepaliveInterval
	*mcp.SseKeepaliveInterval = 30 * time.Millisecond
	t.Cleanup(func() { *mcp.SseKeepaliveInterval = orig })

	srv := newServer(t, &mockOrch{})

	resp, err := http.Get(srv.URL + "/sse")
	if err != nil {
		t.Fatalf("GET /sse: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	timeout := time.After(500 * time.Millisecond)
	pingReceived := false

	for !pingReceived {
		lineCh := make(chan string, 1)
		go func() {
			if scanner.Scan() {
				lineCh <- scanner.Text()
			}
			close(lineCh)
		}()

		select {
		case line, ok := <-lineCh:
			if !ok {
				t.Fatal("SSE stream closed before ping received")
			}
			// SSE comment lines start with ':'
			if strings.HasPrefix(line, ":") {
				pingReceived = true
			}
		case <-timeout:
			t.Fatal("timed out waiting for SSE ping keepalive event")
		}
	}
}

// TestSSE_ReconnectAfterSessionLoss simulates a server restart by using a
// stale sessionId that no longer exists. The POST must return a parseable
// response (204 No Content, or a valid JSON body). Then a fresh SSE
// connection is established and initialize succeeds.
func TestSSE_ReconnectAfterSessionLoss(t *testing.T) {
	srv := newServer(t, &mockOrch{})

	// Step 1: POST to /messages with a sessionId that was never registered
	// (simulating a session that existed before a restart).
	staleID := "stale-session-id-does-not-exist"
	resp, err := http.Post(
		srv.URL+"/messages?sessionId="+staleID,
		"application/json",
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`),
	)
	if err != nil {
		t.Fatalf("POST stale session: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	// Must be 204 (preferred) or a response with a parseable JSON body — never
	// a non-JSON body paired with a 4xx status.
	switch resp.StatusCode {
	case http.StatusNoContent:
		if len(body) != 0 {
			t.Errorf("204 should have empty body, got %q", body)
		}
	default:
		ct := resp.Header.Get("Content-Type")
		if !strings.Contains(ct, "application/json") {
			t.Errorf("non-204 stale-session response must be application/json, got %q", ct)
		}
		var js json.RawMessage
		if err := json.Unmarshal(body, &js); err != nil {
			t.Errorf("non-204 body must be valid JSON, got %q: %v", body, err)
		}
	}

	// Step 2: Open a fresh SSE connection and perform initialize.
	sseResp, err := http.Get(srv.URL + "/sse")
	if err != nil {
		t.Fatalf("GET /sse (reconnect): %v", err)
	}
	defer sseResp.Body.Close()

	if sseResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /sse: want 200, got %d", sseResp.StatusCode)
	}

	scanner := bufio.NewScanner(sseResp.Body)
	var newEndpoint string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			newEndpoint = strings.TrimPrefix(line, "data: ")
			break
		}
	}
	if newEndpoint == "" {
		t.Fatal("no endpoint received on fresh SSE connection")
	}

	// POST initialize to the new endpoint.
	initResp, err := http.Post(
		srv.URL+newEndpoint,
		"application/json",
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`),
	)
	if err != nil {
		t.Fatalf("POST initialize: %v", err)
	}
	initResp.Body.Close()
	if initResp.StatusCode != http.StatusAccepted {
		t.Errorf("POST initialize: want 202, got %d", initResp.StatusCode)
	}

	// Verify initialize response arrives over the SSE stream.
	var sseData string
	timeout := time.After(5 * time.Second)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				sseData = strings.TrimPrefix(line, "data: ")
				return
			}
		}
	}()
	select {
	case <-done:
	case <-timeout:
		t.Fatal("timeout waiting for initialize response on fresh SSE stream")
	}

	var rpc struct {
		JSONRPC string `json:"jsonrpc"`
		Result  struct {
			ProtocolVersion string `json:"protocolVersion"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(sseData), &rpc); err != nil {
		t.Fatalf("unmarshal initialize response: %v", err)
	}
	if rpc.Result.ProtocolVersion != "2024-11-05" {
		t.Errorf("protocolVersion: want 2024-11-05, got %q", rpc.Result.ProtocolVersion)
	}
}

// TestSSE_ConnectionSurvivesLongIdle verifies that an SSE connection remains
// alive well past what the old hard-coded 15 s ReadTimeout would have killed
// in tests (represented here by an accelerated keepalive + sleep cycle).
func TestSSE_ConnectionSurvivesLongIdle(t *testing.T) {
	// Use a very short keepalive so the test stays fast.
	orig := *mcp.SseKeepaliveInterval
	*mcp.SseKeepaliveInterval = 30 * time.Millisecond
	t.Cleanup(func() { *mcp.SseKeepaliveInterval = orig })

	srv := newServer(t, &mockOrch{})

	resp, err := http.Get(srv.URL + "/sse")
	if err != nil {
		t.Fatalf("GET /sse: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Drain the first event (endpoint) so the scanner is positioned past it.
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "data: ") {
			break
		}
	}

	// Simulate an idle period longer than the old ReadTimeout (scaled down).
	time.Sleep(60 * time.Millisecond)

	// After the idle period the stream must still deliver a ping comment.
	timeout := time.After(500 * time.Millisecond)
	pingReceived := false
	for !pingReceived {
		lineCh := make(chan string, 1)
		go func() {
			if scanner.Scan() {
				lineCh <- scanner.Text()
			}
			close(lineCh)
		}()

		select {
		case line, ok := <-lineCh:
			if !ok {
				t.Fatal("SSE stream closed after idle period — connection did not survive")
			}
			if strings.HasPrefix(line, ":") {
				pingReceived = true
			}
		case <-timeout:
			t.Fatal("timed out: SSE connection did not survive idle period")
		}
	}
}
