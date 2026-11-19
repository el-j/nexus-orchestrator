package gateway

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

func TestServer_CORS_And_Auth(t *testing.T) {
	c := &fakeClient{name: "OpenAI", model: "gpt-4o", alive: true}
	disc := services.NewDiscoveryService(c)
	gw := NewGateway(Config{APIKey: "secret-key"}, disc)
	server := NewServer(Config{APIKey: "secret-key"}, gw)

	// OPTIONS preflight
	reqOpt := httptest.NewRequest(http.MethodOptions, "/v1/chat/completions", nil)
	wOpt := httptest.NewRecorder()
	server.Handler().ServeHTTP(wOpt, reqOpt)
	if wOpt.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS, got %d", wOpt.Code)
	}
	if wOpt.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("missing CORS header")
	}

	// Missing auth
	reqNoAuth := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	wNoAuth := httptest.NewRecorder()
	server.Handler().ServeHTTP(wNoAuth, reqNoAuth)
	if wNoAuth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", wNoAuth.Code)
	}

	// Invalid auth
	reqBadAuth := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	reqBadAuth.Header.Set("Authorization", "Bearer wrong-key")
	wBadAuth := httptest.NewRecorder()
	server.Handler().ServeHTTP(wBadAuth, reqBadAuth)
	if wBadAuth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", wBadAuth.Code)
	}

	// Valid auth
	reqGoodAuth := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	reqGoodAuth.Header.Set("Authorization", "Bearer secret-key")
	wGoodAuth := httptest.NewRecorder()
	server.Handler().ServeHTTP(wGoodAuth, reqGoodAuth)
	if wGoodAuth.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", wGoodAuth.Code)
	}
}

func TestServer_HealthAndNotFound(t *testing.T) {
	gw := NewGateway(Config{}, nil)
	server := NewServer(Config{}, gw)

	// Health
	for _, path := range []string{"/health", "/v1/health"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "nexus-router") {
			t.Fatalf("health check failed for %s: %d %s", path, w.Code, w.Body.String())
		}
	}

	// Not Found
	reqNF := httptest.NewRequest(http.MethodGet, "/v1/unknown", nil)
	wNF := httptest.NewRecorder()
	server.Handler().ServeHTTP(wNF, reqNF)
	if wNF.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", wNF.Code)
	}
}

func TestServer_Models(t *testing.T) {
	c := &fakeClient{name: "Anthropic", model: "claude-3-5-sonnet", alive: true}
	disc := services.NewDiscoveryService(c)
	gw := NewGateway(Config{}, disc)
	server := NewServer(Config{}, gw)

	// Method not allowed
	reqPost := httptest.NewRequest(http.MethodPost, "/v1/models", nil)
	wPost := httptest.NewRecorder()
	server.Handler().ServeHTTP(wPost, reqPost)
	if wPost.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", wPost.Code)
	}

	// GET /v1/models and /models
	for _, path := range []string{"/v1/models", "/models"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d", path, w.Code)
		}
		var list ModelList
		if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
			t.Fatal(err)
		}
		if list.Object != "list" || len(list.Data) == 0 {
			t.Fatalf("unexpected model list: %+v", list)
		}
	}
}

