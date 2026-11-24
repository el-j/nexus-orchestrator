package services_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

func waitTerminal(t *testing.T, orch *services.OrchestratorService, id string) domain.Task {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		got, err := orch.GetTask(id)
		if err == nil && (got.Status == domain.StatusCompleted || got.Status == domain.StatusFailed || got.Status == domain.StatusCancelled) {
			return got
		}
		time.Sleep(10 * time.Millisecond)
	}
	got, _ := orch.GetTask(id)
	t.Fatalf("task %s did not settle (status=%s)", id, got.Status)
	return got
}

func TestOrchestrator_DependencyChain_SuccessAndFailureCascade(t *testing.T) {
	repo, err := repo_sqlite.New(filepath.Join(t.TempDir(), "chain.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	llm := &scriptLLM{
		name:    "ChainLLM",
		alive:   true,
		replies: []string{"package a\n", "package b\n"},
	}
	w := newMemFS(map[string]string{})
	disc := services.NewDiscoveryService(llm)
	orch := services.NewOrchestrator(disc, repo, w, nil)
	t.Cleanup(orch.Stop)

	proj := t.TempDir()
	id1, err := orch.SubmitTask(domain.Task{ProjectPath: proj, TargetFile: "a.go", Instruction: "step 1"})
	if err != nil {
		t.Fatal(err)
	}
	id2, err := orch.SubmitTask(domain.Task{ProjectPath: proj, TargetFile: "b.go", Instruction: "step 2", DependsOn: []string{id1}})
	if err != nil {
		t.Fatal(err)
	}

	got1 := waitTerminal(t, orch, id1)
	got2 := waitTerminal(t, orch, id2)
	if got1.Status != domain.StatusCompleted || got2.Status != domain.StatusCompleted {
		t.Fatalf("expected both COMPLETED, got %s and %s", got1.Status, got2.Status)
	}

	// Failure cascade: id3 fails -> id4 -> id5 both cascade-fail.
	llm.errs = []error{errors.New("boom"), errors.New("boom"), errors.New("boom"), errors.New("boom")}
	llm.replies = nil
	llm.calls = 0
	id3, err := orch.SubmitTask(domain.Task{ProjectPath: proj, TargetFile: "c.go", Instruction: "step 3 fails"})
	if err != nil {
		t.Fatal(err)
	}
	id4, err := orch.SubmitTask(domain.Task{ProjectPath: proj, TargetFile: "d.go", Instruction: "step 4", DependsOn: []string{id3}})
	if err != nil {
		t.Fatal(err)
	}
	id5, err := orch.SubmitTask(domain.Task{ProjectPath: proj, TargetFile: "e.go", Instruction: "step 5", DependsOn: []string{id4}})
	if err != nil {
		t.Fatal(err)
	}

	got3 := waitTerminal(t, orch, id3)
	got4 := waitTerminal(t, orch, id4)
	got5 := waitTerminal(t, orch, id5)
	if got3.Status != domain.StatusFailed || got4.Status != domain.StatusFailed || got5.Status != domain.StatusFailed {
		t.Fatalf("expected cascade failure, got %s / %s / %s", got3.Status, got4.Status, got5.Status)
	}
	if !strings.Contains(got4.Logs, "dependency "+id3) {
		t.Errorf("expected dependency failure reason in logs, got %q", got4.Logs)
	}

	pending, err := repo.GetPending(context.Background())
	if err != nil || len(pending) != 0 {
		t.Fatalf("expected 0 pending tasks, got %d err=%v", len(pending), err)
	}
}
