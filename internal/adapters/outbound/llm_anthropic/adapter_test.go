// Tests for the Anthropic adapter. Uses package llm_anthropic (not _test) so
// that the unexported baseURL field can be overridden to point at a local test
// server instead of the real Anthropic API.
package llm_anthropic

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// Interface compliance check: Adapter must satisfy ports.LLMClient.
var _ ports.LLMClient = (*Adapter)(nil)

// newTestAdapter creates an Adapter whose baseURL points at the supplied test
// server. Since baseURL is unexported, this helper must live in the same package.
func newTestAdapter(t *testing.T, serverURL, model string) *Adapter {
	t.Helper()
	a := NewAdapter("test-api-key", model)
	a.baseURL = serverURL
	return a
}

// newFullTestServer creates a test server handling all Anthropic endpoints used
// by the adapter: GET /v1/models and POST /v1/messages.
func newFullTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "claude-opus-4-5"},
				{"id": "claude-sonnet-4-5"},
			},
		})
	})

	mux.HandleFunc("/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": "generated output"},
			},
		})
	})

	return httptest.NewServer(mux)
}

func TestAnthropic_Ping_Success(t *testing.T) {
	srv := newFullTestServer(t)
	defer srv.Close()

	a := newTestAdapter(t, srv.URL, "claude-sonnet-4-5")
	if !a.Ping() {
		t.Fatal("Ping() returned false; expected true")
	}
}

func TestAnthropic_Ping_ConnectionRefused(t *testing.T) {
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := dead.URL
	dead.Close()

	a := newTestAdapter(t, url, "claude-sonnet-4-5")
	if a.Ping() {
		t.Fatal("Ping() returned true; expected false (connection refused)")
	}
}

func TestAnthropic_Ping_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	a := newTestAdapter(t, srv.URL, "claude-sonnet-4-5")
	if a.Ping() {
		t.Fatal("Ping() returned true on 401; expected false")
	}
}

func TestAnthropic_Ping_SetsRequiredHeaders(t *testing.T) {
	var gotAPIKey, gotVersion string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("x-api-key")
		gotVersion = r.Header.Get("anthropic-version")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := newTestAdapter(t, srv.URL, "claude-sonnet-4-5")
	a.Ping()

	if gotAPIKey != "test-api-key" {
		t.Errorf("expected x-api-key %q, got %q", "test-api-key", gotAPIKey)
	}
	if gotVersion != anthropicVersion {
		t.Errorf("expected anthropic-version %q, got %q", anthropicVersion, gotVersion)
	}
}

func TestAnthropic_GetAvailableModels(t *testing.T) {
	srv := newFullTestServer(t)
	defer srv.Close()

	a := newTestAdapter(t, srv.URL, "claude-sonnet-4-5")
	models, err := a.GetAvailableModels()
	if err != nil {
		t.Fatalf("GetAvailableModels() error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0] != "claude-opus-4-5" || models[1] != "claude-sonnet-4-5" {
		t.Errorf("unexpected models: %v", models)
	}
}

func TestAnthropic_GetAvailableModels_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	a := newTestAdapter(t, srv.URL, "claude-sonnet-4-5")
	_, err := a.GetAvailableModels()
	if err == nil {
		t.Fatal("expected error from GetAvailableModels() on 500 response")
	}
}

func TestAnthropic_GenerateCode(t *testing.T) {
	srv := newFullTestServer(t)
	defer srv.Close()

	a := newTestAdapter(t, srv.URL, "claude-sonnet-4-5")
	result, err := a.GenerateCode("write hello world")
	if err != nil {
		t.Fatalf("GenerateCode() error: %v", err)
	}
	if result != "generated output" {
		t.Errorf("expected %q, got %q", "generated output", result)
	}
}

func TestAnthropic_GenerateCode_RateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer srv.Close()

	a := newTestAdapter(t, srv.URL, "claude-sonnet-4-5")
	_, err := a.GenerateCode("prompt")
	if err == nil {
		t.Fatal("expected error from GenerateCode() on 429 response")
	}
}

