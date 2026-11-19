package repo_sqlite_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

func TestRepository_GetPending_ReturnsQueuedAndProcessing(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	base := time.Now().Add(-10 * time.Second)

	tasks := []domain.Task{
		newTask("t-queued", domain.StatusQueued, base.Add(0)),
		newTask("t-processing", domain.StatusProcessing, base.Add(1*time.Second)),
		newTask("t-completed", domain.StatusCompleted, base.Add(2*time.Second)),
		newTask("t-failed", domain.StatusFailed, base.Add(3*time.Second)),
		newTask("t-cancelled", domain.StatusCancelled, base.Add(4*time.Second)),
	}

	for _, task := range tasks {
		if err := repo.Save(context.Background(), task); err != nil {
			t.Fatalf("Save %q: %v", task.ID, err)
		}
	}

	pending, err := repo.GetPending(context.Background())
	if err != nil {
		t.Fatalf("GetPending: %v", err)
	}

	if len(pending) != 2 {
		t.Fatalf("GetPending: got %d tasks, want 2", len(pending))
	}

	if pending[0].ID != "t-queued" {
		t.Errorf("pending[0].ID: got %q, want %q", pending[0].ID, "t-queued")
	}
	if pending[1].ID != "t-processing" {
		t.Errorf("pending[1].ID: got %q, want %q", pending[1].ID, "t-processing")
	}

	for _, p := range pending {
		if p.Status != domain.StatusQueued && p.Status != domain.StatusProcessing {
			t.Errorf("unexpected status in pending result: %q", p.Status)
		}
	}
}

func TestRepository_GetPending_Empty(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	pending, err := repo.GetPending(context.Background())
	if err != nil {
		t.Fatalf("GetPending on empty db: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("expected 0 pending tasks, got %d", len(pending))
	}
}

func TestRepository_ClaimNextQueued_ClaimsOldestQueuedAndMarksProcessing(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	base := time.Now().Add(-5 * time.Minute)
	queuedOldest := newTask("queued-oldest", domain.StatusQueued, base)
	queuedNewest := newTask("queued-newest", domain.StatusQueued, base.Add(time.Minute))
	processing := newTask("already-processing", domain.StatusProcessing, base.Add(2*time.Minute))

	for _, task := range []domain.Task{queuedNewest, processing, queuedOldest} {
		if err := repo.Save(context.Background(), task); err != nil {
			t.Fatalf("Save %s: %v", task.ID, err)
		}
	}

	claimed, err := repo.ClaimNextQueued(context.Background())
	if err != nil {
		t.Fatalf("ClaimNextQueued: %v", err)
	}
	if claimed.ID != queuedOldest.ID {
		t.Fatalf("claimed task: want %q, got %q", queuedOldest.ID, claimed.ID)
	}
	if claimed.Status != domain.StatusProcessing {
		t.Fatalf("claimed status: want %s, got %s", domain.StatusProcessing, claimed.Status)
	}

	saved, err := repo.GetByID(context.Background(), queuedOldest.ID)
	if err != nil {
		t.Fatalf("GetByID claimed task: %v", err)
	}
	if saved.Status != domain.StatusProcessing {
		t.Fatalf("persisted claimed status: want %s, got %s", domain.StatusProcessing, saved.Status)
	}
}

func TestRepository_ClaimNextQueued_ReturnsErrNotFoundWhenEmpty(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	_, err := repo.ClaimNextQueued(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected domain.ErrNotFound, got %v", err)
	}
}

func TestRepository_UpdateStatusIfCurrent(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	task := newTask("conditional-status", domain.StatusQueued, time.Now())
	if err := repo.Save(context.Background(), task); err != nil {
		t.Fatalf("Save: %v", err)
	}

	ok, err := repo.UpdateStatusIfCurrent(context.Background(), task.ID, domain.StatusQueued, domain.StatusCancelled)
	if err != nil {
		t.Fatalf("UpdateStatusIfCurrent: %v", err)
	}
	if !ok {
		t.Fatal("expected transition to succeed")
	}

	ok, err = repo.UpdateStatusIfCurrent(context.Background(), task.ID, domain.StatusQueued, domain.StatusCompleted)
	if err != nil {
		t.Fatalf("UpdateStatusIfCurrent second transition: %v", err)
	}
	if ok {
		t.Fatal("expected second transition to fail because status no longer matches")
	}

	saved, err := repo.GetByID(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if saved.Status != domain.StatusCancelled {
		t.Fatalf("status after guarded update: want %s, got %s", domain.StatusCancelled, saved.Status)
	}
}

func TestRepository_UpdateStatus(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	task := newTask("task-upd-status", domain.StatusQueued, time.Now())
	if err := repo.Save(context.Background(), task); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := repo.UpdateStatus(context.Background(), "task-upd-status", domain.StatusCompleted); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	got, err := repo.GetByID(context.Background(), "task-upd-status")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != domain.StatusCompleted {
		t.Errorf("Status: got %q, want %q", got.Status, domain.StatusCompleted)
	}
}

func TestRepository_UpdateLogs(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	task := newTask("task-upd-logs", domain.StatusQueued, time.Now())
	if err := repo.Save(context.Background(), task); err != nil {
		t.Fatalf("Save: %v", err)
	}

	wantLogs := "step 1 done\nstep 2 done\n"
	if err := repo.UpdateLogs(context.Background(), "task-upd-logs", wantLogs); err != nil {
		t.Fatalf("UpdateLogs: %v", err)
	}

	got, err := repo.GetByID(context.Background(), "task-upd-logs")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Logs != wantLogs {
		t.Errorf("Logs: got %q, want %q", got.Logs, wantLogs)
	}
}

func TestRepository_UpdateStatus_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	err := repo.UpdateStatus(context.Background(), "nonexistent-id", domain.StatusCompleted)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected domain.ErrNotFound, got %v", err)
	}
}

