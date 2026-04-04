package services_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

// TestStartupRecovery verifies that tasks stuck in PROCESSING after a crash are
// re-queued to QUEUED by NewOrchestrator's startup recovery and eventually processed.
func TestStartupRecovery(t *testing.T) {
	repo := newMemRepo()

	// Seed a task that was PROCESSING when the service last crashed.
	stuckTask := domain.Task{
		ID:          "stuck-task-recovery",
		ProjectPath: "/tmp/recovery-test",
		Instruction: "recover me",
		Status:      domain.StatusProcessing,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := repo.Save(stuckTask); err != nil {
		t.Fatalf("setup: save stuck task: %v", err)
	}

	// Wire a working LLM so the worker can complete the re-queued task.
	llm := &mockLLMClient{alive: true, name: "mock", code: "recovered code"}
	discovery := services.NewDiscoveryService(llm)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	// Startup recovery must have re-queued the task; the worker will complete it.
	// Reaching COMPLETED proves the task left the PROCESSING state and was re-queued.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
		saved, _ := repo.GetByID(stuckTask.ID)
		if saved.Status == domain.StatusCompleted {
			return // success
		}
	}
	saved, _ := repo.GetByID(stuckTask.ID)
	t.Errorf("task did not reach COMPLETED after startup recovery; final status: %s", saved.Status)
}

// TestPathNormalization verifies that SubmitTask normalises a relative ProjectPath
// to an absolute, cleaned path before storage.
func TestPathNormalization(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService() // no providers — task stays QUEUED
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, err := orch.SubmitTask(domain.Task{
		ProjectPath: "relative/path/to/project",
		Instruction: "normalize me",
	})
	if err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}

	saved, err := repo.GetByID(id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if !filepath.IsAbs(saved.ProjectPath) {
		t.Errorf("expected absolute ProjectPath, got: %q", saved.ProjectPath)
	}
	if saved.ProjectPath != filepath.Clean(saved.ProjectPath) {
		t.Errorf("expected cleaned ProjectPath, got: %q", saved.ProjectPath)
	}
}

