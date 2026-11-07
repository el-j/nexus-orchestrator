package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
)

func TestHTTPAPI_TokenAuth_GuardsAPI(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "test-token")

	mock := &mockOrchestrator{}
	srv := httpapi.NewServer(mock, nil, nil)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	// Health stays open.
	resp, err := http.Get(ts.URL + "/api/health")
	if err != nil {
		t.Fatalf("GET /api/health: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health: want 200, got %d", resp.StatusCode)
	}

	// Other /api endpoints require a bearer token.
	resp, err = http.Get(ts.URL + "/api/tasks")
	if err != nil {
		t.Fatalf("GET /api/tasks: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated: want 401, got %d", resp.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/tasks", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/tasks with auth: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("authenticated: want 200, got %d", resp.StatusCode)
	}
}

func TestHTTPAPI_ConfigEndpoints(t *testing.T) {
	mock := &mockOrchestrator{}
	srv := httpapi.NewServer(mock, nil, nil)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/api/config")
	if err != nil {
		t.Fatalf("GET /api/config: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/config: want 200, got %d", resp.StatusCode)
	}
	var getResp map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&getResp); err != nil {
		t.Fatalf("decode GET /api/config: %v", err)
	}
	if getResp["queueCap"] == nil {
		t.Fatalf("expected queueCap in response, got %#v", getResp)
	}

	body := strings.NewReader(`{"queueCap":123}`)
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/config", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT /api/config: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT /api/config: want 200, got %d", resp.StatusCode)
	}
	var putResp map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&putResp); err != nil {
		t.Fatalf("decode PUT /api/config: %v", err)
	}
	if putResp["queueCap"] != float64(123) {
		t.Fatalf("expected queueCap=123, got %#v", putResp["queueCap"])
	}
}

func TestSecurityHeaders_OnAPIResponse(t *testing.T) {
	mock := &mockOrchestrator{}
	srv := httpapi.NewServer(mock, nil, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("expected X-Frame-Options DENY, got %q", got)
	}
	if got := resp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("expected X-Content-Type-Options nosniff, got %q", got)
	}
}

func TestDashboard_CSPHeader(t *testing.T) {
	mock := &mockOrchestrator{}
	srv := httpapi.NewServer(mock, nil, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/ui")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	csp := resp.Header.Get("Content-Security-Policy")
	if csp == "" {
		t.Error("expected Content-Security-Policy header on /ui, got empty")
	}
	if got := resp.Header.Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("expected X-Frame-Options DENY on /ui, got %q", got)
	}
}

// TestErrorResponse_HasJSONContentType verifies that error responses from the
// production handler set Content-Type: application/json and return a valid
// {"error":"..."} JSON body rather than plain text.
func TestErrorResponse_HasJSONContentType(t *testing.T) {
	mock := &mockOrchestrator{}
	srv := httpapi.NewServer(mock, nil, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Submitting malformed JSON triggers the decode-error path in handleCreateTask.
	body := bytes.NewBufferString(`not-valid-json`)
	resp, err := http.Post(ts.URL+"/api/tasks", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json for error response, got %q", ct)
	}
	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("error response body is not valid JSON: %v", err)
	}
	if _, ok := result["error"]; !ok {
		t.Error("expected 'error' key in JSON error response body")
	}
}