func TestRepository_UpdateLogs_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	err := repo.UpdateLogs(context.Background(), "nonexistent-id", "some logs")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected domain.ErrNotFound, got %v", err)
	}
}

func TestGetStaleProcessing_ReturnsOnlyStale(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	ctx := context.Background()
	now := time.Now()

	staleTask := domain.Task{
		ID:          "stale-task",
		ProjectPath: "/proj/stale",
		TargetFile:  "main.go",
		Instruction: "old work",
		Status:      domain.StatusProcessing,
		CreatedAt:   now.Add(-10 * time.Minute),
		UpdatedAt:   now.Add(-10 * time.Minute),
	}
	freshTask := domain.Task{
		ID:          "fresh-task",
		ProjectPath: "/proj/stale",
		TargetFile:  "main.go",
		Instruction: "new work",
		Status:      domain.StatusProcessing,
		CreatedAt:   now.Add(-1 * time.Minute),
		UpdatedAt:   now.Add(-1 * time.Minute),
	}

	if err := repo.Save(context.Background(), staleTask); err != nil {
		t.Fatalf("Save staleTask: %v", err)
	}
	if err := repo.Save(context.Background(), freshTask); err != nil {
		t.Fatalf("Save freshTask: %v", err)
	}

	got, err := repo.GetStaleProcessing(ctx, 5*time.Minute)
	if err != nil {
		t.Fatalf("GetStaleProcessing: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 stale task, got %d", len(got))
	}
	if got[0].ID != "stale-task" {
		t.Errorf("expected stale-task, got %q", got[0].ID)
	}
}

func TestGetStaleProcessing_Empty(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	ctx := context.Background()

	got, err := repo.GetStaleProcessing(ctx, 5*time.Minute)
	if err != nil {
		t.Fatalf("GetStaleProcessing on empty DB: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 stale tasks, got %d", len(got))
	}
}
