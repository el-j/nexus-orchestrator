//go:build e2e

package e2e_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func skipOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("verification commands run through sh")
	}
}

// A queued task is executed by the daemon against the (fake) provider, the
// generated code is written into the project, and the verification gate runs.
func TestDaemonRunsATaskThroughTheVerificationGate(t *testing.T) {
	skipOnWindows(t)
	llm := newFakeOllama(t)
	llm.reply = func(string) string { return "```go\npackage main\n\nfunc Hello() string { return \"hi\" }\n```" }
	d := startDaemon(t, llm, daemonOpts{})
	d.waitHealthy()

	project := t.TempDir()
	var created map[string]string
	if code := d.do(http.MethodPost, "/api/tasks", map[string]any{
		"projectPath": project, "targetFile": "hello.go", "instruction": "write Hello",
		"verificationCommand": "grep -q 'func Hello' hello.go", "maxCorrectionTurns": 1,
	}, &created); code != http.StatusCreated {
		t.Fatalf("create task: HTTP %d\n%s", code, d.logs())
	}

	done := d.waitStatus(created["task_id"], "COMPLETED")
	if got := mustRead(t, filepath.Join(project, "hello.go")); !contains(got, "package main", "func Hello") {
		t.Errorf("generated file = %q", got)
	}
	if done.Logs == "" {
		t.Error("a finished task should carry its log")
	}
}

// A failing verification command makes the daemon ask the model to correct itself.
func TestDaemonSelfHealsAfterAFailedVerification(t *testing.T) {
	skipOnWindows(t)
	llm := newFakeOllama(t)
	llm.chat = func(turn int, _ string) string {
		if turn == 1 {
			return "```go\npackage main\n\nfunc Broken() {}\n```"
		}
		return "```go\npackage main\n\nfunc Fixed() {}\n```"
	}
	llm.generate = func(prompt string) string { return llm.chat(1, prompt) }
	d := startDaemon(t, llm, daemonOpts{})
	d.waitHealthy()

	project := t.TempDir()
	var created map[string]string
	d.do(http.MethodPost, "/api/tasks", map[string]any{
		"projectPath": project, "targetFile": "x.go", "instruction": "write Fixed",
		"verificationCommand": "grep -q 'func Fixed' x.go", "maxCorrectionTurns": 2,
	}, &created)

	d.waitStatus(created["task_id"], "COMPLETED")
	if got := mustRead(t, filepath.Join(project, "x.go")); !contains(got, "func Fixed") {
		t.Errorf("the corrected file was not written: %q", got)
	}
}

// An agentic task reads and writes several project files over multiple turns.
func TestDaemonAgenticTaskEditsSeveralFiles(t *testing.T) {
	skipOnWindows(t)
	project := t.TempDir()
	mustWrite(t, filepath.Join(project, "notes.txt"), "remember the milk")

	llm := newFakeOllama(t)
	llm.chat = func(turn int, last string) string {
		switch turn {
		case 1:
			return "```nexus-tool\n{\"tool\":\"read_file\",\"path\":\"notes.txt\"}\n```"
		case 2:
			if !strings.Contains(last, "remember the milk") {
				return "```nexus-tool\n{\"tool\":\"finish\",\"summary\":\"never saw the file\"}\n```"
			}
			return "```nexus-tool\n{\"tool\":\"write_file\",\"path\":\"out/a.txt\",\"content\":\"alpha\"}\n```\n" +
				"```nexus-tool\n{\"tool\":\"write_file\",\"path\":\"out/b.txt\",\"content\":\"beta\"}\n```"
		default:
			return "```nexus-tool\n{\"tool\":\"finish\",\"summary\":\"wrote two files\"}\n```"
		}
	}
	d := startDaemon(t, llm, daemonOpts{})
	d.waitHealthy()

	var created map[string]string
	if code := d.do(http.MethodPost, "/api/tasks", map[string]any{
		"projectPath": project, "instruction": "copy the note into two files", "agentic": true, "maxAgentTurns": 5,
	}, &created); code != http.StatusCreated {
		t.Fatalf("create task: HTTP %d\n%s", code, d.logs())
	}
	d.waitStatus(created["task_id"], "COMPLETED")

	if got := mustRead(t, filepath.Join(project, "out", "a.txt")); got != "alpha" {
		t.Errorf("a.txt = %q", got)
	}
	if got := mustRead(t, filepath.Join(project, "out", "b.txt")); got != "beta" {
		t.Errorf("b.txt = %q", got)
	}
}

