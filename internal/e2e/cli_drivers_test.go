//go:build e2e

package e2e_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeClients writes stand-ins for the three subscription clients into a new folder and returns it. Each
// answers with a Go file; claude speaks stream-json with token usage and a plan reading, agy answers
// `agy models` and plain text, copilot plain text. Each records the arguments it was started with, and
// claude fails with a rejected seven-day limit when the prompt contains QUOTA.
func fakeClients(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	scripts := map[string]string{
		"claude": `#!/bin/sh
echo "$@" > "$(dirname "$0")/claude.args"
prompt=$(cat)
case "$prompt" in
*QUOTA*)
  printf '%s\n' '{"type":"rate_limit_event","rate_limit_info":{"status":"rejected","resetsAt":'$(( $(date +%s) + 3600 ))',"rateLimitType":"seven_day","utilization":1}}'
  printf '%s\n' '{"type":"result","is_error":true,"result":"limit reached","api_error_status":429}'
  exit 1;;
esac
printf '%s\n' '{"type":"system","subtype":"init","model":"claude-fake"}'
printf '%s\n' '{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"` + "```go\\npackage main\\n\\nfunc FromClaude() {}\\n```" + `"}}}'
printf '%s\n' '{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","resetsAt":'$(( $(date +%s) + 3600 ))',"rateLimitType":"five_hour","utilization":0.25}}'
printf '%s\n' '{"type":"result","is_error":false,"result":"` + "```go\\npackage main\\n\\nfunc FromClaude() {}\\n```" + `","total_cost_usd":0.01,"usage":{"input_tokens":21,"output_tokens":9}}'
`,
		"agy": `#!/bin/sh
echo "$@" > "$(dirname "$0")/agy.args"
if [ "$1" = models ]; then printf 'agy-fast\tFast\n'; exit 0; fi
cat > /dev/null
printf '` + "```go\\npackage main\\n\\nfunc FromAgy() {}\\n```" + `\n'
`,
		"copilot": `#!/bin/sh
echo "$@" > "$(dirname "$0")/copilot.args"
cat > /dev/null
printf '` + "```go\\npackage main\\n\\nfunc FromCopilot() {}\\n```" + `\n'
`,
	}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte(scripts[n]), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// Subscription clients found on the PATH are discovered, promoted to providers and then run a task each:
// the daemon starts them with the flags that keep the call cheap, writes what they answer, and records the
// tokens claude reports.
func TestDaemonRunsTasksThroughTheClaudeAgyAndCopilotClients(t *testing.T) {
	skipOnWindows(t)
	dir := fakeClients(t, "claude", "agy", "copilot")
	d := startDaemon(t, nil, daemonOpts{pathDir: dir})
	d.waitHealthy()

	var found []struct {
		ID, Name string
	}
	if code := d.do(http.MethodPost, "/api/providers/discovered/scan", nil, &found); code != http.StatusOK {
		t.Fatalf("scan: HTTP %d\n%s", code, d.logs())
	}
	ids := map[string]string{}
	for _, f := range found {
		ids[f.ID] = f.Name
	}
	for _, id := range []string{"cli-claude", "cli-agy", "cli-copilot"} {
		if ids[id] == "" {
			t.Fatalf("%s was not discovered: %v", id, ids)
		}
		if code := d.do(http.MethodPost, "/api/providers/promote/"+id, nil, nil); code != http.StatusOK && code != http.StatusNoContent {
			t.Fatalf("promote %s: HTTP %d\n%s", id, code, d.logs())
		}
	}

	project := t.TempDir()
	for _, c := range []struct{ id, wantFile, wantFunc string }{
		{"cli-claude", "claude.go", "FromClaude"},
		{"cli-agy", "agy.go", "FromAgy"},
		{"cli-copilot", "copilot.go", "FromCopilot"},
	} {
		var created map[string]string
		if code := d.do(http.MethodPost, "/api/tasks", map[string]any{
			"projectPath": project, "targetFile": c.wantFile, "instruction": "write " + c.wantFunc,
			"providerName": ids[c.id],
		}, &created); code != http.StatusCreated {
			t.Fatalf("create task for %s: HTTP %d\n%s", c.id, code, d.logs())
		}
		d.waitStatus(created["task_id"], "COMPLETED")
		if got := mustRead(t, filepath.Join(project, c.wantFile)); !contains(got, "func "+c.wantFunc) {
			t.Errorf("%s wrote %q", c.id, got)
		}
	}

	claudeArgs := mustRead(t, filepath.Join(dir, "claude.args"))
	for _, flag := range []string{"-p", "--output-format stream-json", "--tools", "--no-session-persistence", "--strict-mcp-config"} {
		if !strings.Contains(claudeArgs, flag) {
			t.Errorf("claude was started without %q: %s", flag, claudeArgs)
		}
	}
	if args := mustRead(t, filepath.Join(dir, "copilot.args")); !contains(args, "-s", "--no-ask-user") {
		t.Errorf("copilot args = %q", args)
	}

	var usage struct {
		Usage []struct {
			Provider     string `json:"provider"`
			Model        string `json:"model"`
			InputTokens  int    `json:"inputTokens"`
			OutputTokens int    `json:"outputTokens"`
		} `json:"usage"`
	}
	d.do(http.MethodGet, "/api/usage?since=1h", nil, &usage)
	var seen bool
	for _, u := range usage.Usage {
		seen = seen || (u.Model == "claude-fake" && u.InputTokens == 21 && u.OutputTokens == 9)
	}
	if !seen {
		t.Errorf("the tokens claude reported are not in the ledger: %+v", usage.Usage)
	}
}

// When the only provider is a subscription that has hit its limit, the task waits for the reset instead of
// failing: its status is WAITING and the log names the limit.
func TestDaemonHoldsATaskWhenTheSubscriptionLimitIsReached(t *testing.T) {
	skipOnWindows(t)
	dir := fakeClients(t, "claude")
	d := startDaemon(t, nil, daemonOpts{pathDir: dir})
	d.waitHealthy()
	d.do(http.MethodPost, "/api/providers/discovered/scan", nil, nil)
	if code := d.do(http.MethodPost, "/api/providers/promote/cli-claude", nil, nil); code != http.StatusOK && code != http.StatusNoContent {
		t.Fatalf("promote: HTTP %d\n%s", code, d.logs())
	}

	var created map[string]string
	if code := d.do(http.MethodPost, "/api/tasks", map[string]any{
		"projectPath": t.TempDir(), "targetFile": "x.go", "instruction": "QUOTA please",
	}, &created); code != http.StatusCreated {
		t.Fatalf("create task: HTTP %d\n%s", code, d.logs())
	}
	waiting := d.waitStatus(created["task_id"], "WAITING")
	if !strings.Contains(strings.ToLower(waiting.Logs), "limit") {
		t.Errorf("the log of a waiting task should name the limit: %q", waiting.Logs)
	}
}
