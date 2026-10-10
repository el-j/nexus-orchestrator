package llm_ollama

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

func showServer(t *testing.T, info map[string]any, hits *atomic.Int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/show" {
			http.NotFound(w, r)
			return
		}
		if hits != nil {
			hits.Add(1)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"model_info": info})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestContextLimit_ReadsTheArchitectureSpecificKey(t *testing.T) {
	cases := []struct {
		name string
		info map[string]any
		want int
	}{
		{"llama", map[string]any{"general.architecture": "llama", "llama.context_length": 8192.0}, 8192},
		{"qwen2 (not llama)", map[string]any{"general.architecture": "qwen2", "qwen2.context_length": 32768.0}, 32768},
		{"gemma3", map[string]any{"general.architecture": "gemma3", "gemma3.context_length": 131072.0}, 131072},
		{"architecture missing: any *.context_length", map[string]any{"phi3.context_length": 4096.0}, 4096},
		{"architecture key without a value falls back", map[string]any{"general.architecture": "mistral", "other.context_length": 2048.0}, 2048},
		{"several keys: deterministic (alphabetical)", map[string]any{"b.context_length": 2.0, "a.context_length": 1.0}, 1},
		{"no context length at all", map[string]any{"general.architecture": "x"}, 0},
		{"non-numeric value", map[string]any{"llama.context_length": "big"}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := showServer(t, tc.info, nil)
			if got := NewOllamaAdapter(srv.URL, "m").ContextLimit(); got != tc.want {
				t.Errorf("ContextLimit = %d, want %d", got, tc.want)
			}
		})
	}
	if contextLengthFromModelInfo(map[string]any{"x.context_length": 7}) != 7 {
		t.Error("int values must be accepted too")
	}
}

func TestContextLimit_CachesSuccessAndRetriesFailuresSoon(t *testing.T) {
	oldTTL, oldRetry := infoTTL, infoRetry
	defer func() { infoTTL, infoRetry = oldTTL, oldRetry }()
	infoTTL, infoRetry = time.Hour, 20*time.Millisecond

	// Ollama is down at the first call: the limit is unknown but is NOT cached forever.
	var up atomic.Bool
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if !up.Load() {
			http.Error(w, "starting", http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"model_info": map[string]any{"llama.context_length": 4096.0}})
	}))
	defer srv.Close()
	a := NewOllamaAdapter(srv.URL, "m")

	if got := a.ContextLimit(); got != 0 {
		t.Fatalf("down: %d", got)
	}
	before := hits.Load()
	a.ContextLimit() // inside the retry window: no new request
	if hits.Load() != before {
		t.Error("failed lookups must be throttled")
	}
	up.Store(true)
	time.Sleep(40 * time.Millisecond)
	if got := a.ContextLimit(); got != 4096 {
		t.Fatalf("after recovery the limit must be read, got %d", got)
	}
	// Success is cached: further calls do not hit the server.
	n := hits.Load()
	for i := 0; i < 5; i++ {
		a.ContextLimit()
	}
	if hits.Load() != n {
		t.Error("a successful value must be cached within the TTL")
	}
	// ... but it is refreshed once the TTL expires (the model may have changed).
	infoTTL = time.Millisecond
	time.Sleep(40 * time.Millisecond) // past both the TTL and the retry throttle
	a.ContextLimit()
	if hits.Load() == n {
		t.Error("the cache must expire")
	}
}

func TestContextLimit_ToleratesBadResponses(t *testing.T) {
	for name, h := range map[string]http.HandlerFunc{
		"500":      func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "x", 500) },
		"bad json": func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "nope") },
	} {
		srv := httptest.NewServer(h)
		if got := NewOllamaAdapter(srv.URL, "m").ContextLimit(); got != 0 {
			t.Errorf("%s: %d", name, got)
		}
		srv.Close()
	}
	if got := NewOllamaAdapter("http://127.0.0.1:1", "m").ContextLimit(); got != 0 {
		t.Errorf("unreachable: %d", got)
	}
}

func TestAccessorsAndErrorPaths(t *testing.T) {
	a := NewOllamaAdapter("http://example.invalid", "llama3")
	if a.ProviderName() != "Ollama" || a.ActiveModel() != "llama3" || a.BaseURL() != "http://example.invalid" {
		t.Error("accessors")
	}

	status := func(code int) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(code) }))
	}
	bad := status(500)
	defer bad.Close()
	b := NewOllamaAdapter(bad.URL, "m")
	if _, err := b.GetAvailableModels(); err == nil {
		t.Error("models 500")
	}
	if _, err := b.GenerateCode("p"); err == nil {
		t.Error("generate 500")
	}
	if _, err := b.Chat([]domain.Message{{Role: domain.RoleUser, Content: "x"}}); err == nil {
		t.Error("chat 500")
	}
	if b.Ping() {
		t.Error("ping must fail on 500")
	}

	garbage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "not json") }))
	defer garbage.Close()
	g := NewOllamaAdapter(garbage.URL, "m")
	if _, err := g.GetAvailableModels(); err == nil {
		t.Error("models decode")
	}
	if _, err := g.GenerateCode("p"); err == nil {
		t.Error("generate decode")
	}
	if _, err := g.Chat(nil); err == nil {
		t.Error("chat decode")
	}

	dead := NewOllamaAdapter("http://127.0.0.1:1", "m")
	if _, err := dead.GetAvailableModels(); err == nil {
		t.Error("models transport")
	}
	if _, err := dead.GenerateCode("p"); err == nil {
		t.Error("generate transport")
	}
	if _, err := dead.Chat(nil); err == nil {
		t.Error("chat transport")
	}
	if dead.Ping() {
		t.Error("ping transport")
	}

	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, `{"response":""}`) }))
	defer empty.Close()
	if _, err := NewOllamaAdapter(empty.URL, "m").GenerateCode("p"); err == nil {
		t.Error("an empty generate response must be an error")
	}
}
