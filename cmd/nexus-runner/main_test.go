package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/ports"
)

func TestMain_Run_Help(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 for --help, got %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "nexus-runner") {
		t.Fatalf("expected help output to mention nexus-runner, got: %s", stdout.String())
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

func TestMain_Run_Verify(t *testing.T) {
	var stdout, stderr bytes.Buffer
	tmpDir := t.TempDir()

	code := run(context.Background(), []string{"verify", "--project", tmpDir, "--cmd", "echo ok"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected verify code 0, got %d, stderr: %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"verify", "--project", tmpDir, "--cmd", "exit 1"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected verify failure code 1, got %d", code)
	}
}

func TestMain_Run_Inspect(t *testing.T) {
	var stdout, stderr bytes.Buffer
	tmpDir := t.TempDir()

	code := run(context.Background(), []string{"inspect", "--project", tmpDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected inspect code 0, got %d, stderr: %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"inspect", "--project", tmpDir, "--policy", filepath.Join(tmpDir, "missing.json")}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected inspect failure code 1 on missing policy, got %d", code)
	}
}

func TestMain_Run_Execute(t *testing.T) {
	var stdout, stderr bytes.Buffer
	tmpDir := t.TempDir()

	code := run(context.Background(), []string{"execute", "--project", tmpDir, "--cmd", "echo ok"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected execute code 0, got %d, stderr: %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"execute", "--project", tmpDir, "--cmd", "exit 1", "--heal=false"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected execute failure code 1, got %d", code)
	}
}

func TestBuildGenerator(t *testing.T) {
	// Exercise buildGenerator directly
	_ = buildGenerator()

	// Empty providers list returns nil
	origFn := buildProvidersFn
	buildProvidersFn = func() []ports.LLMClient {
		return nil
	}
	if gen := buildGenerator(); gen != nil {
		t.Fatal("expected nil generator when no providers")
	}
	buildProvidersFn = origFn

	// Set an env var so bootstrap.BuildProviders finds an active provider
	t.Setenv("OLLAMA_BASE_URL", "http://localhost:11434")
	gen := buildGenerator()
	if gen == nil {
		t.Fatal("expected generator to be constructed when provider is present")
	}
	if gen.ProviderName() == "" {
		t.Fatal("expected provider name to be populated")
	}
}