// Tasks, backlog drafts and their routing fields survive a restart.
func TestDaemonPersistsAcrossRestart(t *testing.T) {
	home, db := t.TempDir(), filepath.Join(t.TempDir(), "keep.db")
	first := startDaemon(t, nil, daemonOpts{db: db, home: home})
	first.waitHealthy()

	var draft map[string]string
	if code := first.do(http.MethodPost, "/api/tasks/draft", map[string]any{
		"projectPath": "/work/persist", "instruction": "remember me", "role": "architect", "providerHint": "Ollama",
	}, &draft); code != http.StatusCreated {
		t.Fatalf("create draft: HTTP %d", code)
	}
	if err := first.stop(); err != nil {
		t.Fatalf("first daemon should exit cleanly, got %v\n%s", err, first.logs())
	}

	second := startDaemon(t, nil, daemonOpts{db: db, home: home})
	second.waitHealthy()
	var got struct {
		ID, Status, Role, ProviderHint string
	}
	if code := second.do(http.MethodGet, "/api/tasks/"+draft["id"], nil, &got); code != http.StatusOK {
		t.Fatalf("task lost across restart: HTTP %d", code)
	}
	if got.Status != "DRAFT" || got.Role != "architect" || got.ProviderHint != "Ollama" {
		t.Errorf("restored task = %+v", got)
	}
}

func TestDaemonShutsDownGracefully(t *testing.T) {
	d := startDaemon(t, nil, daemonOpts{})
	d.waitHealthy()
	start := time.Now()
	if err := d.stop(); err != nil {
		t.Fatalf("exit after SIGTERM: %v\n%s", err, d.logs())
	}
	if time.Since(start) > 15*time.Second {
		t.Errorf("shutdown took %s", time.Since(start))
	}
	if resp, err := http.Get(d.http + "/api/health"); err == nil {
		_ = resp.Body.Close()
		t.Error("the HTTP port should be closed after shutdown")
	}
}

func TestDaemonRefusesAPortThatIsTaken(t *testing.T) {
	first := startDaemon(t, nil, daemonOpts{})
	first.waitHealthy()

	clash := startDaemon(t, nil, daemonOpts{httpAddr: strings.TrimPrefix(first.http, "http://")})
	select {
	case err := <-clash.exited:
		clash.exited <- err
		if err == nil {
			t.Error("a second daemon on a busy port must not exit 0")
		}
	case <-time.After(20 * time.Second):
		t.Fatalf("second daemon kept running on a busy port\n%s", clash.logs())
	}
}

// Two projects on one daemon never see each other's work.
func TestDaemonIsolatesProjects(t *testing.T) {
	d := startDaemon(t, nil, daemonOpts{})
	d.waitHealthy()
	for _, p := range []string{"/work/a", "/work/b", "/work/b"} {
		d.do(http.MethodPost, "/api/tasks/draft", map[string]any{"projectPath": p, "instruction": "x"}, nil)
	}
	count := func(project string) int {
		var tasks []apiTask
		d.do(http.MethodGet, "/api/tasks/backlog?project="+project, nil, &tasks)
		return len(tasks)
	}
	if a, b := count("/work/a"), count("/work/b"); a != 1 || b != 2 {
		t.Errorf("backlog sizes a=%d b=%d, want 1 and 2", a, b)
	}
}

