package llm_gemini_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nexus-orchestrator/internal/adapters/outbound/llm_gemini"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// Ensure Adapter implements ports.LLMClient at compile time
var _ ports.LLMClient = (*llm_gemini.Adapter)(nil)

func TestAdapter_ProviderNameAndMetadata(t *testing.T) {
	a := llm_gemini.NewAdapter("test-key", "gemini-2.5-flash")
	if a.ProviderName() != "Google Gemini" {
		t.Errorf("expected 'Google Gemini', got %q", a.ProviderName())
	}
	if a.ActiveModel() != "gemini-2.5-flash" {
		t.Errorf("expected 'gemini-2.5-flash', got %q", a.ActiveModel())
	}
	if a.ContextLimit() != 1048576 {
		t.Errorf("expected 1048576, got %d", a.ContextLimit())
	}

	aDefault := llm_gemini.NewAdapter("test-key", "")
	if aDefault.ActiveModel() != "gemini-2.5-pro" {
		t.Errorf("expected default 'gemini-2.5-pro', got %q", aDefault.ActiveModel())
	}
	if aDefault.ContextLimit() != 2097152 {
		t.Errorf("expected 2097152, got %d", aDefault.ContextLimit())
	}
}

func TestAdapter_Ping(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-goog-api-key") != "valid-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"models": []}`))
	}))
	defer ts.Close()

	aValid := llm_gemini.NewAdapter("valid-key", "gemini-2.5-pro", ts.URL)
	if !aValid.Ping() {
		t.Error("expected ping to succeed")
	}

	aInvalid := llm_gemini.NewAdapter("wrong-key", "gemini-2.5-pro", ts.URL)
	if aInvalid.Ping() {
		t.Error("expected ping to fail on unauthorized")
	}
}

func TestAdapter_GetAvailableModels(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"models": []map[string]any{
				{
					"name":                       "models/gemini-2.5-pro",
					"supportedGenerationMethods": []string{"generateContent", "countTokens"},
				},
				{
					"name":                       "models/gemini-2.5-flash",
					"supportedGenerationMethods": []string{"generateContent"},
				},
				{
					"name":                       "models/text-embedding-004",
					"supportedGenerationMethods": []string{"embedContent"},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	a := llm_gemini.NewAdapter("key", "gemini-2.5-pro", ts.URL)
	models, err := a.GetAvailableModels()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 content-generating models, got %d: %v", len(models), models)
	}
	if models[0] != "gemini-2.5-pro" || models[1] != "gemini-2.5-flash" {
		t.Errorf("unexpected model names: %v", models)
	}
}

func TestAdapter_GenerateCode(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/models/gemini-2.5-pro:generateContent" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode req: %v", err)
		}

		resp := map[string]any{
			"candidates": []map[string]any{
				{
					"content": map[string]any{
						"role": "model",
						"parts": []map[string]any{
							{"text": "func hello() string { return \"world\" }"},
						},
					},
					"finishReason": "STOP",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	a := llm_gemini.NewAdapter("key", "gemini-2.5-pro", ts.URL)
	code, err := a.GenerateCode("Write a hello function in Go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "func hello() string { return \"world\" }"
	if code != expected {
		t.Errorf("expected %q, got %q", expected, code)
	}
}

func TestAdapter_Chat_WithSystemInstructionAndTurns(t *testing.T) {
	var receivedBody map[string]any

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		resp := map[string]any{
			"candidates": []map[string]any{
				{
					"content": map[string]any{
						"role": "model",
						"parts": []map[string]any{
							{"text": "Refactored code result"},
						},
					},
					"finishReason": "STOP",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	a := llm_gemini.NewAdapter("key", "gemini-2.5-flash", ts.URL)
	messages := []domain.Message{
		{Role: domain.RoleSystem, Content: "You are a senior Go architect."},
		{Role: domain.RoleUser, Content: "Part 1"},
		{Role: domain.RoleUser, Content: "Part 2"},
		{Role: domain.RoleAssistant, Content: "Understood"},
		{Role: domain.RoleUser, Content: "Now execute refactoring"},
	}

	reply, err := a.Chat(messages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply != "Refactored code result" {
		t.Errorf("expected 'Refactored code result', got %q", reply)
	}

	// Verify system instruction was extracted
	sys, ok := receivedBody["systemInstruction"].(map[string]any)
	if !ok {
		t.Fatal("expected systemInstruction in request payload")
	}
	sysParts := sys["parts"].([]any)
	if len(sysParts) != 1 || sysParts[0].(map[string]any)["text"] != "You are a senior Go architect." {
		t.Errorf("unexpected system instruction parts: %v", sysParts)
	}

	// Verify consecutive user messages were merged
	contents, ok := receivedBody["contents"].([]any)
	if !ok {
		t.Fatal("expected contents array")
	}
	// Turns: User (merged Part 1 + Part 2), Model, User -> 3 content items
	if len(contents) != 3 {
		t.Fatalf("expected 3 merged turns, got %d: %v", len(contents), contents)
	}
}

func TestAdapter_Chat_Errors(t *testing.T) {
	t.Run("rate_limited_429", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer ts.Close()

		a := llm_gemini.NewAdapter("key", "gemini-2.5-pro", ts.URL)
		_, err := a.Chat([]domain.Message{{Role: domain.RoleUser, Content: "hi"}})
		if err == nil || err.Error() != "gemini: rate limited (429)" {
			t.Errorf("expected rate limit error, got: %v", err)
		}
	})

	t.Run("server_error_500", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("internal quota error"))
		}))
		defer ts.Close()

		a := llm_gemini.NewAdapter("key", "gemini-2.5-pro", ts.URL)
		_, err := a.Chat([]domain.Message{{Role: domain.RoleUser, Content: "hi"}})
		if err == nil {
			t.Error("expected error on 500, got nil")
		}
	})

	t.Run("empty_candidates", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"candidates": []any{}})
		}))
		defer ts.Close()

		a := llm_gemini.NewAdapter("key", "gemini-2.5-pro", ts.URL)
		_, err := a.Chat([]domain.Message{{Role: domain.RoleUser, Content: "hi"}})
		if err == nil {
			t.Error("expected error on empty candidates, got nil")
		}
	})
}
