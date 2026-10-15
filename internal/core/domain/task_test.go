package domain

import (
	"testing"
	"time"
)

func TestTaskComputeDuration(t *testing.T) {
	now := time.Now()
	task := Task{
		CreatedAt: now.Add(-5 * time.Second),
		UpdatedAt: now,
	}
	task.ComputeDuration()
	if task.DurationMs < 4900 || task.DurationMs > 5100 {
		t.Errorf("expected ~5000ms, got %d", task.DurationMs)
	}
}

func TestTaskComputeDuration_ZeroCreatedAt(t *testing.T) {
	task := Task{
		UpdatedAt: time.Now(),
	}
	task.ComputeDuration()
	if task.DurationMs != 0 {
		t.Errorf("expected 0ms for zero CreatedAt, got %d", task.DurationMs)
	}
}

func TestTaskComputeDuration_UpdatedBeforeCreated(t *testing.T) {
	now := time.Now()
	task := Task{
		CreatedAt: now,
		UpdatedAt: now.Add(-1 * time.Second),
	}
	task.ComputeDuration()
	if task.DurationMs != 0 {
		t.Errorf("expected 0ms when UpdatedAt < CreatedAt, got %d", task.DurationMs)
	}
}

func TestTaskIsExecutable(t *testing.T) {
	tests := []struct {
		status TaskStatus
		want   bool
	}{
		{StatusQueued, true},
		{StatusProcessing, false},
		{StatusCompleted, false},
		{StatusDraft, false},
	}
	for _, tt := range tests {
		task := Task{Status: tt.status}
		if got := task.IsExecutable(); got != tt.want {
			t.Errorf("IsExecutable() for status %q = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestTaskVerificationFields(t *testing.T) {
	task := Task{
		ID:                  "task-123",
		VerificationCommand: "go test ./...",
		MaxCorrectionTurns:  3,
		VerificationOutput:  "PASS",
	}
	if task.VerificationCommand != "go test ./..." {
		t.Errorf("unexpected command: %s", task.VerificationCommand)
	}
	if task.MaxCorrectionTurns != 3 {
		t.Errorf("unexpected max turns: %d", task.MaxCorrectionTurns)
	}
	if task.VerificationOutput != "PASS" {
		t.Errorf("unexpected output: %s", task.VerificationOutput)
	}
}

// Whether Nexus itself ran a task is told by the log the engine writes ("completed via X at ...",
// "failed via X: ..."). A task an agent marked done by hand has no such line.
func TestRanBy(t *testing.T) {
	cases := map[string]string{
		"completed via Claude (subscription) at 2026-10-15T08:18:10Z":                                                         "Claude (subscription)",
		"first line\nfailed via Ollama: connection refused":                                                                   "Ollama",
		"verification passed (go test ./...) — completed via LM Studio at 2026-10-15T08:18:10Z":                               "LM Studio",
		"verification passed on correction turn 2/3 (npm test) — completed via vscode-copilot-gpt-4o at 2026-10-15T08:18:10Z": "vscode-copilot-gpt-4o",
		"TASK-001 completed: Landing claims corrected":                                                                        "",
		"":            "",
		"failed via ": "",
		"the previous attempt completed via X is not this one but is read": "X",
	}
	for logs, want := range cases {
		if got := RanBy(logs); got != want {
			t.Errorf("RanBy(%q) = %q, want %q", logs, got, want)
		}
	}
}
