package llm_anthropic

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

type captured struct {
	Model    string `json:"model"`
	System   string `json:"system"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

func captureServer(t *testing.T, got *captured, reply string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, got)
		if r.Header.Get("x-api-key") != "test-api-key" || r.Header.Get("anthropic-version") == "" {
			t.Errorf("auth headers missing: %v", r.Header)
		}
		fmt.Fprint(w, reply)
	}))
	t.Cleanup(srv.Close)
	return srv
}

const okReply = `{"content":[{"type":"text","text":"hi"}]}`

// The Messages API has a separate top-level "system" field. System prompts used
// to be dropped silently.
func TestChat_SendsSystemPromptAndAValidTurnSequence(t *testing.T) {
	var got captured
	srv := captureServer(t, &got, okReply)
	a := newTestAdapter(t, srv.URL, "claude-sonnet-4-5")

	_, err := a.Chat([]domain.Message{
		{Role: domain.RoleSystem, Content: "Be terse."},
		{Role: domain.RoleAssistant, Content: "orphaned assistant turn"}, // may not come first
		{Role: domain.RoleSystem, Content: "Answer in Go."},
		{Role: domain.RoleUser, Content: "first"},
		{Role: domain.RoleUser, Content: "second"}, // merged with the previous user turn
		{Role: domain.RoleAssistant, Content: ""},  // empty: rejected by the API, so skipped
		{Role: domain.RoleAssistant, Content: "reply"},
		{Role: domain.RoleUser, Content: "third"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.System != "Be terse.\n\nAnswer in Go." {
		t.Errorf("system = %q", got.System)
	}
	roles := ""
	for _, m := range got.Messages {
		roles += m.Role[:1]
	}
	if roles != "uau" || got.Messages[0].Content != "first\nsecond" {
		t.Errorf("turns must start with user and alternate: %+v", got.Messages)
	}
}

func TestChat_WithoutAnyUserTurnFailsBeforeCallingTheAPI(t *testing.T) {
	var got captured
	srv := captureServer(t, &got, okReply)
	a := newTestAdapter(t, srv.URL, "m")
	if _, err := a.Chat([]domain.Message{{Role: domain.RoleSystem, Content: "only system"}}); err == nil {
		t.Error("a conversation with no user turn must be rejected locally")
	}
	if got.Model != "" {
		t.Error("no request should have been sent")
	}
}

func TestAPIErrorsCarryAnthropicsExplanation(t *testing.T) {
	cases := []struct {
		status int
		body   string
		want   string
	}{
		{400, `{"type":"error","error":{"type":"invalid_request_error","message":"prompt is too long: 250000 tokens"}}`, "prompt is too long"},
		{401, `{"type":"error","error":{"message":"invalid x-api-key"}}`, "invalid x-api-key"},
		{500, `<html>gateway</html>`, "unexpected status 500"},
		{429, `{}`, "rate limited"},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.status)
			fmt.Fprint(w, tc.body)
		}))
		a := newTestAdapter(t, srv.URL, "m")
		_, err := a.GenerateCode("p")
		srv.Close()
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("status %d: %v, want mention of %q", tc.status, err, tc.want)
		}
		if err != nil && strings.Contains(err.Error(), "html") {
			t.Errorf("a non-JSON body must not leak into the error: %v", err)
		}
	}
}

func TestAccessorsContextLimitAndOtherPaths(t *testing.T) {
	a := NewAdapter("k", "claude-3-opus-20240229")
	if a.ProviderName() != "Anthropic" || a.ActiveModel() != "claude-3-opus-20240229" || a.BaseURL() == "" {
		t.Error("accessors")
	}
	if a.ContextLimit() != 200000 || NewAdapter("k", "some-future-model").ContextLimit() != 200000 {
		t.Error("context limit")
	}

	for name, h := range map[string]http.HandlerFunc{
		"500":     func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "x", 500) },
		"garbage": func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "nope") },
		"no text": func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, `{"content":[{"type":"tool_use"}]}`) },
	} {
		srv := httptest.NewServer(h)
		b := newTestAdapter(t, srv.URL, "m")
		if _, err := b.GenerateCode("p"); err == nil {
			t.Errorf("%s: GenerateCode must fail", name)
		}
		if name != "no text" {
			if _, err := b.GetAvailableModels(); err == nil {
				t.Errorf("%s: GetAvailableModels must fail", name)
			}
		}
		if name == "500" && b.Ping() {
			t.Error("Ping must be false on 500")
		}
		srv.Close()
	}
	dead := newTestAdapter(t, "http://127.0.0.1:1", "m")
	if dead.Ping() {
		t.Error("ping transport")
	}
	if _, err := dead.GetAvailableModels(); err == nil {
		t.Error("models transport")
	}
	if _, err := dead.GenerateCode("p"); err == nil {
		t.Error("generate transport")
	}
	// Invalid base URL fails while building the request.
	bad := newTestAdapter(t, "http://bad host", "m")
	if bad.Ping() {
		t.Error("ping bad url")
	}
	if _, err := bad.GetAvailableModels(); err == nil {
		t.Error("models bad url")
	}
	if _, err := bad.GenerateCode("p"); err == nil {
		t.Error("generate bad url")
	}
}