func TestAnthropic_GenerateCode_SetsAuthHeaders(t *testing.T) {
	var gotAPIKey, gotVersion string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("x-api-key")
		gotVersion = r.Header.Get("anthropic-version")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": "ok"},
			},
		})
	}))
	defer srv.Close()

	a := newTestAdapter(t, srv.URL, "claude-sonnet-4-5")
	_, err := a.GenerateCode("prompt")
	if err != nil {
		t.Fatalf("GenerateCode() error: %v", err)
	}
	if gotAPIKey != "test-api-key" {
		t.Errorf("expected x-api-key %q, got %q", "test-api-key", gotAPIKey)
	}
	if gotVersion != anthropicVersion {
		t.Errorf("expected anthropic-version %q, got %q", anthropicVersion, gotVersion)
	}
}

func TestAnthropic_Chat(t *testing.T) {
	srv := newFullTestServer(t)
	defer srv.Close()

	a := newTestAdapter(t, srv.URL, "claude-sonnet-4-5")
	messages := []domain.Message{
		{Role: domain.RoleUser, Content: "hello"},
		{Role: domain.RoleAssistant, Content: "hi there"},
		{Role: domain.RoleUser, Content: "write code"},
	}
	result, err := a.Chat(messages)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if result != "generated output" {
		t.Errorf("expected %q, got %q", "generated output", result)
	}
}

func TestAnthropic_Chat_MergesConsecutiveSameRoleMessages(t *testing.T) {
	var gotMessages []anthropicMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/messages" {
			var body struct {
				Messages []anthropicMessage `json:"messages"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			gotMessages = body.Messages
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": "ok"},
			},
		})
	}))
	defer srv.Close()

	a := newTestAdapter(t, srv.URL, "claude-sonnet-4-5")
	// Two consecutive user messages should be merged into one.
	messages := []domain.Message{
		{Role: domain.RoleUser, Content: "first"},
		{Role: domain.RoleUser, Content: "second"},
		{Role: domain.RoleAssistant, Content: "reply"},
	}
	_, err := a.Chat(messages)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if len(gotMessages) != 2 {
		t.Fatalf("expected 2 messages after merging consecutive user turns, got %d: %v", len(gotMessages), gotMessages)
	}
	const wantContent = "first\nsecond"
	if gotMessages[0].Content != wantContent {
		t.Errorf("expected merged content %q, got %q", wantContent, gotMessages[0].Content)
	}
	if gotMessages[0].Role != "user" {
		t.Errorf("expected role %q, got %q", "user", gotMessages[0].Role)
	}
}

func TestAnthropic_Chat_FiltersSystemMessages(t *testing.T) {
	var gotMessages []anthropicMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/messages" {
			var body struct {
				Messages []anthropicMessage `json:"messages"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			gotMessages = body.Messages
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": "ok"},
			},
		})
	}))
	defer srv.Close()

	a := newTestAdapter(t, srv.URL, "claude-sonnet-4-5")
	// A "system" role is not user/assistant — it should be filtered out.
	messages := []domain.Message{
		{Role: domain.MessageRole("system"), Content: "system preamble"},
		{Role: domain.RoleUser, Content: "user message"},
		{Role: domain.RoleAssistant, Content: "assistant reply"},
	}
	_, err := a.Chat(messages)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if len(gotMessages) != 2 {
		t.Fatalf("expected 2 messages (system filtered), got %d: %v", len(gotMessages), gotMessages)
	}
	if gotMessages[0].Role != "user" {
		t.Errorf("expected first forwarded message to be user, got %q", gotMessages[0].Role)
	}
}

func TestAnthropic_Chat_NoTextContentBlock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return a content block with a non-"text" type — adapter should return an error.
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "tool_use", "id": "tool123"},
			},
		})
	}))
	defer srv.Close()

	a := newTestAdapter(t, srv.URL, "claude-sonnet-4-5")
	_, err := a.Chat([]domain.Message{{Role: domain.RoleUser, Content: "hi"}})
	if err == nil {
		t.Fatal("expected error when response has no text content block")
	}
}
