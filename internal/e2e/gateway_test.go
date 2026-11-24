//go:build e2e

package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// post sends a JSON body to the daemon with extra headers and returns the status and the body.
func (d *daemon) post(path string, body any, headers map[string]string) (int, string) {
	d.t.Helper()
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, d.http+path, bytes.NewReader(b))
	if err != nil {
		d.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		d.t.Fatalf("POST %s: %v\n%s", path, err, d.logs())
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(raw)
}

// Any tool that speaks the OpenAI API can use the daemon as its model endpoint: the models are listed, a
// chat is answered by the daemon's provider, and when the request names a project the model is shown that
// project's memory before the conversation, and the call lands in the usage ledger.
func TestDaemonServesAnOpenAICompatibleEndpointWithTheProjectMemory(t *testing.T) {
	llm := newFakeOllama(t)
	llm.reply = func(string) string { return "use the router in routes.go" }
	d := startDaemon(t, llm, daemonOpts{})
	d.waitHealthy()

	project := t.TempDir()
	d.do(http.MethodPost, "/api/brain/knowledge", map[string]any{
		"projectPath": project, "kind": "convention", "topic": "Routes", "content": "every route lives in routes.go and has an e2e test",
	}, nil)

	var models struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if code := d.do(http.MethodGet, "/v1/models", nil, &models); code != http.StatusOK || len(models.Data) == 0 {
		t.Fatalf("GET /v1/models: HTTP %d %+v", code, models)
	}

	chat := map[string]any{"model": "coder", "messages": []map[string]string{{"role": "user", "content": "where do routes live?"}}}
	code, body := d.post("/v1/chat/completions", chat, map[string]string{"X-Nexus-Project": project})
	if code != http.StatusOK || !strings.Contains(body, "use the router in routes.go") {
		t.Fatalf("chat: HTTP %d %s\n%s", code, body, d.logs())
	}

	llm.mu.Lock()
	seen := llm.chats
	llm.mu.Unlock()
	if len(seen) == 0 {
		t.Fatal("the provider was never asked")
	}
	first := strings.Join(seen[len(seen)-1], "\n")
	if !strings.HasPrefix(first, "system: Project memory") || !strings.Contains(first, "routes.go") || !strings.Contains(first, "user: where do routes live?") {
		t.Errorf("the model was not shown the project memory before the question:\n%s", first)
	}

	var usage struct {
		Totals struct {
			Calls int `json:"calls"`
		} `json:"totals"`
	}
	d.do(http.MethodGet, "/api/usage?since=1h&projectPath="+url.QueryEscape(project), nil, &usage)
	if usage.Totals.Calls != 1 {
		t.Errorf("the call is not in the usage ledger of the project: %+v", usage)
	}

	// A client that sends the content as a list of typed parts is understood too.
	parts := map[string]any{"model": "coder", "messages": []map[string]any{{"role": "user", "content": []map[string]string{{"type": "text", "text": "parts question"}}}}}
	if code, body := d.post("/v1/chat/completions", parts, nil); code != http.StatusOK || !strings.Contains(body, "use the router") {
		t.Errorf("content as parts: HTTP %d %s", code, body)
	}

	// Without a project the conversation reaches the model as it was sent.
	d.post("/v1/chat/completions", chat, nil)
	llm.mu.Lock()
	last := strings.Join(llm.chats[len(llm.chats)-1], "\n")
	llm.mu.Unlock()
	if strings.Contains(last, "Project memory") {
		t.Errorf("memory was added to a request that named no project:\n%s", last)
	}
}

// A token set for the API also guards the endpoint under /v1.
func TestDaemonGuardsTheOpenAIEndpointWithTheAPIToken(t *testing.T) {
	llm := newFakeOllama(t)
	d := startDaemon(t, llm, daemonOpts{env: []string{"NEXUS_API_TOKEN=s3cret"}})
	d.waitHealthy()
	if code, _ := d.post("/v1/chat/completions", map[string]any{"messages": []map[string]string{{"role": "user", "content": "hi"}}}, nil); code != http.StatusUnauthorized {
		t.Errorf("without the token: HTTP %d, want 401", code)
	}
	code, _ := d.post("/v1/chat/completions", map[string]any{"messages": []map[string]string{{"role": "user", "content": "hi"}}}, map[string]string{"Authorization": "Bearer s3cret"})
	if code != http.StatusOK {
		t.Errorf("with the token: HTTP %d, want 200\n%s", code, d.logs())
	}
}
