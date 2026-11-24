//go:build e2e

package e2e_test

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestDaemonAutoModeLifecycle verifies the Project Auto-Mode REST and CLI
// surfaces end-to-end against a real daemon process: starting a run, listing
// runs, polling a run by ID, stopping an active run, and querying status via
// nexus-cli.
func TestDaemonAutoModeLifecycle(t *testing.T) {
	ollama := newFakeOllama(t)
	ollama.reply = func(prompt string) string {
		// Return a small valid task plan when asked to plan, and agentic tool
		// calls (write_file + finish) when executing the planned task.
		if strings.Contains(prompt, "JSON array") {
			return `[{"targetFile":"hello.txt","instruction":"Write hello world"}]`
		}
		return "```nexus-tool\n{\"tool\":\"write_file\",\"path\":\"hello.txt\",\"content\":\"hello from auto-mode\\n\"}\n```\n```nexus-tool\n{\"tool\":\"finish\",\"summary\":\"wrote hello.txt\"}\n```"
	}

	d := startDaemon(t, ollama, daemonOpts{})
	d.waitHealthy()

	proj := t.TempDir()
	if err := os.WriteFile(filepath.Join(proj, "README.md"), []byte("# Demo\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Start an Auto-Mode run via POST /api/auto/start.
	var started struct {
		ID          string `json:"id"`
		ProjectPath string `json:"projectPath"`
		Phase       string `json:"phase"`
	}
	code := d.do(http.MethodPost, "/api/auto/start", map[string]any{
		"projectPath":         proj,
		"goal":                "Create hello.txt",
		"verificationCommand": "true",
	}, &started)
	if code != http.StatusCreated || started.ID == "" {
		t.Fatalf("POST /api/auto/start: HTTP %d %+v\n%s", code, started, d.logs())
	}

	// List runs via GET /api/auto.
	var listed []struct {
		ID string `json:"id"`
	}
	if code := d.do(http.MethodGet, "/api/auto?projectPath="+url.QueryEscape(proj), nil, &listed); code != http.StatusOK || len(listed) == 0 {
		t.Fatalf("GET /api/auto: HTTP %d len=%d", code, len(listed))
	}

	// Poll GET /api/auto/{id} until terminal (completed/failed/stopped).
	deadline := time.Now().Add(20 * time.Second)
	var final struct {
		ID             string `json:"id"`
		Phase          string `json:"phase"`
		CompletedTasks int    `json:"completedTasks"`
		Error          string `json:"error"`
	}
	for time.Now().Before(deadline) {
		if code := d.do(http.MethodGet, "/api/auto/"+started.ID, nil, &final); code != http.StatusOK {
			t.Fatalf("GET /api/auto/%s: HTTP %d", started.ID, code)
		}
		if final.Phase == "completed" || final.Phase == "failed" || final.Phase == "stopped" {
			break
		}
		time.Sleep(150 * time.Millisecond)
	}
	if final.Phase != "completed" {
		t.Fatalf("auto-mode run ended in phase=%q err=%q (completed=%d)\n%s", final.Phase, final.Error, final.CompletedTasks, d.logs())
	}

	// Exercise POST /api/auto/{id}/stop on a second run (with held LLM reply so it stays active).
	ollama.hold = make(chan struct{})
	defer close(ollama.hold)
	var held struct {
		ID string `json:"id"`
	}
	if code := d.do(http.MethodPost, "/api/auto/start", map[string]any{
		"projectPath": proj,
		"goal":        "Long running task",
	}, &held); code != http.StatusCreated || held.ID == "" {
		t.Fatalf("second POST /api/auto/start: HTTP %d %+v", code, held)
	}
	if code := d.do(http.MethodPost, "/api/auto/"+held.ID+"/stop", nil, nil); code != http.StatusOK && code != http.StatusNoContent {
		t.Fatalf("POST /api/auto/%s/stop: HTTP %d", held.ID, code)
	}

	// Verify nexus-cli auto status reports the runs.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin("nexus-cli"), "auto", "status", "--project", proj)
	cmd.Env = append(os.Environ(), "NEXUS_ADDR="+d.http)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("nexus-cli auto status failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), started.ID) {
		t.Errorf("nexus-cli auto status output missing run id %s:\n%s", started.ID, out)
	}
}
