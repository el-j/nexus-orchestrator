package cmd_runner_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/outbound/cmd_runner"
)

func TestRunner_Success(t *testing.T) {
	runner := cmd_runner.New()
	ctx := context.Background()

	output, err := runner.Run(ctx, "", "echo hello")
	if err != nil {
		t.Fatalf("expected success, got err: %v", err)
	}
	if !strings.Contains(output, "hello") {
		t.Errorf("expected output to contain 'hello', got: %q", output)
	}
}

func TestRunner_EmptyCommand(t *testing.T) {
	runner := cmd_runner.New()
	output, err := runner.Run(context.Background(), "", "   ")
	if err != nil {
		t.Fatalf("unexpected error on empty command: %v", err)
	}
	if output != "" {
		t.Errorf("expected empty output, got: %q", output)
	}
}

func TestRunner_NonZeroExit(t *testing.T) {
	runner := cmd_runner.New()
	ctx := context.Background()

	output, err := runner.Run(ctx, "", "echo 'fatal syntax error' && exit 1")
	if err == nil {
		t.Fatal("expected error on exit 1, got nil")
	}
	if !strings.Contains(output, "fatal syntax error") {
		t.Errorf("expected output to contain 'fatal syntax error', got: %q", output)
	}
}

func TestRunner_WorkingDirectory(t *testing.T) {
	tempDir := t.TempDir()
	runner := cmd_runner.New()

	output, err := runner.Run(context.Background(), tempDir, "pwd")
	if err != nil {
		t.Fatalf("pwd failed: %v", err)
	}

	cleanedOutput := filepath.Clean(strings.TrimSpace(output))
	cleanedTemp := filepath.Clean(tempDir)
	if cleanedOutput != cleanedTemp {
		// on macOS /var vs /private/var symlink
		if !strings.HasSuffix(cleanedTemp, cleanedOutput) && !strings.HasSuffix(cleanedOutput, cleanedTemp) {
			t.Errorf("expected working directory %q, got: %q", cleanedTemp, cleanedOutput)
		}
	}
}

func TestRunner_Timeout(t *testing.T) {
	runner := cmd_runner.New(cmd_runner.WithDefaultTimeout(100 * time.Millisecond))
	ctx := context.Background()

	_, err := runner.Run(ctx, "", "sleep 2")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}