// Every advertised MCP tool answers over real HTTP without taking the daemon down.
func TestDaemonAnswersEveryMCPTool(t *testing.T) {
	d := startDaemon(t, nil, daemonOpts{})
	d.waitHealthy()

	if code, resp := d.rpc("initialize", map[string]any{}); code != http.StatusOK || resp["result"] == nil {
		t.Fatalf("initialize: HTTP %d %v", code, resp)
	}
	code, resp := d.rpc("tools/list", map[string]any{})
	if code != http.StatusOK {
		t.Fatalf("tools/list: HTTP %d", code)
	}
	tools, _ := resp["result"].(map[string]any)["tools"].([]any)
	if len(tools) < 41 {
		t.Fatalf("the daemon advertises %d tools, want at least 41", len(tools))
	}

	for _, raw := range tools {
		name := raw.(map[string]any)["name"].(string)
		code, resp := d.rpc("tools/call", map[string]any{"name": name, "arguments": map[string]any{}})
		if code >= 500 {
			t.Errorf("tool %s: HTTP %d", name, code)
		}
		if resp["result"] == nil && resp["error"] == nil {
			t.Errorf("tool %s: neither result nor error: %v", name, resp)
		}
	}
	if code := d.do(http.MethodGet, "/api/health", nil, nil); code != http.StatusOK {
		t.Errorf("daemon unhealthy after calling every tool: HTTP %d\n%s", code, d.logs())
	}
}

// The stdio proxy that editors launch forwards JSON-RPC to the daemon.
func TestStdioProxyForwardsToTheDaemon(t *testing.T) {
	d := startDaemon(t, nil, daemonOpts{})
	d.waitHealthy()

	cmd := exec.Command(bin("nexus-mcp-stdio"))
	cmd.Env = append(cmd.Environ(), "NEXUS_MCP_URL="+d.mcp+"/mcp")
	in, _ := cmd.StdinPipe()
	out, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill() }()

	r := bufio.NewReader(out)
	for _, method := range []string{"initialize", "tools/list"} {
		msg, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 7, "method": method, "params": map[string]any{}})
		_, _ = in.Write(append(msg, '\n'))
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("%s: read: %v", method, err)
		}
		var resp map[string]any
		if err := json.Unmarshal([]byte(line), &resp); err != nil || resp["result"] == nil {
			t.Errorf("%s: unexpected reply %q", method, line)
		}
	}
	_ = in.Close()
	if err := cmd.Wait(); err != nil {
		t.Errorf("proxy should exit 0 when its input closes: %v", err)
	}
}

func runCLI(t *testing.T, name, addr string, args ...string) string {
	t.Helper()
	cmd := exec.Command(bin(name), args...)
	cmd.Env = append(cmd.Environ(), "NEXUS_ADDR="+addr)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out.String())
	}
	return out.String()
}

// nexus-cli drafts into the backlog and reads it back.
func TestNexusCLIDraftsAndListsTheBacklog(t *testing.T) {
	d := startDaemon(t, nil, daemonOpts{})
	d.waitHealthy()

	runCLI(t, "nexus-cli", d.http, "draft", "--project", "/work/cli", "--instruction", "from the cli", "--role", "coder")
	out := runCLI(t, "nexus-cli", d.http, "backlog", "--project", "/work/cli")
	if !contains(out, "from the cli") {
		t.Errorf("backlog output missing the draft:\n%s", out)
	}
}

// nexus-submit sends a task file, waits, and exits 0 once the daemon completed it.
func TestNexusSubmitWaitsForCompletion(t *testing.T) {
	skipOnWindows(t)
	llm := newFakeOllama(t)
	llm.reply = func(string) string { return "```go\npackage main\n\nfunc Submitted() {}\n```" }
	d := startDaemon(t, llm, daemonOpts{})
	d.waitHealthy()

	project := t.TempDir()
	taskFile := filepath.Join(t.TempDir(), "TASK-1.md")
	mustWrite(t, taskFile, "write Submitted")
	out := runCLI(t, "nexus-submit", d.http, "--task-file", taskFile, "--project", project,
		"--target", "sub.go", "--verify", "grep -q Submitted sub.go", "--wait", "--timeout", "60s")
	if !contains(out, "submitted: task_id=", "COMPLETED") {
		t.Errorf("unexpected nexus-submit output:\n%s", out)
	}
	if got := mustRead(t, filepath.Join(project, "sub.go")); !contains(got, "func Submitted") {
		t.Errorf("sub.go = %q", got)
	}
}
