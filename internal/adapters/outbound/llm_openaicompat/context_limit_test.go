package llm_openaicompat

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func modelsServer(t *testing.T, body string, status int, hits *atomic.Int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// The editor's models differ a lot in size (and so do local servers'); the adapter uses the
// context length the server reports for its model, and 128K only when it reports none.
func TestContextLimit_UsesWhatTheServerReports(t *testing.T) {
	var hits atomic.Int32
	srv := modelsServer(t, `{"data":[
		{"id":"small","context_length":8192},
		{"id":"vllm","max_model_len":32768},
		{"id":"other","context_window":65536},
		{"id":"silent"}]}`, 200, &hits)
	for model, want := range map[string]int{"small": 8192, "vllm": 32768, "other": 65536, "silent": 131072, "unlisted": 131072} {
		if got := NewAdapter("p", srv.URL, "", model).ContextLimit(context.Background()); got != want {
			t.Errorf("%s: ContextLimit = %d, want %d", model, got, want)
		}
	}
	if got := NewAdapter("p", srv.URL, "", "").ContextLimit(context.Background()); got != 0 {
		t.Errorf("no model: %d, want 0", got)
	}
}

func TestContextLimit_FallsBackAndDoesNotHammerAFailingServer(t *testing.T) {
	var hits atomic.Int32
	srv := modelsServer(t, `nope`, 500, &hits)
	a := NewAdapter("p", srv.URL, "", "m")
	for i := 0; i < 5; i++ {
		if got := a.ContextLimit(context.Background()); got != 131072 {
			t.Fatalf("ContextLimit = %d, want the default", got)
		}
	}
	if hits.Load() != 1 {
		t.Errorf("/models asked %d times, want once within the retry window", hits.Load())
	}
}
