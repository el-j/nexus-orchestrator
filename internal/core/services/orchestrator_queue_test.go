package services_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

func TestOrchestrator_SubmitTask_AssignsIDAndQueues(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService() // no providers — worker won't process
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	task := domain.Task{
		ProjectPath: "/tmp/project",
		TargetFile:  "main.go",
		Instruction: "write hello world",
	}

	id, err := orch.SubmitTask(task)
	if err != nil {
		t.Fatalf("SubmitTask returned error: %v", err)
	}
	if id == "" {
		t.Fatal("expected a non-empty task ID")
	}

	saved, err := repo.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("task not persisted: %v", err)
	}
	if saved.Status != domain.StatusQueued {
		t.Errorf("expected status QUEUED, got %s", saved.Status)
	}
}

func TestOrchestrator_GetQueue_ReturnsPendingTasks(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	for i, status := range []domain.TaskStatus{domain.StatusQueued, domain.StatusProcessing, domain.StatusQueued} {
		err := repo.Save(context.Background(), domain.Task{
			ID:          fmt.Sprintf("pending-%d", i),
			Instruction: "task",
			Status:      status,
		})
		if err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	tasks, err := orch.GetQueue()
	if err != nil {
		t.Fatalf("GetQueue: %v", err)
	}
	if len(tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(tasks))
	}
}

func TestOrchestrator_CancelTask_RemovesFromQueue(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, _ := orch.SubmitTask(domain.Task{Instruction: "to cancel"})

	if err := orch.CancelTask(id); err != nil {
		t.Fatalf("CancelTask: %v", err)
	}

	tasks, _ := orch.GetQueue()
	for _, task := range tasks {
		if task.ID == id {
			t.Error("cancelled task still present in queue")
		}
	}
}

func TestOrchestrator_WorkerProcessesTask(t *testing.T) {
	repo := newMemRepo()
	llm := &mockLLMClient{alive: true, name: "mock", code: "package main"}
	discovery := services.NewDiscoveryService(llm)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, _ := orch.SubmitTask(domain.Task{Instruction: "write code"})

	// Allow the worker loop (2 s ticker) to fire
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(300 * time.Millisecond)
		saved, _ := repo.GetByID(context.Background(), id)
		if saved.Status == domain.StatusCompleted {
			return // success
		}
	}
	t.Errorf("task did not reach COMPLETED within timeout; last status from repo")
}

func TestOrchestrator_PreFlight_TooLarge(t *testing.T) {
	repo := newMemRepo()
	// contextLimit=10 means limit-512 = -502; any non-empty instruction overflows
	llm := &mockLLMClient{alive: true, name: "mock", code: "ok", contextLimit: 10}
	discovery := services.NewDiscoveryService(llm)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, err := orch.SubmitTask(domain.Task{
		Instruction: strings.Repeat("x", 200), // ~50 tokens
	})
	if err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
		saved, _ := repo.GetByID(context.Background(), id)
		if saved.Status == domain.StatusTooLarge {
			if !strings.Contains(saved.Logs, "context too large") {
				t.Errorf("expected 'context too large' in Logs, got: %s", saved.Logs)
			}
			return
		}
		if saved.Status == domain.StatusCompleted || saved.Status == domain.StatusFailed {
			t.Fatalf("expected StatusTooLarge but got %s (logs: %s)", saved.Status, saved.Logs)
		}
	}
	t.Fatal("task did not reach StatusTooLarge within timeout")
}

func TestOrchestrator_PreFlight_NoLimitSkipsCheck(t *testing.T) {
	repo := newMemRepo()
	// contextLimit=0 means no pre-flight check; large instruction must still complete
	llm := &mockLLMClient{alive: true, name: "mock", code: "package main", contextLimit: 0}
	discovery := services.NewDiscoveryService(llm)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, _ := orch.SubmitTask(domain.Task{Instruction: strings.Repeat("x", 10000)})
	waitCompleted(t, repo, id, 10*time.Second)
}

func TestOrchestrator_PreFlight_WithinLimit_Completes(t *testing.T) {
	repo := newMemRepo()
	// contextLimit=8192 — "short instruction" is well under 8192-512=7680 tokens
	llm := &mockLLMClient{alive: true, name: "mock", code: "ok", contextLimit: 8192}
	discovery := services.NewDiscoveryService(llm)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, _ := orch.SubmitTask(domain.Task{Instruction: "short instruction"})
	waitCompleted(t, repo, id, 10*time.Second)
}

