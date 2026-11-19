package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMain_Run_Help(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 for --help, got %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "nexus-router") {
		t.Fatalf("expected help output to mention nexus-router, got: %s", stdout.String())
	}
}

func TestMain_Run_Version(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"--version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 for --version, got %d", code)
	}
	if !strings.Contains(stdout.String(), "version") {
		t.Fatalf("expected version output, got: %s", stdout.String())
	}
}

func TestMain_Run_StatusAndRouting(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"status"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected status code 0, got %d", code)
	}

	stdout.Reset()
	code = run(context.Background(), []string{"routing", "--role", "coder"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected routing code 0, got %d", code)
	}

	stdout.Reset()
	code = run(context.Background(), []string{"providers"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected providers code 0, got %d", code)
	}
}

func TestMain_Run_WithDaemonAddr(t *testing.T) {
	// Create mock daemon server responding to GET /api/providers
	mockDaemon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/providers" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"name":"MockRemote","active":true,"baseURL":"http://remote"}]`))
			return
		}
		http.NotFound(w, r)
	}))
	defer mockDaemon.Close()

	t.Setenv("NEXUS_ADDR", mockDaemon.URL)
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"status"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected status 0 with mock daemon, got %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "MockRemote") {
		t.Fatalf("expected output to mention MockRemote, got: %s", stdout.String())
	}
}

func TestMain_Run_Failure(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"invalid-subcommand"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1 for invalid subcommand, got %d", code)
	}
	if stderr.Len() == 0 {
		t.Fatal("expected error message on stderr")
	}
}

func TestMain_DaemonAddr(t *testing.T) {
	t.Setenv("NEXUS_ADDR", "")
	if addr := daemonAddr(); addr != "" {
		t.Fatalf("expected empty addr when NEXUS_ADDR unset, got %q", addr)
	}

	t.Setenv("NEXUS_ADDR", "http://custom:1234")
	if addr := daemonAddr(); addr != "http://custom:1234" {
		t.Fatalf("expected custom addr, got %q", addr)
	}
}

func TestMain_DefaultAddrConstant(t *testing.T) {
	if defaultAddr != "http://127.0.0.1:63987" {
		t.Fatalf("unexpected defaultAddr: %s", defaultAddr)
	}
}
