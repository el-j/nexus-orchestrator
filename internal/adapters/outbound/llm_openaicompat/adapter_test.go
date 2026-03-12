package llm_openaicompat

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

// newFullTestServer creates a test server handling /models and /chat/completions.
func newFullTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/models", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "gpt-4o"},
				{"id": "gpt-3.5-turbo"},
			},
		})
	})

	mux.HandleFunc("/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"content": "generated output",
					},
				},
			},
		})
	})

	return httptest.NewServer(mux)
}

func TestOpenAICompat_Ping_Success(t *testing.T) {
	srv := newFullTestServer(t)
	defer srv.Close()

	a := NewAdapter("TestProvider", srv.URL, "", "gpt-4o")
	if !a.Ping() {
		t.Fatal("Ping() returned false; expected true")
	}
}

func TestOpenAICompat_Ping_ConnectionRefused(t *testing.T) {
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := dead.URL
	dead.Close()

	a := NewAdapter("TestProvider", url, "", "gpt-4o")
	if a.Ping() {
		t.Fatal("Ping() returned true; expected false (connection refused)")
	}
}

func TestOpenAICompat_Ping_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	a := NewAdapter("TestProvider", srv.URL, "", "gpt-4o")
	if a.Ping() {
		t.Fatal("Ping() returned true on 401; expected false")
	}
}

func TestOpenAICompat_Ping_SetsAuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := NewAdapter("TestProvider", srv.URL, "my-secret-key", "gpt-4o")
	a.Ping()

	const expected = "Bearer my-secret-key"
	if gotAuth != expected {
		t.Errorf("expected Authorization header %q, got %q", expected, gotAuth)
	}
}

func TestOpenAICompat_Ping_NoAuthHeader_WhenKeyEmpty(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := NewAdapter("TestProvider", srv.URL, "", "gpt-4o")
	a.Ping()

	if gotAuth != "" {
		t.Errorf("expected no Authorization header, got %q", gotAuth)
	}
}

func TestOpenAICompat_GetAvailableModels(t *testing.T) {
	srv := newFullTestServer(t)
	defer srv.Close()

	a := NewAdapter("TestProvider", srv.URL, "", "gpt-4o")
	models, err := a.GetAvailableModels()
	if err != nil {
		t.Fatalf("GetAvailableModels() error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0] != "gpt-4o" || models[1] != "gpt-3.5-turbo" {
		t.Errorf("unexpected models: %v", models)
	}
}

func TestOpenAICompat_GetAvailableModels_Cached(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{{"id": "gpt-4o"}},
		})
	}))
	defer srv.Close()

	a := NewAdapter("TestProvider", srv.URL, "", "gpt-4o")
	a.GetAvailableModels() //nolint:errcheck
	a.GetAvailableModels() //nolint:errcheck

	if callCount != 1 {
		t.Errorf("expected exactly 1 HTTP call (result should be cached), got %d", callCount)
	}
}

func TestOpenAICompat_GetAvailableModels_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	a := NewAdapter("TestProvider", srv.URL, "", "gpt-4o")
	_, err := a.GetAvailableModels()
	if err == nil {
		t.Fatal("expected error from GetAvailableModels() on 500 response")
	}
}

func TestOpenAICompat_GenerateCode(t *testing.T) {
	srv := newFullTestServer(t)
	defer srv.Close()

	a := NewAdapter("TestProvider", srv.URL, "", "gpt-4o")
	result, err := a.GenerateCode("write hello world")
	if err != nil {
		t.Fatalf("GenerateCode() error: %v", err)
	}
	if result != "generated output" {
		t.Errorf("expected %q, got %q", "generated output", result)
	}
}

func TestOpenAICompat_GenerateCode_RateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer srv.Close()

	a := NewAdapter("TestProvider", srv.URL, "", "gpt-4o")
	_, err := a.GenerateCode("prompt")
	if err == nil {
		t.Fatal("expected error from GenerateCode() on 429 response")
	}
}

func TestOpenAICompat_GenerateCode_SendsModel(t *testing.T) {
	type chatRequest struct {
		Model    string              `json:"model"`
		Messages []map[string]string `json:"messages"`
	}

	var captured chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&captured)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{"content": "ok"}},
			},
		})
	}))
	defer srv.Close()

	a := NewAdapter("TestProvider", srv.URL, "", "gpt-4o")
	_, err := a.GenerateCode("test prompt")
	if err != nil {
		t.Fatalf("GenerateCode() error: %v", err)
	}
	if captured.Model != "gpt-4o" {
		t.Errorf("expected model %q, got %q", "gpt-4o", captured.Model)
	}
	if len(captured.Messages) != 1 || captured.Messages[0]["content"] != "test prompt" {
		t.Errorf("unexpected messages: %v", captured.Messages)
	}
}

func TestOpenAICompat_Chat(t *testing.T) {
	srv := newFullTestServer(t)
	defer srv.Close()

	a := NewAdapter("TestProvider", srv.URL, "", "gpt-4o")
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

func TestOpenAICompat_Chat_ForwardsAllMessages(t *testing.T) {
	type chatRequest struct {
		Messages []map[string]string `json:"messages"`
	}

	var captured chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&captured)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{"content": "ok"}},
			},
		})
	}))
	defer srv.Close()

	a := NewAdapter("TestProvider", srv.URL, "", "gpt-4o")
	messages := []domain.Message{
		{Role: domain.RoleUser, Content: "turn 1"},
		{Role: domain.RoleAssistant, Content: "reply 1"},
		{Role: domain.RoleUser, Content: "turn 2"},
	}
	_, err := a.Chat(messages)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if len(captured.Messages) != 3 {
		t.Errorf("expected 3 messages forwarded, got %d", len(captured.Messages))
	}
}
