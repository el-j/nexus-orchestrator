package llm_lmstudio

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

type lmState struct {
	model   atomic.Value // string
	ctx     atomic.Int32
	native  atomic.Bool // whether /api/v0/model works
	infoHit atomic.Int32
}

func lmServer(t *testing.T) (*httptest.Server, *lmState) {
	t.Helper()
	st := &lmState{}
	st.model.Store("model-a")
	st.ctx.Store(4096)
	st.native.Store(true)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v0/model":
			st.infoHit.Add(1)
			if !st.native.Load() {
				http.NotFound(w, r)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"identifier": st.model.Load(), "contextLength": st.ctx.Load()})
		case "/v1/models":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"id": "fallback-model"}}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, st
}

func fast(t *testing.T) {
	t.Helper()
	oldTTL, oldRetry := infoTTL, infoRetry
	infoTTL, infoRetry = 15*time.Millisecond, 15*time.Millisecond
	t.Cleanup(func() { infoTTL, infoRetry = oldTTL, oldRetry })
}

// Users load a different model in LM Studio while the daemon keeps running; the
// adapter must notice instead of pinning the first model forever.
func TestModelInfo_FollowsModelSwapsInsteadOfPinningTheFirstOne(t *testing.T) {
	fast(t)
	srv, st := lmServer(t)
	a := NewLMStudioAdapter(srv.URL + "/v1")
	if a.ActiveModel() != "model-a" || a.ContextLimit() != 4096 {
		t.Fatalf("initial: %q %d", a.ActiveModel(), a.ContextLimit())
	}
	st.model.Store("model-b")
	st.ctx.Store(32768)
	time.Sleep(30 * time.Millisecond)
	if a.ActiveModel() != "model-b" || a.ContextLimit() != 32768 {
		t.Errorf("after a swap: %q %d", a.ActiveModel(), a.ContextLimit())
	}
}

func TestModelInfo_RecoversWhenLMStudioWasDownAtFirstCall(t *testing.T) {
	fast(t)
	var up atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !up.Load() {
			http.Error(w, "loading", http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"identifier": "late-model", "contextLength": 2048})
	}))
	defer srv.Close()
	a := NewLMStudioAdapter(srv.URL + "/v1")
	if a.ActiveModel() != "" || a.activeModelOrDefault() != "local-model" || a.ContextLimit() != 0 {
		t.Fatal("down: unknown model, safe defaults")
	}
	up.Store(true)
	time.Sleep(30 * time.Millisecond)
	if a.ActiveModel() != "late-model" || a.ContextLimit() != 2048 {
		t.Errorf("must recover once LM Studio is up: %q %d", a.ActiveModel(), a.ContextLimit())
	}
}

func TestModelInfo_FallsBackToTheModelsListAndKeepsLastKnownOnFailure(t *testing.T) {
	fast(t)
	srv, st := lmServer(t)
	st.native.Store(false)
	a := NewLMStudioAdapter(srv.URL + "/v1")
	if a.ActiveModel() != "fallback-model" {
		t.Errorf("fallback to /models: %q", a.ActiveModel())
	}
	// Server goes away: the last known model is kept.
	srv.Close()
	time.Sleep(30 * time.Millisecond)
	if a.ActiveModel() != "fallback-model" {
		t.Errorf("last known value lost: %q", a.ActiveModel())
	}
}

func TestAccessorsAndErrorPaths(t *testing.T) {
	a := NewLMStudioAdapter("http://example.invalid/v1")
	if a.ProviderName() != "LM Studio" || a.BaseURL() != "http://example.invalid/v1" {
		t.Error("accessors")
	}
	srv, _ := lmServer(t)

	for name, h := range map[string]http.HandlerFunc{
		"500":     func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "x", 500) },
		"garbage": func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "not json") },
		"no choices": func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, `{"choices":[]}`)
		},
	} {
		bad := httptest.NewServer(h)
		b := NewLMStudioAdapter(bad.URL + "/v1")
		if _, err := b.GenerateCode("p"); err == nil {
			t.Errorf("%s: GenerateCode must fail", name)
		}
		if _, err := b.Chat([]domain.Message{{Role: domain.RoleUser, Content: "x"}}); err == nil {
			t.Errorf("%s: Chat must fail", name)
		}
		if name != "no choices" {
			if _, err := b.GetAvailableModels(); err == nil {
				t.Errorf("%s: GetAvailableModels must fail", name)
			}
		}
		bad.Close()
	}
	dead := NewLMStudioAdapter("http://127.0.0.1:1/v1")
	if _, err := dead.GenerateCode("p"); err == nil {
		t.Error("transport generate")
	}
	if _, err := dead.Chat(nil); err == nil {
		t.Error("transport chat")
	}
	if _, err := dead.GetAvailableModels(); err == nil {
		t.Error("transport models")
	}
	if dead.Ping() {
		t.Error("transport ping")
	}
	if !NewLMStudioAdapter(srv.URL + "/v1").Ping() {
		t.Error("ping against a live server")
	}
}
