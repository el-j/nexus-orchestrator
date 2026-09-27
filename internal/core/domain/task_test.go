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