func TestOrchestrator_StatusNoProvider_WhenModelUnavailable(t *testing.T) {
	repo := newMemRepo()
	// Provider is alive but only has "llama3" — "gpt-4o" will not be found.
	llm := &mockLLMClient{alive: true, name: "Ollama", activeModel: "llama3", models: []string{"llama3"}}
	discovery := services.NewDiscoveryService(llm)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, err := orch.SubmitTask(domain.Task{
		Instruction: "do something",
		ModelID:     "gpt-4o",
	})
	if err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
		saved, _ := repo.GetByID(context.Background(), id)
		if saved.Status == domain.StatusNoProvider {
			return // success
		}
		if saved.Status == domain.StatusCompleted || saved.Status == domain.StatusFailed {
			t.Fatalf("expected StatusNoProvider but got %s", saved.Status)
		}
	}
	t.Fatal("task did not reach StatusNoProvider within timeout")
}

func TestOrchestrator_CancelTask_UnknownReturnsNotFound(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	err := orch.CancelTask("unknown-task-id")
	if err == nil {
		t.Fatal("expected error for unknown task, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected domain.ErrNotFound, got: %v", err)
	}
}

func TestOrchestrator_Stop_ThenSubmitReturnsError(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)

	orch.Stop()

	_, err := orch.SubmitTask(domain.Task{Instruction: "should fail"})
	if err == nil {
		t.Fatal("expected error after Stop, got nil")
	}
	if !strings.Contains(err.Error(), "stopped") {
		t.Errorf("expected 'stopped' in error message, got: %v", err)
	}
}

func TestOrchestrator_Stop_Idempotent(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)

	// Calling Stop() multiple times must not panic.
	orch.Stop()
	orch.Stop()
	orch.Stop()
}

func TestOrchestrator_CommandPlan_Succeeds(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, err := orch.SubmitTask(domain.Task{
		ProjectPath: "/proj/cmd",
		Instruction: "create plan",
		Command:     domain.CommandPlan,
	})
	if err != nil {
		t.Fatalf("SubmitTask with CommandPlan: %v", err)
	}
	saved, err := repo.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if saved.Command != domain.CommandPlan {
		t.Errorf("expected command %q, got %q", domain.CommandPlan, saved.Command)
	}
}

func TestOrchestrator_CommandExecute_NoPlan_ReturnsErrNoPlan(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	_, err := orch.SubmitTask(domain.Task{
		ProjectPath: "/proj/no-plan",
		Instruction: "execute now",
		Command:     domain.CommandExecute,
	})
	if err == nil {
		t.Fatal("expected ErrNoPlan, got nil")
	}
	if !errors.Is(err, domain.ErrNoPlan) {
		t.Errorf("expected ErrNoPlan, got: %v", err)
	}
}

func TestOrchestrator_CommandExecute_WithPlan_Succeeds(t *testing.T) {
	repo := newMemRepo()
	llm := &mockLLMClient{alive: true, name: "mock", code: "done"}
	discovery := services.NewDiscoveryService(llm)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	// Submit a plan task first
	planID, err := orch.SubmitTask(domain.Task{
		ProjectPath: "/proj/has-plan",
		Instruction: "plan the work",
		Command:     domain.CommandPlan,
	})
	if err != nil {
		t.Fatalf("SubmitTask plan: %v", err)
	}

	// Wait for plan to complete
	waitCompleted(t, repo, planID, 10*time.Second)

	// Now submit execute — should succeed
	execID, err := orch.SubmitTask(domain.Task{
		ProjectPath: "/proj/has-plan",
		Instruction: "do the work",
		Command:     domain.CommandExecute,
	})
	if err != nil {
		t.Fatalf("SubmitTask execute: %v", err)
	}
	if execID == "" {
		t.Fatal("expected non-empty task ID")
	}
}

func TestOrchestrator_InvalidCommand_ReturnsError(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	_, err := orch.SubmitTask(domain.Task{
		ProjectPath: "/proj/bad",
		Instruction: "bad command",
		Command:     domain.CommandType("bogus"),
	})
	if err == nil {
		t.Fatal("expected error for invalid command, got nil")
	}
	if !strings.Contains(err.Error(), "invalid command type") {
		t.Errorf("expected 'invalid command type' in error, got: %v", err)
	}
}

