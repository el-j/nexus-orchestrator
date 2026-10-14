//go:build e2e

package e2e_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

const e2ePolicy = `{
  "verify": "test -f hello.go",
  "maxFileLines": 4,
  "testsFirst": true,
  "remember": true,
  "rules": ["Keep every function tiny."]
}`

func writeE2EPolicy(t *testing.T, project string) {
	t.Helper()
	mustWrite(t, filepath.Join(project, ".nexus", "policy.json"), e2ePolicy)
}

// A fresh session that only asks the daemon for its onboarding gets the project's rules.
func TestDaemonTellsAFreshSessionTheProjectRules(t *testing.T) {
	d := startDaemon(t, nil, daemonOpts{})
	d.waitHealthy()
	project := t.TempDir()
	writeE2EPolicy(t, project)

	code, resp := d.rpc("tools/call", map[string]any{
		"name": "get_onboarding_context", "arguments": map[string]any{"projectPath": project},
	})
	if code != http.StatusOK {
		t.Fatalf("get_onboarding_context: HTTP %d", code)
	}
	raw, _ := json.Marshal(resp)
	text := string(raw)
	for _, want := range []string{"Workflow rules", "test -f hello.go", "Keep every function tiny.", "Tests first"} {
		if !strings.Contains(text, want) {
			t.Errorf("onboarding lacks %q:\n%s", want, text)
		}
	}
}

// Introducing a project creates its rule file once and never overwrites it.
func TestDaemonInitCreatesTheRuleFileOnce(t *testing.T) {
	d := startDaemon(t, nil, daemonOpts{})
	d.waitHealthy()
	project := t.TempDir()
	mustWrite(t, filepath.Join(project, "go.mod"), "module demo\n")

	if code := d.do(http.MethodPost, "/api/brain/init", map[string]any{"projectPath": project}, nil); code != http.StatusOK {
		t.Fatalf("init: HTTP %d\n%s", code, d.logs())
	}
	got := mustRead(t, filepath.Join(project, ".nexus", "policy.json"))
	if !contains(got, `"verify": "go vet ./... \u0026\u0026 go test ./..."`, `"testsFirst": true`) {
		t.Errorf("default policy = %s", got)
	}
	mustWrite(t, filepath.Join(project, ".nexus", "policy.json"), `{"verify":"mine"}`)
	d.do(http.MethodPost, "/api/brain/init", map[string]any{"projectPath": project}, nil)
	if got := mustRead(t, filepath.Join(project, ".nexus", "policy.json")); got != `{"verify":"mine"}` {
		t.Errorf("init overwrote the project's own policy: %s", got)
	}
}

// A plain task is verified with the policy's command, the model is shown the rules and a
// briefing, and the outcome is remembered for the next session.
func TestDaemonAppliesPolicyToATask(t *testing.T) {
	skipOnWindows(t)
	project := t.TempDir()
	writeE2EPolicy(t, project)

	llm := newFakeOllama(t)
	var mu sync.Mutex
	var prompt string
	llm.generate = func(p string) string {
		mu.Lock()
		prompt = p
		mu.Unlock()
		return "```go\npackage main\n```"
	}
	llm.chat = func(_ int, last string) string { return llm.generate(last) }
	d := startDaemon(t, llm, daemonOpts{})
	d.waitHealthy()

	var created map[string]string
	if code := d.do(http.MethodPost, "/api/tasks", map[string]any{
		"projectPath": project, "targetFile": "hello.go", "instruction": "write the entry point",
	}, &created); code != http.StatusCreated {
		t.Fatalf("create task: HTTP %d\n%s", code, d.logs())
	}
	done := d.waitStatus(created["task_id"], "COMPLETED")
	if !strings.Contains(done.Logs, "test -f hello.go") {
		t.Errorf("the task was not verified with the policy command; logs: %q", done.Logs)
	}

	mu.Lock()
	p := prompt
	mu.Unlock()
	for _, want := range []string{"Project Onboarding", "## Workflow rules", "Keep every function tiny.", "## Task", "write the entry point"} {
		if !strings.Contains(p, want) {
			t.Errorf("the model prompt lacks %q:\n%s", want, p)
		}
	}

	var entries []struct {
		Content string `json:"content"`
		Source  string `json:"source"`
	}
	q := url.Values{"projectPath": {project}, "kind": {"learning"}}
	if code := d.do(http.MethodGet, "/api/brain/knowledge?"+q.Encode(), nil, &entries); code != http.StatusOK {
		t.Fatalf("list knowledge: HTTP %d", code)
	}
	var remembered bool
	for _, e := range entries {
		remembered = remembered || (e.Source == "nexus:task" && strings.Contains(e.Content, "Completed: write the entry point"))
	}
	if !remembered {
		t.Errorf("the finished task was not remembered: %+v", entries)
	}
}

// An agent that writes source before a test, or an oversized file, is turned back by the daemon.
func TestDaemonEnforcesTestsFirstAndFileSizeOnAnAgent(t *testing.T) {
	skipOnWindows(t)
	project := t.TempDir()
	writeE2EPolicy(t, project)

	var mu sync.Mutex
	var told []string
	llm := newFakeOllama(t)
	llm.chat = func(turn int, last string) string {
		mu.Lock()
		told = append(told, last)
		mu.Unlock()
		switch turn {
		case 1: // source first: must be refused
			return "```nexus-tool\n{\"tool\":\"write_file\",\"path\":\"hello.go\",\"content\":\"package main\\n\"}\n```"
		case 2: // test first, then an over-long source file: second must be refused
			return "```nexus-tool\n{\"tool\":\"write_file\",\"path\":\"hello_test.go\",\"content\":\"package main\\n\"}\n```\n" +
				"```nexus-tool\n{\"tool\":\"write_file\",\"path\":\"hello.go\",\"content\":\"1\\n2\\n3\\n4\\n5\\n6\\n\"}\n```"
		case 3: // within the limit
			return "```nexus-tool\n{\"tool\":\"write_file\",\"path\":\"hello.go\",\"content\":\"package main\\n\"}\n```"
		default:
			return "```nexus-tool\n{\"tool\":\"finish\",\"summary\":\"done\"}\n```"
		}
	}
	d := startDaemon(t, llm, daemonOpts{})
	d.waitHealthy()

	var created map[string]string
	if code := d.do(http.MethodPost, "/api/tasks", map[string]any{
		"projectPath": project, "instruction": "agent obeys the rules", "agentic": true, "maxAgentTurns": 6,
	}, &created); code != http.StatusCreated {
		t.Fatalf("create task: HTTP %d\n%s", code, d.logs())
	}
	d.waitStatus(created["task_id"], "COMPLETED")

	mu.Lock()
	defer mu.Unlock()
	if len(told) < 4 || !strings.Contains(told[1], "tests first") {
		t.Errorf("the agent was not told to write a test first: %q", told)
	}
	if len(told) >= 3 && !strings.Contains(told[2], "the limit is 4") {
		t.Errorf("the agent was not told about the size limit: %q", told[2])
	}
	for _, f := range []string{"hello.go", "hello_test.go"} {
		if _, err := os.Stat(filepath.Join(project, f)); err != nil {
			t.Errorf("%s was not written in the end: %v", f, err)
		}
	}
}
