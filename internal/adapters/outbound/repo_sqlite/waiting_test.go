package repo_sqlite_test

import (
	"context"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

func TestRepository_AWaitingTaskKeepsItsWaitUntilAndShowsInTheQueue(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()
	ctx := context.Background()
	now := time.Now().Truncate(time.Millisecond)
	until := now.Add(3 * time.Hour)

	task := domain.Task{ID: "w1", ProjectPath: "/p", Instruction: "x", Status: domain.StatusQueued, CreatedAt: now, UpdatedAt: now}
	if err := repo.Save(ctx, task); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.GetByID(ctx, "w1"); !got.WaitUntil.IsZero() {
		t.Errorf("a task that never waited has no wait time: %v", got.WaitUntil)
	}

	task.Status, task.WaitUntil = domain.StatusWaiting, until
	if err := repo.Update(ctx, task); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByID(ctx, "w1")
	if err != nil || got.Status != domain.StatusWaiting || !got.WaitUntil.Equal(until) {
		t.Fatalf("after update: %+v %v", got, err)
	}

	pending, err := repo.GetPending(ctx)
	if err != nil || len(pending) != 1 || pending[0].ID != "w1" {
		t.Errorf("the queue lists waiting tasks: %+v %v", pending, err)
	}
	if byProject, _ := repo.GetByProjectPathAndStatus(ctx, "/p", domain.StatusWaiting); len(byProject) != 1 {
		t.Errorf("by status: %+v", byProject)
	}

	// Back in the queue the wait time is cleared.
	task.Status, task.WaitUntil = domain.StatusQueued, time.Time{}
	if err := repo.Update(ctx, task); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.GetByID(ctx, "w1"); !got.WaitUntil.IsZero() || got.Status != domain.StatusQueued {
		t.Errorf("after resume: %+v", got)
	}

	// A task saved with a wait time keeps it.
	saved := domain.Task{ID: "w2", ProjectPath: "/p", Instruction: "y", Status: domain.StatusWaiting, WaitUntil: until, CreatedAt: now, UpdatedAt: now}
	if err := repo.Save(ctx, saved); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.GetByID(ctx, "w2"); !got.WaitUntil.Equal(until) {
		t.Errorf("save: %v", got.WaitUntil)
	}
}

func TestRepository_WAITINGIsAKnownStatus(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()
	ctx := context.Background()
	now := time.Now()
	if err := repo.Save(ctx, domain.Task{ID: "k", ProjectPath: "/p", Status: domain.StatusWaiting, WaitUntil: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if n, err := repo.RepairForeignRows(ctx); err != nil || n != 0 {
		t.Errorf("a WAITING row is not a foreign row: repaired %d, %v", n, err)
	}
}

func TestRepository_AnUnreadableWaitTimeDoesNotHideTheTask(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	now := time.Now()
	if err := r.Save(ctx, domain.Task{ID: "h", ProjectPath: "/p", Status: domain.StatusWaiting, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.DB().Exec(`UPDATE tasks SET wait_until = 'soon' WHERE id = 'h'`); err != nil {
		t.Fatal(err)
	}
	got, err := r.GetByID(ctx, "h")
	if err != nil || got.Status != domain.StatusWaiting || !got.WaitUntil.IsZero() {
		t.Errorf("%+v %v", got, err)
	}
}
