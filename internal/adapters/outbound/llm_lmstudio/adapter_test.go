package llm_lmstudio

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

// newFullTestServer creates a test server that handles all LM Studio endpoints
// used by the adapter: /v1/models, /api/v0/model, /v1/chat/completions.
func newFullTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "test-model-1"},
				{"id": "test-model-2"},
			},
		})
	})

	// LM Studio native endpoint — used by ActiveModel() and ContextLimit().
	mux.HandleFunc("/api/v0/model", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"identifier":    "test-model-1",
			"contextLength": 4096,
		})
	})

	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
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

func TestLMStudio_Ping_Success(t *testing.T) {
	srv := newFullTestServer(t)
	defer srv.Close()

	a := NewLMStudioAdapter(srv.URL + "/v1")
	if !a.Ping() {
		t.Fatal("Ping() returned false; expected true")
	}
}

func TestLMStudio_Ping_ConnectionRefused(t *testing.T) {
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := dead.URL
	dead.Close()

	a := NewLMStudioAdapter(url + "/v1")
	if a.Ping() {
		t.Fatal("Ping() returned true; expected false (connection refused)")
	}
}

func TestLMStudio_Ping_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	a := NewLMStudioAdapter(srv.URL + "/v1")
	if a.Ping() {
		t.Fatal("Ping() returned true on 503; expected false")
	}
}

func TestLMStudio_GetAvailableModels(t *testing.T) {
	srv := newFullTestServer(t)
	defer srv.Close()

	a := NewLMStudioAdapter(srv.URL + "/v1")
	models, err := a.GetAvailableModels()
	if err != nil {
		t.Fatalf("GetAvailableModels() error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0] != "test-model-1" || models[1] != "test-model-2" {
		t.Errorf("unexpected models: %v", models)
	}
}

func TestLMStudio_GetAvailableModels_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	a := NewLMStudioAdapter(srv.URL + "/v1")
	_, err := a.GetAvailableModels()
	if err == nil {
		t.Fatal("expected error from GetAvailableModels() on 500 response")
	}
}

func TestLMStudio_GenerateCode(t *testing.T) {
	srv := newFullTestServer(t)
	defer srv.Close()

	a := NewLMStudioAdapter(srv.URL + "/v1")
	result, err := a.GenerateCode("write hello world")
	if err != nil {
		t.Fatalf("GenerateCode() error: %v", err)
	}
	if result != "generated output" {
		t.Errorf("expected %q, got %q", "generated output", result)
	}
}

func TestLMStudio_GenerateCode_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v0/model" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"identifier": "test-model"})
			return
		}
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	a := NewLMStudioAdapter(srv.URL + "/v1")
	_, err := a.GenerateCode("prompt")
	if err == nil {
		t.Fatal("expected error from GenerateCode() on 500 response")
	}
}

func TestLMStudio_Chat(t *testing.T) {
	srv := newFullTestServer(t)
	defer srv.Close()

	a := NewLMStudioAdapter(srv.URL + "/v1")
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

func TestLMStudio_Chat_ForwardsMessages(t *testing.T) {
	type chatRequest struct {
		Messages []map[string]string `json:"messages"`
	}

	var captured chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v0/model":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"identifier": "test-model"})
		case "/v1/chat/completions":
			json.NewDecoder(r.Body).Decode(&captured)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"choices": []map[string]interface{}{
					{"message": map[string]interface{}{"content": "ok"}},
				},
			})
		}
	}))
	defer srv.Close()

	a := NewLMStudioAdapter(srv.URL + "/v1")
	_, err := a.Chat([]domain.Message{
		{Role: domain.RoleUser, Content: "turn 1"},
		{Role: domain.RoleAssistant, Content: "reply 1"},
		{Role: domain.RoleUser, Content: "turn 2"},
	})
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if len(captured.Messages) != 3 {
		t.Errorf("expected 3 messages forwarded, got %d", len(captured.Messages))
	}
}
