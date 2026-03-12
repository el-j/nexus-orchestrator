package services_test

import (
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
