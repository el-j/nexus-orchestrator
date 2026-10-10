package llm_openaicompat

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

func TestAPIErrorsCarryTheServersExplanation(t *testing.T) {
	cases := []struct {
		status int
		body   string
		want   string
	}{
		{400, `{"error":{"message":"This model's maximum context length is 8192 tokens"}}`, "maximum context length"},
		{401, `{"error":{"message":"Incorrect API key provided"}}`, "Incorrect API key"},
		{502, `<html>bad gateway</html>`, "unexpected status 502"},
		{429, `{}`, "rate limited"},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.status)
			fmt.Fprint(w, tc.body)
		}))
		a := NewAdapter("OpenAI", srv.URL, "sk-x", "gpt")
		_, err := a.GenerateCode("p")
		srv.Close()
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("status %d: %v", tc.status, err)
		}
		if err != nil && strings.Contains(err.Error(), "sk-x") {
			t.Errorf("the API key must never appear in errors: %v", err)
		}
	}
}

func TestRequestShapeAuthAndOtherPaths(t *testing.T) {
	var auth, path string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth, path = r.Header.Get("Authorization"), r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		fmt.Fprint(w, `{"choices":[{"message":{"content":"ok"}}]}`)
	}))
	defer srv.Close()
	a := NewAdapter("Custom", srv.URL, "sk-secret", "my-model")
	if a.ProviderName() != "Custom" || a.ActiveModel() != "my-model" || a.BaseURL() != srv.URL || a.ContextLimit() < 0 {
		t.Error("accessors")
	}
	if _, err := a.Chat([]domain.Message{{Role: domain.RoleUser, Content: "hi"}}); err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer sk-secret" || path != "/chat/completions" || body["model"] != "my-model" {
		t.Errorf("auth=%q path=%q body=%v", auth, path, body)
	}
	// No key: no Authorization header at all.
	keyless := NewAdapter("Local", srv.URL, "", "m")
	_, _ = keyless.GenerateCode("p")
	if auth != "" {
		t.Errorf("keyless adapter sent %q", auth)
	}

	for name, h := range map[string]http.HandlerFunc{
		"garbage":    func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "nope") },
		"no choices": func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, `{"choices":[]}`) },
	} {
		s := httptest.NewServer(h)
		if _, err := NewAdapter("X", s.URL, "", "m").GenerateCode("p"); err == nil {
			t.Errorf("%s must fail", name)
		}
		s.Close()
	}
	dead := NewAdapter("X", "http://127.0.0.1:1", "", "m")
	if _, err := dead.GenerateCode("p"); err == nil {
		t.Error("transport")
	}
	if _, err := dead.GetAvailableModels(); err == nil {
		t.Error("models transport")
	}
	if dead.Ping() {
		t.Error("ping transport")
	}
	if _, err := NewAdapter("X", "http://bad host", "", "m").GenerateCode("p"); err == nil {
		t.Error("bad url")
	}
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "x", 500) }))
	defer bad.Close()
	if NewAdapter("X", bad.URL, "", "m").Ping() {
		t.Error("ping must be false on 500")
	}
	if _, err := NewAdapter("X", bad.URL, "", "m").GetAvailableModels(); err == nil {
		t.Error("models 500")
	}
}
