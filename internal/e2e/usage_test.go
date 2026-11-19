//go:build e2e

package e2e_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// Every model call that reports its token usage lands in the usage ledger, and the ledger can be read
// over REST, MCP and the CLI.
func TestDaemonRecordsTokenUsageAndReportsItOnEverySurface(t *testing.T) {
	skipOnWindows(t)
	llm := newFakeOllama(t)
	llm.reply = func(string) string { return "```go\npackage main\n```" }
	d := startDaemon(t, llm, daemonOpts{})
	d.waitHealthy()

	project := t.TempDir()
	var created map[string]string
	if code := d.do(http.MethodPost, "/api/tasks", map[string]any{
		"projectPath": project, "targetFile": "main.go", "instruction": "write main", "role": "architect",
	}, &created); code != http.StatusCreated {
		t.Fatalf("create task: HTTP %d\n%s", code, d.logs())
	}
	d.waitStatus(created["task_id"], "COMPLETED")

	var rest struct {
		Totals struct {
			Calls        int `json:"calls"`
			InputTokens  int `json:"inputTokens"`
			OutputTokens int `json:"outputTokens"`
		} `json:"totals"`
		Usage []struct {
			Provider string `json:"provider"`
			Model    string `json:"model"`
			Role     string `json:"role"`
		} `json:"usage"`
	}
	if code := d.do(http.MethodGet, "/api/usage?since=1h", nil, &rest); code != http.StatusOK {
		t.Fatalf("GET /api/usage: HTTP %d\n%s", code, d.logs())
	}
	if rest.Totals.Calls != 1 || rest.Totals.InputTokens != 11 || rest.Totals.OutputTokens != 7 {
		t.Errorf("totals: %+v", rest.Totals)
	}
	if len(rest.Usage) != 1 || rest.Usage[0].Provider != "Ollama" || rest.Usage[0].Role != "architect" || rest.Usage[0].Model == "" {
		t.Errorf("usage: %+v", rest.Usage)
	}

	out := runCLI(t, "nexus-cli", d.http, "usage", "--since", "1h")
	if !contains(out, "Ollama", "architect", "Total") {
		t.Errorf("nexus-cli usage:\n%s", out)
	}

	code, resp := d.rpc("tools/call", map[string]any{"name": "get_usage", "arguments": map[string]any{"since": "1h"}})
	raw, _ := json.Marshal(resp)
	if code != http.StatusOK || !strings.Contains(string(raw), `inputTokens\":11`) {
		t.Errorf("MCP get_usage: HTTP %d %s", code, raw)
	}
}
