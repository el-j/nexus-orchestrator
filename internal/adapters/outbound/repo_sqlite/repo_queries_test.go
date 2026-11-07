package repo_sqlite_test

import (
	"context"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

func TestRepository_GetByProjectPath(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	now := time.Now().Truncate(time.Millisecond)
	tasks := []domain.Task{
		{ID: "pp-1", ProjectPath: "/proj/alpha", TargetFile: "a.go", Instruction: "one", Status: domain.StatusQueued, CreatedAt: now, UpdatedAt: now},
		{ID: "pp-2", ProjectPath: "/proj/alpha", TargetFile: "b.go", Instruction: "two", Status: domain.StatusCompleted, CreatedAt: now.Add(time.Second), UpdatedAt: now.Add(time.Second)},
		{ID: "pp-3", ProjectPath: "/proj/beta", TargetFile: "c.go", Instruction: "three", Status: domain.StatusQueued, CreatedAt: now, UpdatedAt: now},
	}
	for _, task := range tasks {
		if err := repo.Save(context.Background(), task); err != nil {
			t.Fatalf("Save %q: %v", task.ID, err)
		}
	}

	got, err := repo.GetByProjectPath(context.Background(), "/proj/alpha")
	if err != nil {
		t.Fatalf("GetByProjectPath: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(got))
	}
	// Ordered by created_at DESC — pp-2 first
	if got[0].ID != "pp-2" {
		t.Errorf("expected first task ID pp-2, got %q", got[0].ID)
	}
	if got[1].ID != "pp-1" {
		t.Errorf("expected second task ID pp-1, got %q", got[1].ID)
	}
}

func TestRepository_GetByProjectPath_Empty(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	got, err := repo.GetByProjectPath(context.Background(), "/proj/nonexistent")
	if err != nil {
		t.Fatalf("GetByProjectPath: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(got))
	}
}

func TestGetByProjectPathAndStatus_FiltersCorrectly(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	now := time.Now()
	tasks := []domain.Task{
		{ID: "s-draft", ProjectPath: "/proj/filter", Instruction: "draft", Status: domain.StatusDraft, Priority: 2, CreatedAt: now, UpdatedAt: now},
		{ID: "s-backlog", ProjectPath: "/proj/filter", Instruction: "backlog", Status: domain.StatusBacklog, Priority: 1, CreatedAt: now.Add(time.Second), UpdatedAt: now.Add(time.Second)},
		{ID: "s-queued", ProjectPath: "/proj/filter", Instruction: "queued", Status: domain.StatusQueued, Priority: 2, CreatedAt: now.Add(2 * time.Second), UpdatedAt: now.Add(2 * time.Second)},
	}
	for _, task := range tasks {
		if err := repo.Save(context.Background(), task); err != nil {
			t.Fatalf("Save %q: %v", task.ID, err)
		}
	}

	got, err := repo.GetByProjectPathAndStatus(context.Background(), "/proj/filter", domain.StatusDraft, domain.StatusBacklog)
	if err != nil {
		t.Fatalf("GetByProjectPathAndStatus: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 results, got %d", len(got))
	}
	// Ordered priority ASC: backlog (priority=1) first, then draft (priority=2)
	if got[0].ID != "s-backlog" {
		t.Errorf("got[0]: want s-backlog, got %q", got[0].ID)
	}
	if got[1].ID != "s-draft" {
		t.Errorf("got[1]: want s-draft, got %q", got[1].ID)
	}
	// Ensure queued task is NOT included.
	for _, task := range got {
		if task.ID == "s-queued" {
			t.Error("s-queued must not appear in DRAFT+BACKLOG filter")
		}
	}
}

func TestGetTasksBySessionID_ReturnsOnlyMatching(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	now := time.Now()

	tasks := []domain.Task{
		{
			ID:          "sess-task-1",
			ProjectPath: "/proj/sess",
			TargetFile:  "a.go",
			Instruction: "one",
			Status:      domain.StatusQueued,
			AISessionID: "session-A",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "sess-task-2",
			ProjectPath: "/proj/sess",
			TargetFile:  "b.go",
			Instruction: "two",
			Status:      domain.StatusQueued,
			AISessionID: "session-A",
			CreatedAt:   now.Add(-time.Second),
			UpdatedAt:   now.Add(-time.Second),
		},
		{
			ID:          "sess-task-3",
			ProjectPath: "/proj/sess",
			TargetFile:  "c.go",
			Instruction: "three",
			Status:      domain.StatusQueued,
			AISessionID: "session-B",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	for _, task := range tasks {
		if err := repo.Save(context.Background(), task); err != nil {
			t.Fatalf("Save %q: %v", task.ID, err)
		}
	}

	got, err := repo.GetTasksBySessionID(context.Background(), "session-A")
	if err != nil {
		t.Fatalf("GetTasksBySessionID: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 tasks for session-A, got %d", len(got))
	}
	// Ordered by created_at DESC: sess-task-1 then sess-task-2
	if got[0].ID != "sess-task-1" {
		t.Errorf("got[0]: want sess-task-1, got %q", got[0].ID)
	}
	if got[1].ID != "sess-task-2" {
		t.Errorf("got[1]: want sess-task-2, got %q", got[1].ID)
	}
	for _, task := range got {
		if task.AISessionID != "session-A" {
			t.Errorf("unexpected AISessionID %q in results", task.AISessionID)
		}
	}
}

func TestGetTasksBySessionID_Empty(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	got, err := repo.GetTasksBySessionID(context.Background(), "no-such-session")
	if err != nil {
		t.Fatalf("GetTasksBySessionID: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(got))
	}
}
