package llm_anthropic

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// hangingServer accepts requests and never answers until the client goes away,
// like a provider stuck mid-generation.
func hangingServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Reading the body lets the server notice the client hanging up.
		_, _ = io.Copy(io.Discard, r.Body)
		select {
		case <-r.Context().Done():
		case <-time.After(30 * time.Second):
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// within fails the test unless f returns before the deadline: a cancelled context
// must abort the call, not wait out the provider.
func within(t *testing.T, what string, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() { defer close(done); f() }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("%s did not return after its context was cancelled", what)
	}
}

func cancelSoon() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	return ctx, cancel
}

func TestCancelledContextAbortsEveryRequest(t *testing.T) {
	srv := hangingServer(t)
	a := func() *Adapter { x := NewAdapter("k", "m"); x.baseURL = srv.URL; return x }()
	msgs := []domain.Message{{Role: domain.RoleUser, Content: "hi"}}

	t.Run("Chat", func(t *testing.T) {
		ctx, cancel := cancelSoon()
		defer cancel()
		within(t, "Chat", func() {
			if _, err := a.Chat(ctx, msgs); err == nil || !errors.Is(err, context.Canceled) {
				t.Errorf("want a context.Canceled error, got %v", err)
			}
		})
	})
	t.Run("GenerateCode", func(t *testing.T) {
		ctx, cancel := cancelSoon()
		defer cancel()
		within(t, "GenerateCode", func() {
			if _, err := a.GenerateCode(ctx, "hi"); err == nil || !errors.Is(err, context.Canceled) {
				t.Errorf("want a context.Canceled error, got %v", err)
			}
		})
	})
	t.Run("Ping", func(t *testing.T) {
		ctx, cancel := cancelSoon()
		defer cancel()
		within(t, "Ping", func() {
			if a.Ping(ctx) {
				t.Error("a cancelled probe must report the provider as unreachable")
			}
		})
	})
	t.Run("GetAvailableModels", func(t *testing.T) {
		ctx, cancel := cancelSoon()
		defer cancel()
		within(t, "GetAvailableModels", func() {
			if _, err := a.GetAvailableModels(ctx); err == nil {
				t.Error("want an error")
			}
		})
	})
}

func TestAnAlreadyCancelledContextNeverReachesTheNetwork(t *testing.T) {
	hit := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hit <- struct{}{} }))
	defer srv.Close()
	a := func() *Adapter { x := NewAdapter("k", "m"); x.baseURL = srv.URL; return x }()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := a.Chat(ctx, []domain.Message{{Role: domain.RoleUser, Content: "x"}}); err == nil {
		t.Error("Chat with a dead context must fail")
	}
	select {
	case <-hit:
		t.Error("the request reached the server")
	default:
	}
}
