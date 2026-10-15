package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/cli"
	"nexus-orchestrator/internal/adapters/outbound/httpapi_client"
)

func TestNexusCLI_RootCmdConstruction(t *testing.T) {
	orch := httpapi_client.NewClient("http://127.0.0.1:63987")
	brain := httpapi_client.NewBrainClient("http://127.0.0.1:63987")

	root := cli.NewRootCmd(orch, brain)
	root.Version = "0.10.0 (test-commit 2026-09-25)"

	if root.Use != "nexus" {
		t.Errorf("expected root use 'nexus', got %q", root.Use)
	}

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("root.Execute --help: %v", err)
	}

	helpOut := buf.String()
	subcommands := []string{"tasks", "providers", "brain", "status"}
	for _, sub := range subcommands {
		if !strings.Contains(helpOut, sub) {
			t.Errorf("expected help output to mention subcommand %q, got:\n%s", sub, helpOut)
		}
	}
}

func TestNexusCLI_Version(t *testing.T) {
	orch := httpapi_client.NewClient("http://127.0.0.1:63987")
	brain := httpapi_client.NewBrainClient("http://127.0.0.1:63987")

	root := cli.NewRootCmd(orch, brain)
	root.Version = "v0.10.0 (commit-abc 2026-09-25)"

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"--version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("root.Execute --version: %v", err)
	}

	if !strings.Contains(buf.String(), "v0.10.0") {
		t.Errorf("expected version output to contain 'v0.10.0', got %q", buf.String())
	}
}

func TestRun_VersionAndHelpExitZero(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"--version"}, &out, &errb, defaultAddr); code != 0 {
		t.Fatalf("--version: code=%d stderr=%q", code, errb.String())
	}
	if !strings.Contains(out.String(), "dev (unknown unknown)") {
		t.Errorf("version output = %q", out.String())
	}

	out.Reset()
	if code := run([]string{"--help"}, &out, &errb, defaultAddr); code != 0 || !strings.Contains(out.String(), "providers") {
		t.Fatalf("--help: code=%d out=%q", code, out.String())
	}
}

func TestRun_UnknownCommandFails(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"definitely-not-a-command"}, &out, &errb, defaultAddr); code != 1 {
		t.Fatalf("code=%d, want 1", code)
	}
	if !strings.Contains(errb.String(), "unknown command") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestRun_TalksToTheConfiguredDaemon(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var out, errb bytes.Buffer
	if code := run([]string{"queue", "list"}, &out, &errb, srv.URL); code != 0 {
		t.Fatalf("queue list against fake daemon: code=%d stderr=%q", code, errb.String())
	}
	if !strings.Contains(out.String(), "Queue is empty.") {
		t.Errorf("stdout = %q, want the empty-queue message routed through cmd output", out.String())
	}
}

func TestRun_FailsWhenDaemonUnreachable(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"queue", "list"}, &out, &errb, "http://127.0.0.1:1"); code != 1 {
		t.Fatalf("code=%d, want 1 (stderr=%q)", code, errb.String())
	}
	if n := strings.Count(errb.String(), "queue list"); n != 1 {
		t.Errorf("error should be printed exactly once, got %d times:\n%s", n, errb.String())
	}
}

func TestDaemonAddr(t *testing.T) {
	t.Setenv("NEXUS_ADDR", "")
	if got := daemonAddr(); got != defaultAddr {
		t.Errorf("default = %q", got)
	}
	t.Setenv("NEXUS_ADDR", "http://example:1234")
	if got := daemonAddr(); got != "http://example:1234" {
		t.Errorf("override = %q", got)
	}
}
