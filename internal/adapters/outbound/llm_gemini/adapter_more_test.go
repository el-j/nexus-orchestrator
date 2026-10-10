package llm_gemini_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nexus-orchestrator/internal/adapters/outbound/llm_gemini"
	"nexus-orchestrator/internal/core/domain"
)

func TestGemini_AccessorsAndBaseURLHandling(t *testing.T) {
	a := llm_gemini.NewAdapter("k", "gemini-1.0-pro", "http://example.test/")
	if a.BaseURL() != "http://example.test" || a.ContextLimit() != 32768 {
		t.Errorf("baseURL=%q limit=%d", a.BaseURL(), a.ContextLimit())
	}
	if got := llm_gemini.NewAdapter("k", "brand-new-model").ContextLimit(); got != 1048576 {
		t.Errorf("unknown models default to 1M, got %d", got)
	}
	if got := llm_gemini.NewAdapter("k", "m", "   ").BaseURL(); got == "" || strings.TrimSpace(got) == "" {
		t.Errorf("a blank override must fall back to the default endpoint, got %q", got)
	}
}

func TestGemini_ErrorPathsAndAuthHeader(t *testing.T) {
	var keyHeader string
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keyHeader = r.Header.Get("x-goog-api-key")
		if strings.HasSuffix(r.URL.Path, "/models") {
			fmt.Fprint(w, `{"models":[{"name":"models/a","supportedGenerationMethods":["embedContent"]}]}`)
			return
		}
		fmt.Fprint(w, `{"candidates":[{"content":{"parts":[{"text":"hi "},{"text":"there"}]}}]}`)
	}))
	defer ok.Close()
	a := llm_gemini.NewAdapter("secret-key", "m", ok.URL)
	if !a.Ping() || keyHeader != "secret-key" {
		t.Errorf("ping/auth: header=%q", keyHeader)
	}
	if models, err := a.GetAvailableModels(); err != nil || len(models) == 0 {
		t.Errorf("a list with no generateContent models falls back to a default list: %v %v", models, err)
	}
	if got, err := a.GenerateCode("p"); err != nil || got != "hi there" {
		t.Errorf("parts must be concatenated: %q %v", got, err)
	}
	keyless := llm_gemini.NewAdapter("", "m", ok.URL)
	_ = keyless.Ping()
	if keyHeader != "" {
		t.Errorf("no key, no header; got %q", keyHeader)
	}

	for name, h := range map[string]http.HandlerFunc{
		"500":      func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "boom", 500) },
		"429":      func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(429) },
		"garbage":  func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "nope") },
		"no parts": func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, `{"candidates":[]}`) },
	} {
		s := httptest.NewServer(h)
		b := llm_gemini.NewAdapter("k", "m", s.URL)
		if _, err := b.Chat([]domain.Message{{Role: domain.RoleUser, Content: "x"}}); err == nil {
			t.Errorf("%s: Chat must fail", name)
		}
		if name == "500" || name == "garbage" {
			if _, err := b.GetAvailableModels(); err == nil {
				t.Errorf("%s: models must fail", name)
			}
		}
		if name == "500" && b.Ping() {
			t.Error("ping 500")
		}
		s.Close()
	}
	dead := llm_gemini.NewAdapter("k", "m", "http://127.0.0.1:1")
	if dead.Ping() {
		t.Error("ping transport")
	}
	if _, err := dead.GetAvailableModels(); err == nil {
		t.Error("models transport")
	}
	if _, err := dead.GenerateCode("p"); err == nil {
		t.Error("chat transport")
	}
	bad := llm_gemini.NewAdapter("k", "m", "http://bad host")
	if bad.Ping() {
		t.Error("ping bad url")
	}
	if _, err := bad.GetAvailableModels(); err == nil {
		t.Error("models bad url")
	}
	if _, err := bad.GenerateCode("p"); err == nil {
		t.Error("chat bad url")
	}
}