func TestServer_ChatCompletions_Validation(t *testing.T) {
	gw := NewGateway(Config{}, nil)
	server := NewServer(Config{}, gw)

	// Method not allowed
	reqGet := httptest.NewRequest(http.MethodGet, "/v1/chat/completions", nil)
	wGet := httptest.NewRecorder()
	server.Handler().ServeHTTP(wGet, reqGet)
	if wGet.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", wGet.Code)
	}

	// Invalid JSON
	reqBadJSON := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader("{invalid"))
	wBadJSON := httptest.NewRecorder()
	server.Handler().ServeHTTP(wBadJSON, reqBadJSON)
	if wBadJSON.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad json, got %d", wBadJSON.Code)
	}

	// Empty messages
	reqEmpty := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"messages":[]}`))
	wEmpty := httptest.NewRecorder()
	server.Handler().ServeHTTP(wEmpty, reqEmpty)
	if wEmpty.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty messages, got %d", wEmpty.Code)
	}
}

func TestServer_ChatCompletions_NonStreaming(t *testing.T) {
	c := &fakeClient{
		name:      "Anthropic",
		model:     "claude-3-5-sonnet",
		alive:     true,
		chatReply: "Hello from mock assistant",
		usage:     &domain.Usage{InputTokens: 5, OutputTokens: 10},
	}
	disc := services.NewDiscoveryService(c)
	gw := NewGateway(Config{}, disc)
	server := NewServer(Config{}, gw)

	body, _ := json.Marshal(ChatCompletionRequest{
		Model: "coder",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/chat/completions", bytes.NewReader(body))
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ChatCompletionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content != "Hello from mock assistant" {
		t.Fatalf("unexpected choice: %+v", resp.Choices)
	}

	// Test Gateway execution error
	c.chatErr = errors.New("upstream timeout")
	reqErr := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	wErr := httptest.NewRecorder()
	server.Handler().ServeHTTP(wErr, reqErr)
	if wErr.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", wErr.Code)
	}
}

func TestServer_ChatCompletions_Streaming(t *testing.T) {
	c := &fakeClient{
		name:         "Anthropic",
		model:        "claude-3-5-sonnet",
		alive:        true,
		streamChunks: []string{"Hello", " world!"},
	}
	disc := services.NewDiscoveryService(c)
	gw := NewGateway(Config{}, disc)
	server := NewServer(Config{}, gw)

	body, _ := json.Marshal(ChatCompletionRequest{
		Model: "coder",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hi"},
		},
		Stream: true,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "text/event-stream") {
		t.Fatalf("expected event-stream content type, got %s", w.Header().Get("Content-Type"))
	}

	bodyStr := w.Body.String()
	if !strings.Contains(bodyStr, `"role":"assistant"`) || !strings.Contains(bodyStr, `"content":"Hello"`) || !strings.Contains(bodyStr, "data: [DONE]") {
		t.Fatalf("unexpected streaming body:\n%s", bodyStr)
	}

	// Test streaming failure
	c.streamErr = errors.New("connection failed")
	c.streamChunks = nil
	reqErr := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	wErr := httptest.NewRecorder()
	server.Handler().ServeHTTP(wErr, reqErr)
	if !strings.Contains(wErr.Body.String(), "[stream error:") {
		t.Fatalf("expected stream error in output: %s", wErr.Body.String())
	}
}

type nonFlusherResponseWriter struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func (n *nonFlusherResponseWriter) Header() http.Header {
	if n.header == nil {
		n.header = make(http.Header)
	}
	return n.header
}

func (n *nonFlusherResponseWriter) Write(b []byte) (int, error) {
	return n.body.Write(b)
}

func (n *nonFlusherResponseWriter) WriteHeader(status int) {
	n.status = status
}

func TestServer_Streaming_NonFlusher_And_EmptyModel(t *testing.T) {
	c := &fakeClient{
		name:         "Anthropic",
		model:        "claude-3-5-sonnet",
		alive:        true,
		streamChunks: []string{"test"},
	}
	disc := services.NewDiscoveryService(c)
	gw := NewGateway(Config{DefaultRole: "coder"}, disc)
	server := NewServer(Config{DefaultRole: "coder"}, gw)

	// Non-flusher triggers 500 error
	body, _ := json.Marshal(ChatCompletionRequest{
		Model:    "coder",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
		Stream:   true,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	nf := &nonFlusherResponseWriter{}
	server.Handler().ServeHTTP(nf, req)
	if nf.status != http.StatusInternalServerError {
		t.Fatalf("expected 500 for non-flusher, got %d", nf.status)
	}

	// Empty model in streaming defaults to DefaultRole
	bodyEmptyModel, _ := json.Marshal(ChatCompletionRequest{
		Model:    "",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
		Stream:   true,
	})
	reqEmptyModel := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(bodyEmptyModel))
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, reqEmptyModel)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestServer_HistoryBudget_HeaderAndQueryParam(t *testing.T) {
	c := &fakeClient{name: "MockLLM", model: "mock-1", alive: true, chatReply: "ok"}
	disc := services.NewDiscoveryService(c)
	gw := NewGateway(Config{DefaultRole: "coder"}, disc)
	server := NewServer(Config{DefaultRole: "coder"}, gw)

	reqPayload := ChatCompletionRequest{
		Model: "coder",
		Messages: []ChatMessage{
			{Role: "user", Content: "turn 1: instruction"},
			{Role: "assistant", Content: "turn 1: result"},
			{Role: "user", Content: "turn 2: recent question"},
		},
	}
	body, _ := json.Marshal(reqPayload)

	// 1. With X-Nexus-History-Budget header
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("X-Nexus-History-Budget", "25")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 2. With history_budget query parameter
	req2 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions?history_budget=25", bytes.NewReader(body))
	w2 := httptest.NewRecorder()
	server.Handler().ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
}