// TestQueueCap verifies that SubmitTask returns ErrQueueFull once the QUEUED
// task count reaches the configured cap.
func TestQueueCap(t *testing.T) {
	const queueCap = 3
	repo := newMemRepo()
	discovery := services.NewDiscoveryService() // no providers — tasks never leave queue
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	orch.WithQueueCap(queueCap)
	defer orch.Stop()

	// Directly seed `queueCap` QUEUED tasks into the repo to avoid relying on worker timing.
	for i := 0; i < queueCap; i++ {
		qt := domain.Task{
			ID:          fmt.Sprintf("seed-queued-%d", i),
			Instruction: "seed",
			Status:      domain.StatusQueued,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := repo.Save(qt); err != nil {
			t.Fatalf("seed task %d: %v", i, err)
		}
	}

	// The next submission must be rejected.
	_, err := orch.SubmitTask(domain.Task{Instruction: "overflow"})
	if err == nil {
		t.Fatal("expected ErrQueueFull for the (queueCap+1)-th task, got nil")
	}
	if !errors.Is(err, services.ErrQueueFull) {
		t.Errorf("expected errors.Is(err, ErrQueueFull), got: %v", err)
	}
}

func TestCancelTask_CancelsPersistedQueuedTask(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	now := time.Now()
	queuedTask := domain.Task{
		ID:          "persisted-queued-task",
		ProjectPath: "/tmp/recovery-test",
		Instruction: "cancel me after restart",
		Status:      domain.StatusQueued,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := repo.Save(queuedTask); err != nil {
		t.Fatalf("setup: save queued task: %v", err)
	}

	if err := orch.CancelTask(queuedTask.ID); err != nil {
		t.Fatalf("CancelTask: %v", err)
	}

	saved, err := repo.GetByID(queuedTask.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if saved.Status != domain.StatusCancelled {
		t.Fatalf("status after cancel: want %s, got %s", domain.StatusCancelled, saved.Status)
	}
}

func TestStartupRecoverySignalsExistingQueuedTask(t *testing.T) {
	repo := newMemRepo()
	now := time.Now()
	queuedTask := domain.Task{
		ID:          "persisted-queued-on-startup",
		ProjectPath: "/tmp/recovery-test",
		Instruction: "process me from persisted queue",
		Status:      domain.StatusQueued,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := repo.Save(queuedTask); err != nil {
		t.Fatalf("setup: save queued task: %v", err)
	}

	llm := &mockLLMClient{alive: true, name: "mock", code: "queued startup code"}
	discovery := services.NewDiscoveryService(llm)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
		saved, _ := repo.GetByID(queuedTask.ID)
		if saved.Status == domain.StatusCompleted {
			return
		}
	}
	saved, _ := repo.GetByID(queuedTask.ID)
	t.Fatalf("task did not reach COMPLETED from persisted queued state; final status: %s", saved.Status)
}

// TestRetryLimit verifies that a task whose LLM calls always fail is retried
// exactly maxRetries times and then permanently set to StatusFailed.
func TestRetryLimit(t *testing.T) {
	repo := newMemRepo()
	llm := &mockLLMClient{alive: true, name: "mock", codeErr: errors.New("llm unavailable")}
	discovery := services.NewDiscoveryService(llm)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, err := orch.SubmitTask(domain.Task{Instruction: "fail me"})
	if err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}

	// 1 initial attempt + 3 retries = 4 total LLM calls.
	// Allow generous time for all retry cycles to complete.
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(300 * time.Millisecond)
		saved, _ := repo.GetByID(id)
		if saved.Status == domain.StatusFailed {
			const wantRetries = 3 // maxRetries in services package
			if saved.RetryCount != wantRetries {
				t.Errorf("expected RetryCount=%d after exhausting retries, got %d",
					wantRetries, saved.RetryCount)
			}
			return // success
		}
	}
	saved, _ := repo.GetByID(id)
	t.Errorf("task did not reach StatusFailed within timeout; final status=%s RetryCount=%d",
		saved.Status, saved.RetryCount)
}

func TestCancelTask_NoProviderState_Succeeds(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService() // no providers
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, err := orch.CreateDraft(domain.Task{
		ProjectPath: "/proj/cancel-noprovider",
		Instruction: "stuck task",
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	// Manually place the task in NO_PROVIDER state via repo
	if err := repo.UpdateStatus(id, domain.StatusNoProvider); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	if err := orch.CancelTask(id); err != nil {
		t.Fatalf("CancelTask on NO_PROVIDER task: %v", err)
	}

	saved, err := repo.GetByID(id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if saved.Status != domain.StatusCancelled {
		t.Errorf("status after cancel: want CANCELLED, got %s", saved.Status)
	}
}

func TestCancelTask_DraftState_Succeeds(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, err := orch.CreateDraft(domain.Task{
		ProjectPath: "/proj/cancel-draft",
		Instruction: "dismiss this draft",
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	if err := orch.CancelTask(id); err != nil {
		t.Fatalf("CancelTask on DRAFT task: %v", err)
	}

	saved, err := repo.GetByID(id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if saved.Status != domain.StatusCancelled {
		t.Errorf("status after cancel: want CANCELLED, got %s", saved.Status)
	}
}

func TestCancelTask_BacklogState_Succeeds(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	now := time.Now()
	task := domain.Task{
		ID:          "backlog-cancel-1",
		ProjectPath: "/proj/cancel-backlog",
		Instruction: "dismiss this backlog item",
		Status:      domain.StatusBacklog,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := repo.Save(task); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := orch.CancelTask(task.ID); err != nil {
		t.Fatalf("CancelTask on BACKLOG task: %v", err)
	}

	saved, err := repo.GetByID(task.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if saved.Status != domain.StatusCancelled {
		t.Errorf("status after cancel: want CANCELLED, got %s", saved.Status)
	}
}

func TestPromoteTask_NoProvider_ReturnsWarning(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService() // no providers registered
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, err := orch.CreateDraft(domain.Task{
		ProjectPath: "/proj/promote-noprovider",
		Instruction: "needs a provider",
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	result, err := orch.PromoteTask(id)
	if err != nil {
		t.Fatalf("PromoteTask: %v", err)
	}
	if !result.Promoted {
		t.Error("Promoted: want true, got false")
	}
	if result.Warning == "" {
		t.Error("Warning: expected non-empty warning when no provider available, got empty")
	}

	saved, err := repo.GetByID(id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if saved.Status != domain.StatusQueued {
		t.Errorf("status after promote: want QUEUED, got %s", saved.Status)
	}
}

// ---- TASK-498: stale-task watchdog ------------------------------------------

// staleAwareRepo wraps memRepo and returns tasks whose UpdatedAt is older than
// the given threshold when GetStaleProcessing is called.
type staleAwareRepo struct {
	*memRepo
}

func (r *staleAwareRepo) GetStaleProcessing(_ context.Context, threshold time.Duration) ([]domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cutoff := time.Now().Add(-threshold)
	var out []domain.Task
	for _, t := range r.tasks {
		if t.Status == domain.StatusProcessing && t.UpdatedAt.Before(cutoff) {
			out = append(out, t)
		}
	}
	return out, nil
}

// TestWatchdog_StaleProcessingTaskFailed verifies that a PROCESSING task whose
// UpdatedAt is older than the stale threshold is marked FAILED by the watchdog.
func TestWatchdog_StaleProcessingTaskFailed(t *testing.T) {
	base := newMemRepo()
	repo := &staleAwareRepo{base}

	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil,
		services.WithWatchdogInterval(200*time.Millisecond),
	)
	defer orch.Stop()

	// Seed AFTER orchestrator starts so startup recovery (which runs on construction)
	// does not re-queue the task before the watchdog can fail it.
	staleTask := domain.Task{
		ID:          "stale-task-watchdog",
		ProjectPath: "/tmp/watchdog",
		Instruction: "stale",
		Status:      domain.StatusProcessing,
		CreatedAt:   time.Now().Add(-10 * time.Minute),
		UpdatedAt:   time.Now().Add(-10 * time.Minute), // older than 5-minute threshold
	}
	if err := repo.Save(staleTask); err != nil {
		t.Fatalf("seed stale task: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(150 * time.Millisecond)
		saved, _ := repo.GetByID(staleTask.ID)
		if saved.Status == domain.StatusFailed {
			return // success
		}
	}
	saved, _ := repo.GetByID(staleTask.ID)
	t.Errorf("stale task not FAILED by watchdog; final status: %s", saved.Status)
}

// TestWatchdog_FreshProcessingTaskUntouched verifies that a PROCESSING task with
// a recent UpdatedAt is NOT failed by the watchdog.
func TestWatchdog_FreshProcessingTaskUntouched(t *testing.T) {
	base := newMemRepo()
	repo := &staleAwareRepo{base}

	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil,
		services.WithWatchdogInterval(200*time.Millisecond),
	)
	defer orch.Stop()

	// Seed AFTER orchestrator starts so startup recovery doesn't re-queue it.
	freshTask := domain.Task{
		ID:          "fresh-task-watchdog",
		ProjectPath: "/tmp/watchdog-fresh",
		Instruction: "fresh",
		Status:      domain.StatusProcessing,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(), // brand new → not stale
	}
	if err := repo.Save(freshTask); err != nil {
		t.Fatalf("seed fresh task: %v", err)
	}

	// Wait long enough for 3+ watchdog cycles.
	time.Sleep(700 * time.Millisecond)

	saved, _ := repo.GetByID(freshTask.ID)
	if saved.Status != domain.StatusProcessing {
		t.Errorf("fresh task should remain PROCESSING; got %s", saved.Status)
	}
}