func TestOrchestrator_EmptyCommand_DefaultsToAuto(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, err := orch.SubmitTask(domain.Task{
		ProjectPath: "/proj/default",
		Instruction: "auto route",
		Command:     "",
	})
	if err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}
	saved, err := repo.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if saved.Command != domain.CommandAuto {
		t.Errorf("expected command %q, got %q", domain.CommandAuto, saved.Command)
	}
}

func TestOrchestrator_CommandExecute_PlanNotCompleted_ReturnsErrNoPlan(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	// Submit a plan task but DON'T wait for it to complete — it stays QUEUED
	_, err := orch.SubmitTask(domain.Task{
		ProjectPath: "/proj/pending-plan",
		Instruction: "plan the work",
		Command:     domain.CommandPlan,
	})
	if err != nil {
		t.Fatalf("SubmitTask plan: %v", err)
	}

	// Now submit execute — should fail because plan is not completed
	_, err = orch.SubmitTask(domain.Task{
		ProjectPath: "/proj/pending-plan",
		Instruction: "execute now",
		Command:     domain.CommandExecute,
	})
	if err == nil {
		t.Fatal("expected ErrNoPlan, got nil")
	}
	if !errors.Is(err, domain.ErrNoPlan) {
		t.Errorf("expected ErrNoPlan, got: %v", err)
	}
}

func TestOrchestrator_StatusNoProvider_AfterRemove(t *testing.T) {
	repo := newMemRepo()
	// Provider supports llama3. We remove it, then submit a task that needs it.
	stub := &mockLLMClient{alive: true, name: "ollama", activeModel: "llama3", models: []string{"llama3"}}
	discovery := services.NewDiscoveryService(stub)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	if err := orch.RemoveProvider("ollama"); err != nil {
		t.Fatalf("RemoveProvider: %v", err)
	}

	id, err := orch.SubmitTask(domain.Task{Instruction: "do something", ModelID: "llama3"})
	if err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
		saved, _ := repo.GetByID(context.Background(), id)
		if saved.Status == domain.StatusNoProvider {
			return
		}
		if saved.Status == domain.StatusCompleted || saved.Status == domain.StatusFailed {
			t.Fatalf("expected StatusNoProvider but got %s", saved.Status)
		}
	}
	t.Fatal("task did not reach StatusNoProvider within timeout")
}

