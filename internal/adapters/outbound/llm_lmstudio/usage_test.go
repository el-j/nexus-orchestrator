package llm_lmstudio

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

var _ ports.UsageReporter = (*Adapter)(nil)

func TestUsage_NothingBeforeTheFirstCall(t *testing.T) {
	if _, ok := NewLMStudioAdapter("http://127.0.0.1:1/v1").LastUsage(); ok {
		t.Error("no usage yet")
	}
}

func completionServer(t *testing.T, body string) *Adapter {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	})
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, `{"data":[{"id":"local-m"}]}`) })
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return NewLMStudioAdapter(srv.URL + "/v1")
}

func TestUsage_FromGenerateAndChat(t *testing.T) {
	body := `{"model":"local-m","choices":[{"message":{"content":"hi"}}],"usage":{"prompt_tokens":14,"completion_tokens":6}}`
	want := domain.Usage{Model: "local-m", InputTokens: 14, OutputTokens: 6}
	a := completionServer(t, body)
	if _, err := a.GenerateCode(context.Background(), "p"); err != nil {
		t.Fatal(err)
	}
	if u, ok := a.LastUsage(); !ok || u != want {
		t.Errorf("generate: %+v %v", u, ok)
	}
	b := completionServer(t, body)
	if _, err := b.Chat(context.Background(), []domain.Message{{Role: domain.RoleUser, Content: "hi"}}); err != nil {
		t.Fatal(err)
	}
	if u, ok := b.LastUsage(); !ok || u != want {
		t.Errorf("chat: %+v %v", u, ok)
	}
}

func TestUsage_FromAStreamThatAsksForIt(t *testing.T) {
	var seen map[string]any
	body := piece("he") + piece("llo") +
		"data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: {\"model\":\"local-m\",\"choices\":[],\"usage\":{\"prompt_tokens\":21,\"completion_tokens\":8}}\n\n" +
		"data: [DONE]\n\n"
	a := streamingServer(t, http.StatusOK, "text/event-stream", body, &seen)
	if _, out, err := drive(a); err != nil || out != "hello" {
		t.Fatalf("%q %v", out, err)
	}
	if opts, _ := seen["stream_options"].(map[string]any); opts["include_usage"] != true {
		t.Errorf("the stream must ask for usage: %v", seen)
	}
	if u, ok := a.LastUsage(); !ok || u.InputTokens != 21 || u.OutputTokens != 8 || u.Model != "local-m" {
		t.Errorf("usage %+v %v", u, ok)
	}
}

func TestUsage_ACompletionWithoutCountsStillNamesTheModel(t *testing.T) {
	a := completionServer(t, `{"choices":[{"message":{"content":"x"}}]}`)
	if _, err := a.GenerateCode(context.Background(), "p"); err != nil {
		t.Fatal(err)
	}
	if u, ok := a.LastUsage(); !ok || u.InputTokens != 0 {
		t.Errorf("%+v %v", u, ok)
	}
}
