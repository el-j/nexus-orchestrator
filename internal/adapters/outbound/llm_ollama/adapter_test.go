package llm_ollama

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

// newFullTestServer creates a test server that handles all Ollama endpoints
// used by the adapter: /api/tags, /api/generate, /api/chat.
func newFullTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/api/tags", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []map[string]interface{}{
				{"name": "llama3"},
				{"name": "mistral"},
			},
		})
	})

	mux.HandleFunc("/api/generate", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"response": "generated output",
		})
	})

	mux.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": map[string]interface{}{
				"content": "chat reply",
			},
		})
	})

	return httptest.NewServer(mux)
}

func TestOllama_Ping_Success(t *testing.T) {
	srv := newFullTestServer(t)
	defer srv.Close()

	a := NewOllamaAdapter(srv.URL, "llama3")
	if !a.Ping() {
		t.Fatal("Ping() returned false; expected true")
	}
}

func TestOllama_Ping_ConnectionRefused(t *testing.T) {
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := dead.URL
	dead.Close()

	a := NewOllamaAdapter(url, "llama3")
	if a.Ping() {
		t.Fatal("Ping() returned true; expected false (connection refused)")
	}
}

func TestOllama_Ping_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	a := NewOllamaAdapter(srv.URL, "llama3")
	if a.Ping() {
		t.Fatal("Ping() returned true on 503; expected false")
	}
}

func TestOllama_GetAvailableModels(t *testing.T) {
	srv := newFullTestServer(t)
	defer srv.Close()

	a := NewOllamaAdapter(srv.URL, "llama3")
	models, err := a.GetAvailableModels()
	if err != nil {
		t.Fatalf("GetAvailableModels() error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0] != "llama3" || models[1] != "mistral" {
		t.Errorf("unexpected models: %v", models)
	}
}

func TestOllama_GetAvailableModels_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	a := NewOllamaAdapter(srv.URL, "llama3")
	_, err := a.GetAvailableModels()
	if err == nil {
		t.Fatal("expected error from GetAvailableModels() on 500 response")
	}
}

func TestOllama_GenerateCode(t *testing.T) {
	srv := newFullTestServer(t)
	defer srv.Close()

	a := NewOllamaAdapter(srv.URL, "llama3")
	result, err := a.GenerateCode("write hello world")
	if err != nil {
		t.Fatalf("GenerateCode() error: %v", err)
	}
	if result != "generated output" {
		t.Errorf("expected %q, got %q", "generated output", result)
	}
}

func TestOllama_GenerateCode_SendsModelAndPrompt(t *testing.T) {
	type generateRequest struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
		Stream bool   `json:"stream"`
	}

	var captured generateRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/generate" {
			json.NewDecoder(r.Body).Decode(&captured)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"response": "ok"})
	}))
	defer srv.Close()

	a := NewOllamaAdapter(srv.URL, "llama3")
	_, err := a.GenerateCode("test prompt")
	if err != nil {
		t.Fatalf("GenerateCode() error: %v", err)
	}
	if captured.Model != "llama3" {
		t.Errorf("expected model %q, got %q", "llama3", captured.Model)
	}
	if captured.Prompt != "test prompt" {
		t.Errorf("expected prompt %q, got %q", "test prompt", captured.Prompt)
	}
	if captured.Stream {
		t.Error("expected stream=false, got stream=true")
	}
}

func TestOllama_GenerateCode_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	a := NewOllamaAdapter(srv.URL, "llama3")
	_, err := a.GenerateCode("prompt")
	if err == nil {
		t.Fatal("expected error from GenerateCode() on 500 response")
	}
}

func TestOllama_Chat(t *testing.T) {
	srv := newFullTestServer(t)
	defer srv.Close()

	a := NewOllamaAdapter(srv.URL, "llama3")
	result, err := a.Chat([]domain.Message{
		{Role: domain.RoleUser, Content: "hello"},
		{Role: domain.RoleAssistant, Content: "hi there"},
		{Role: domain.RoleUser, Content: "write code"},
	})
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if result != "chat reply" {
		t.Errorf("expected %q, got %q", "chat reply", result)
	}
}

func TestOllama_Chat_ForwardsMessages(t *testing.T) {
	type chatRequest struct {
		Model    string              `json:"model"`
		Messages []map[string]string `json:"messages"`
		Stream   bool                `json:"stream"`
	}

	var captured chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/chat" {
			json.NewDecoder(r.Body).Decode(&captured)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": map[string]interface{}{"content": "ok"},
		})
	}))
	defer srv.Close()

	a := NewOllamaAdapter(srv.URL, "llama3")
	_, err := a.Chat([]domain.Message{
		{Role: domain.RoleUser, Content: "turn 1"},
		{Role: domain.RoleAssistant, Content: "reply 1"},
		{Role: domain.RoleUser, Content: "turn 2"},
	})
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if captured.Model != "llama3" {
		t.Errorf("expected model %q, got %q", "llama3", captured.Model)
	}
	if len(captured.Messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(captured.Messages))
	}
	if captured.Stream {
		t.Error("expected stream=false, got stream=true")
	}
}
