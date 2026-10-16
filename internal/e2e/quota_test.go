//go:build e2e

package e2e_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// limitedOpenAI is an OpenAI-compatible server that answers the first chat request with 429 and a
// Retry-After, then behaves.
type limitedOpenAI struct {
	*httptest.Server
	chats atomic.Int32
}

func newLimitedOpenAI(t *testing.T, retryAfter string) *limitedOpenAI {
	t.Helper()
	f := &limitedOpenAI{}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"data":[{"id":"limited-model"}]}`)
	})
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		if f.chats.Add(1) == 1 {
			w.Header().Set("Retry-After", retryAfter)
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, `{"error":{"message":"slow down"}}`)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"```go\\npackage main\\n```\"}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":5,\"completion_tokens\":3}}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

// The only provider answers 429 with Retry-After: the task does not fail, it waits, is queued again by
// the daemon at the reset time, and completes; the provider list shows the cooldown meanwhile.
func TestDaemonHoldsATaskUntilALimitedProviderIsBackAndThenCompletesIt(t *testing.T) {
	skipOnWindows(t)
	srv := newLimitedOpenAI(t, "1")
	d := startDaemon(t, nil, daemonOpts{})
	d.waitHealthy()

	if code := d.do(http.MethodPost, "/api/providers/config", map[string]any{
		"name": "Limited", "kind": "openaicompat", "baseUrl": srv.URL + "/v1", "model": "limited-model", "enabled": true,
	}, nil); code != http.StatusCreated {
		t.Fatalf("register provider: HTTP %d\n%s", code, d.logs())
	}

	var created map[string]string
	if code := d.do(http.MethodPost, "/api/tasks", map[string]any{
		"projectPath": t.TempDir(), "targetFile": "main.go", "instruction": "write main", "providerName": "Limited",
	}, &created); code != http.StatusCreated {
		t.Fatalf("create task: HTTP %d\n%s", code, d.logs())
	}

	waiting := d.waitStatus(created["task_id"], "WAITING")
	if !strings.Contains(waiting.Logs, "all providers are cooling down") {
		t.Errorf("the log must say why: %s", waiting.Logs)
	}
	var providers []struct {
		Name             string    `json:"name"`
		RateLimited      bool      `json:"rateLimited"`
		RateLimitedUntil time.Time `json:"rateLimitedUntil"`
	}
	if code := d.do(http.MethodGet, "/api/providers", nil, &providers); code != http.StatusOK {
		t.Fatalf("providers: HTTP %d", code)
	}
	var seen bool
	for _, p := range providers {
		if p.Name == "Limited" {
			seen = true
			if !p.RateLimited || p.RateLimitedUntil.IsZero() {
				t.Errorf("the limited provider must show its cooldown: %+v", p)
			}
		}
	}
	if !seen {
		t.Fatalf("provider list: %+v", providers)
	}

	done := d.waitStatus(created["task_id"], "COMPLETED")
	if srv.chats.Load() != 2 {
		t.Errorf("the provider is asked once, limited, and once after the wait: %d", srv.chats.Load())
	}
	if !strings.Contains(done.Logs, "wait is over") {
		t.Errorf("the log must show the resume: %s", done.Logs)
	}
}
