package services_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

func failingVerification(t *testing.T, output string) domain.Task {
	t.Helper()
	repo := newMemRepo()
	llm := &mockLLMClient{alive: true, name: "mock-llm", code: "code"}
	runner := &mockCommandRunner{runFunc: func(context.Context, string, string) (string, error) {
		return output, errors.New("exit status 1")
	}}
	orch := services.NewOrchestrator(services.NewDiscoveryService(llm), repo, &recordingWriter{}, nil, services.WithCommandRunner(runner))
	defer orch.Stop()
	id, err := orch.SubmitTask(domain.Task{Instruction: "x", TargetFile: "a.go", ProjectPath: t.TempDir(), VerificationCommand: "check", MaxCorrectionTurns: 1})
	if err != nil {
		t.Fatal(err)
	}
	return waitStatus(t, repo, id, domain.StatusFailed, 10*time.Second)
}

// A task whose verification keeps failing was still run by Nexus: the log names the provider, so the UI
// does not claim that an agent set the status.
func TestVerificationFailureNamesTheProviderThatRanTheTask(t *testing.T) {
	task := failingVerification(t, "boom")
	if !strings.Contains(task.Logs, "failed via mock-llm") || domain.RanBy(task.Logs) != "mock-llm" {
		t.Errorf("the log does not say who ran the task: %q", task.Logs)
	}
}

// Terminal colour codes are noise in the log and in the prompt of the next correction turn.
func TestVerificationOutputHasNoTerminalEscapes(t *testing.T) {
	task := failingVerification(t, "\x1b[31m-bad\x1b(B\x1b[m\x1b[32m+good\x1b(B\x1b[m")
	if strings.Contains(task.Logs, "\x1b") || strings.Contains(task.VerificationOutput, "\x1b") {
		t.Errorf("escape codes left: %q / %q", task.Logs, task.VerificationOutput)
	}
	if !strings.Contains(task.VerificationOutput, "-bad") || !strings.Contains(task.VerificationOutput, "+good") {
		t.Errorf("text lost: %q", task.VerificationOutput)
	}
}

// When the corrections change nothing about the failure, the log says the failure may not be the task's.
func TestVerificationFailureThatNeverChangesIsFlagged(t *testing.T) {
	task := failingVerification(t, "Diff in other.rs:1")
	if !strings.Contains(task.Logs, "did not change after the correction") {
		t.Errorf("no hint in %q", task.Logs)
	}
}

// A task that failed (the model, the provider or the project's own check) can be promoted again, so the
// user retries it with the same button instead of re-creating it.
func TestFailedAndCancelledTasksCanBePromotedAgain(t *testing.T) {
	for _, st := range []domain.TaskStatus{domain.StatusFailed, domain.StatusCancelled} {
		repo := newMemRepo()
		orch := services.NewOrchestrator(services.NewDiscoveryService(), repo, &recordingWriter{}, nil)
		if err := repo.Save(context.Background(), domain.Task{ID: "r", ProjectPath: "/p", Instruction: "i", Status: st}); err != nil {
			t.Fatal(err)
		}
		res, err := orch.PromoteTask("r")
		orch.Stop()
		if err != nil || !res.Promoted {
			t.Errorf("%s: promote = %+v, %v", st, res, err)
		}
	}
}
