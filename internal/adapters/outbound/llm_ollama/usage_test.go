package llm_ollama

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

func ollamaServer(t *testing.T, body string) *Adapter {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, body) }))
	t.Cleanup(srv.Close)
	return NewOllamaAdapter(srv.URL, "qwen")
}

func TestUsage_NothingBeforeTheFirstCall(t *testing.T) {
	if _, ok := NewOllamaAdapter("http://x", "m").LastUsage(); ok {
		t.Error("no usage yet")
	}
}

func TestUsage_FromGenerateChatAndAStream(t *testing.T) {
	want := domain.Usage{Model: "qwen", InputTokens: 26, OutputTokens: 298}
	msgs := []domain.Message{{Role: domain.RoleUser, Content: "hi"}}
	counts := `"prompt_eval_count":26,"eval_count":298`

	a := ollamaServer(t, `{"response":"x",`+counts+`}`)
	if _, err := a.GenerateCode(context.Background(), "p"); err != nil {
		t.Fatal(err)
	}
	if u, ok := a.LastUsage(); !ok || u != want {
		t.Errorf("generate: %+v %v", u, ok)
	}

	a = ollamaServer(t, `{"message":{"content":"x"},`+counts+`}`)
	if _, err := a.Chat(context.Background(), msgs); err != nil {
		t.Fatal(err)
	}
	if u, ok := a.LastUsage(); !ok || u != want {
		t.Errorf("chat: %+v %v", u, ok)
	}

	a = ollamaServer(t, `{"message":{"content":"x"},"done":false}`+"\n"+`{"message":{"content":""},"done":true,`+counts+`}`+"\n")
	if _, err := a.ChatStream(context.Background(), msgs, func(string) {}); err != nil {
		t.Fatal(err)
	}
	if u, ok := a.LastUsage(); !ok || u != want {
		t.Errorf("stream: %+v %v", u, ok)
	}
}

func TestUsage_ACallWithoutCountsStillNamesTheModel(t *testing.T) {
	a := ollamaServer(t, `{"response":"x"}`)
	if _, err := a.GenerateCode(context.Background(), "p"); err != nil {
		t.Fatal(err)
	}
	if u, ok := a.LastUsage(); !ok || u.InputTokens != 0 || u.Model != "qwen" {
		t.Errorf("%+v %v", u, ok)
	}
}
