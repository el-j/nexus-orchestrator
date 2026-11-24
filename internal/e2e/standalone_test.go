//go:build e2e

package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// runBin runs one of the built binaries with extra environment and returns what it printed and its exit
// code (-1 when it could not start).
func runBin(t *testing.T, name string, env []string, dir string, stdin string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(bin(name), args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(), env...)
	cmd.Stdin = strings.NewReader(stdin)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	if err == nil {
		return out.String(), 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return out.String(), ee.ExitCode()
	}
	t.Fatalf("%s: %v", name, err)
	return "", -1
}

// nexus-brain works on its own against a database file: it ingests a note, finds it by search, reports the
// project's status and lists its entries, and as an MCP server over stdio it answers tools/list.
func TestNexusBrainStandaloneIngestsSearchesAndServesMCP(t *testing.T) {
	buildBinaries(t)
	project, db := t.TempDir(), filepath.Join(t.TempDir(), "brain.db")
	mustWrite(t, filepath.Join(project, "NOTES.md"), "## Architecture\nThe zebra scheduler owns retries.\n")

	if out, code := runBin(t, "nexus-brain", nil, project, "", "--db", db, "ingest", "--project", project, "--file", "NOTES.md"); code != 0 || !strings.Contains(out, "Ingested 1 knowledge sections") {
		t.Fatalf("ingest: exit %d\n%s", code, out)
	}
	if out, code := runBin(t, "nexus-brain", nil, project, "", "--db", db, "search", "--project", project, "--query", "zebra"); code != 0 || !contains(out, "zebra scheduler", "architecture") {
		t.Errorf("search: exit %d\n%s", code, out)
	}
	if out, code := runBin(t, "nexus-brain", nil, project, "", "--db", db, "status", "--project", project); code != 0 || !contains(out, `"entryCount": 1`) {
		t.Errorf("status: exit %d\n%s", code, out)
	}
	if out, code := runBin(t, "nexus-brain", nil, project, "", "--db", db, "list", "--project", project); code != 0 || !strings.Contains(out, "Architecture (NOTES.md)") {
		t.Errorf("list: exit %d\n%s", code, out)
	}
	out, code := runBin(t, "nexus-brain", nil, project, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n", "--db", db, "mcp")
	if code != 0 || !contains(out, "search_knowledge", "get_onboarding_context") {
		t.Errorf("mcp tools/list: exit %d\n%s", code, out)
	}
}

// nexus-runner is the verification gate for a pipeline: a passing command exits 0, a failing one exits
// non-zero and says why, and inspect reports a file over the size limit.
func TestNexusRunnerGatesOnTheVerificationCommandAndTheInvariants(t *testing.T) {
	buildBinaries(t)
	project := t.TempDir()
	if out, code := runBin(t, "nexus-runner", nil, project, "", "verify", "--cmd", "true"); code != 0 || !strings.Contains(out, "All verification checks passed") {
		t.Errorf("verify true: exit %d\n%s", code, out)
	}
	if out, code := runBin(t, "nexus-runner", nil, project, "", "verify", "--cmd", "exit 3"); code == 0 || !strings.Contains(out, "exited with code 3") {
		t.Errorf("verify exit 3: exit %d\n%s", code, out)
	}
	mustWrite(t, filepath.Join(project, "big.go"), strings.Repeat("var x = 1\n", 600))
	out, code := runBin(t, "nexus-runner", nil, project, "", "inspect", "--project", project)
	if code == 0 || !contains(out, "max_source_lines", "big.go") {
		t.Errorf("inspect: exit %d\n%s", code, out)
	}
}

// nexus-router serves the OpenAI-compatible API on its own port in front of the providers it finds; the
// same providers answer a prompt given on the command line.
func TestNexusRouterServesTheOpenAIAPIAndAnswersAPrompt(t *testing.T) {
	buildBinaries(t)
	llm := newFakeOllama(t)
	llm.reply = func(string) string { return "routed answer" }
	env := []string{"NEXUS_OLLAMA_URL=" + llm.URL, "NEXUS_LMSTUDIO_URL=http://127.0.0.1:1", "HOME=" + t.TempDir()}

	if out, code := runBin(t, "nexus-router", env, t.TempDir(), "", "prompt", "say something"); code != 0 || !strings.Contains(out, "routed answer") {
		t.Errorf("prompt: exit %d\n%s", code, out)
	}

	addr := freeAddr(t)
	host, port, _ := strings.Cut(addr, ":")
	cmd := exec.Command(bin("nexus-router"), "serve", "--host", host, "--port", port)
	cmd.Env = append(cmd.Environ(), env...)
	var logs bytes.Buffer
	cmd.Stdout, cmd.Stderr = &logs, &logs
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })

	var body string
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		b, _ := json.Marshal(map[string]any{"model": "coder", "messages": []map[string]string{{"role": "user", "content": "hi"}}})
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://"+addr+"/v1/chat/completions", bytes.NewReader(b))
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			var buf bytes.Buffer
			_, _ = buf.ReadFrom(resp.Body)
			_ = resp.Body.Close()
			body = buf.String()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !strings.Contains(body, "routed answer") {
		t.Errorf("chat through nexus-router: %q\n%s", body, logs.String())
	}
}

// `nexus-cli orchestrate` carries one instruction through the daemon to a verified file, however long the
// model takes to answer: it waits for the task to settle instead of reporting it half done.
func TestOrchestrateWaitsForASlowModelAndReportsTheVerifiedResult(t *testing.T) {
	skipOnWindows(t)
	llm := newFakeOllama(t)
	// With a verification command the pipeline runs the agent loop: the model writes the file, then
	// finishes. The first answer takes longer than a short poll would wait.
	llm.chat = func(turn int, _ string) string {
		if turn == 1 {
			time.Sleep(4 * time.Second)
			return "```nexus-tool\n{\"tool\":\"write_file\",\"path\":\"hello.go\",\"content\":\"package main\\n\\nfunc Hello() {}\\n\"}\n```"
		}
		return "```nexus-tool\n{\"tool\":\"finish\",\"summary\":\"wrote hello.go\"}\n```"
	}
	d := startDaemon(t, llm, daemonOpts{})
	d.waitHealthy()
	project := t.TempDir()

	out, code := runBin(t, "nexus-cli", []string{"NEXUS_ADDR=" + d.http}, project, "",
		"orchestrate", "--project", project, "--target", "hello.go", "--verify", "grep -q 'func Hello' hello.go", "write Hello")
	if code != 0 {
		t.Fatalf("orchestrate: exit %d\n%s\n%s", code, out, d.logs())
	}
	if got := mustRead(t, filepath.Join(project, "hello.go")); !contains(got, "func Hello") {
		t.Errorf("hello.go = %q", got)
	}
	if !strings.Contains(strings.ToUpper(out), "COMPLETED") {
		t.Errorf("the report does not say the task completed:\n%s", out)
	}
}