func TestCreateDraft_CreatesStatusDraft(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, err := orch.CreateDraft(domain.Task{
		ProjectPath: "/proj/draft",
		Instruction: "draft something",
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty task ID")
	}
	saved, err := repo.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if saved.Status != domain.StatusDraft {
		t.Errorf("status: want DRAFT, got %s", saved.Status)
	}
	// Must NOT appear in the execution queue.
	queue, _ := orch.GetQueue()
	for _, qt := range queue {
		if qt.ID == id {
			t.Error("draft task must not be in the worker queue")
		}
	}
}

func TestCreateDraft_DefaultsPriorityToTwo(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, err := orch.CreateDraft(domain.Task{
		ProjectPath: "/proj/priority",
		Instruction: "low priority idea",
		Priority:    0, // explicit zero — should be defaulted to 2
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	saved, _ := repo.GetByID(context.Background(), id)
	if saved.Priority != 2 {
		t.Errorf("priority: want 2, got %d", saved.Priority)
	}
}

func TestCreateDraft_RequiresInstruction(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	_, err := orch.CreateDraft(domain.Task{
		ProjectPath: "/proj/empty",
		Instruction: "",
	})
	if err == nil {
		t.Fatal("expected error for empty instruction, got nil")
	}
}

func TestGetBacklog_ReturnsOnlyDraftAndBacklog(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	const proj = "/proj/backlog"

	draftID, err := orch.CreateDraft(domain.Task{ProjectPath: proj, Instruction: "draft idea"})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	// Manually save a BACKLOG task.
	if err := repo.Save(context.Background(), domain.Task{
		ID:          "backlog-1",
		ProjectPath: proj,
		Instruction: "backlog idea",
		Status:      domain.StatusBacklog,
		Priority:    2,
	}); err != nil {
		t.Fatalf("Save backlog: %v", err)
	}

	// SubmitTask creates a QUEUED task (should NOT appear in backlog).
	_, err = orch.SubmitTask(domain.Task{ProjectPath: proj, Instruction: "queued work"})
	if err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}

	tasks, err := orch.GetBacklog(proj)
	if err != nil {
		t.Fatalf("GetBacklog: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("GetBacklog: want 2 tasks, got %d", len(tasks))
	}
	for _, task := range tasks {
		if task.Status != domain.StatusDraft && task.Status != domain.StatusBacklog {
			t.Errorf("unexpected status in backlog: %s", task.Status)
		}
	}
	found := false
	for _, task := range tasks {
		if task.ID == draftID {
			found = true
		}
	}
	if !found {
		t.Error("draft task not found in GetBacklog result")
	}
}

func TestPromoteTask_DraftToQueued(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, err := orch.CreateDraft(domain.Task{
		ProjectPath: "/proj/promote",
		Instruction: "promote me",
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	if _, err := orch.PromoteTask(id); err != nil {
		t.Fatalf("PromoteTask: %v", err)
	}

	saved, err := repo.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if saved.Status != domain.StatusQueued {
		t.Errorf("status after promote: want QUEUED, got %s", saved.Status)
	}
}

func TestPromoteTask_ErrorOnAlreadyQueued(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, err := orch.SubmitTask(domain.Task{
		ProjectPath: "/proj/already-queued",
		Instruction: "already in queue",
	})
	if err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}

	_, err = orch.PromoteTask(id)
	if err == nil {
		t.Fatal("expected error when promoting already-queued task, got nil")
	}
}

func TestPromoteTask_EnforcesQueueCap(t *testing.T) {
	const queueCap = 1
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	orch.WithQueueCap(queueCap)
	defer orch.Stop()

	if err := repo.Save(context.Background(), domain.Task{
		ID:          "pending-cap-task",
		ProjectPath: "/proj/cap",
		Instruction: "existing queued task",
		Status:      domain.StatusQueued,
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	id, err := orch.CreateDraft(domain.Task{
		ProjectPath: "/proj/cap",
		Instruction: "promote later",
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	_, err = orch.PromoteTask(id)
	if err == nil {
		t.Fatal("expected queue cap error, got nil")
	}
	if !errors.Is(err, services.ErrQueueFull) {
		t.Fatalf("expected ErrQueueFull, got %v", err)
	}
}

func TestUpdateTask_MergesFields(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, err := orch.CreateDraft(domain.Task{
		ProjectPath: "/proj/update",
		Instruction: "original instruction",
		Priority:    2,
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	updated, err := orch.UpdateTask(id, domain.Task{
		Instruction: "updated instruction",
		Priority:    1,
	})
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if updated.Instruction != "updated instruction" {
		t.Errorf("instruction: want %q, got %q", "updated instruction", updated.Instruction)
	}
	if updated.Priority != 1 {
		t.Errorf("priority: want 1, got %d", updated.Priority)
	}
	// Verify persisted in repository.
	saved, _ := repo.GetByID(context.Background(), id)
	if saved.Instruction != "updated instruction" {
		t.Errorf("persisted instruction: want %q, got %q", "updated instruction", saved.Instruction)
	}
}

func TestUpdateTask_RejectsQueuedTransition(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, err := orch.CreateDraft(domain.Task{
		ProjectPath: "/proj/update-status",
		Instruction: "status mutation",
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	_, err = orch.UpdateTask(id, domain.Task{Status: domain.StatusQueued})
	if err == nil {
		t.Fatal("expected error when forcing queued via UpdateTask")
	}
	if !strings.Contains(err.Error(), "use promote task") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIsExecutable(t *testing.T) {
	cases := []struct {
		status domain.TaskStatus
		want   bool
	}{
		{domain.StatusDraft, false},
		{domain.StatusBacklog, false},
		{domain.StatusQueued, true},
		{domain.StatusProcessing, false},
		{domain.StatusCompleted, false},
		{domain.StatusFailed, false},
		{domain.StatusCancelled, false},
	}
	for _, tc := range cases {
		t.Run(string(tc.status), func(t *testing.T) {
			task := domain.Task{Status: tc.status}
			if got := task.IsExecutable(); got != tc.want {
				t.Errorf("IsExecutable() for %s: want %v, got %v", tc.status, tc.want, got)
			}
		})
	}
}
