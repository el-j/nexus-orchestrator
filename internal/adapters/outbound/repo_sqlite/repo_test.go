package repo_sqlite_test

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	repo_sqlite "nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
)

func newTestRepo(t *testing.T) *repo_sqlite.Repository {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	r, err := repo_sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("newTestRepo: %v", err)
	}
	return r
}

func newTask(id string, status domain.TaskStatus, createdAt time.Time) domain.Task {
	return domain.Task{
		ID:           id,
		ProjectPath:  "/projects/foo",
		TargetFile:   "main.go",
		Instruction:  "do something",
		ContextFiles: []string{"a.go", "b.go"},
		Status:       status,
		CreatedAt:    createdAt,
		UpdatedAt:    createdAt,
		Logs:         "",
	}
}

func TestRepository_Save_And_GetByID(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	now := time.Now().Truncate(time.Millisecond)
	task := domain.Task{
		ID:           "task-1",
		ProjectPath:  "/projects/alpha",
		TargetFile:   "handler.go",
		Instruction:  "refactor this",
		ContextFiles: []string{"util.go", "types.go"},
		Status:       domain.StatusQueued,
		CreatedAt:    now,
		UpdatedAt:    now,
		Logs:         "initial log",
	}

	if err := repo.Save(task); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetByID("task-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.ID != task.ID {
		t.Errorf("ID: got %q, want %q", got.ID, task.ID)
	}
	if got.ProjectPath != task.ProjectPath {
		t.Errorf("ProjectPath: got %q, want %q", got.ProjectPath, task.ProjectPath)
	}
	if got.TargetFile != task.TargetFile {
		t.Errorf("TargetFile: got %q, want %q", got.TargetFile, task.TargetFile)
	}
	if got.Instruction != task.Instruction {
		t.Errorf("Instruction: got %q, want %q", got.Instruction, task.Instruction)
	}
	if got.Status != task.Status {
		t.Errorf("Status: got %q, want %q", got.Status, task.Status)
	}
	if got.Logs != task.Logs {
		t.Errorf("Logs: got %q, want %q", got.Logs, task.Logs)
	}
	if len(got.ContextFiles) != len(task.ContextFiles) {
		t.Fatalf("ContextFiles length: got %d, want %d", len(got.ContextFiles), len(task.ContextFiles))
	}
	for i, f := range task.ContextFiles {
		if got.ContextFiles[i] != f {
			t.Errorf("ContextFiles[%d]: got %q, want %q", i, got.ContextFiles[i], f)
		}
	}
	if diff := got.CreatedAt.UnixMilli() - task.CreatedAt.UnixMilli(); diff > 1000 || diff < -1000 {
		t.Errorf("CreatedAt drift too large: %d ms", diff)
	}
	if diff := got.UpdatedAt.UnixMilli() - task.UpdatedAt.UnixMilli(); diff > 1000 || diff < -1000 {
		t.Errorf("UpdatedAt drift too large: %d ms", diff)
	}
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	_, err := repo.GetByID("nonexistent-id")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected domain.ErrNotFound, got %v", err)
	}
}

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
		if err := repo.Save(task); err != nil {
			t.Fatalf("Save %q: %v", task.ID, err)
		}
	}

	pending, err := repo.GetPending()
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

	pending, err := repo.GetPending()
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
		if err := repo.Save(task); err != nil {
			t.Fatalf("Save %s: %v", task.ID, err)
		}
	}

	claimed, err := repo.ClaimNextQueued()
	if err != nil {
		t.Fatalf("ClaimNextQueued: %v", err)
	}
	if claimed.ID != queuedOldest.ID {
		t.Fatalf("claimed task: want %q, got %q", queuedOldest.ID, claimed.ID)
	}
	if claimed.Status != domain.StatusProcessing {
		t.Fatalf("claimed status: want %s, got %s", domain.StatusProcessing, claimed.Status)
	}

	saved, err := repo.GetByID(queuedOldest.ID)
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

	_, err := repo.ClaimNextQueued()
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
	if err := repo.Save(task); err != nil {
		t.Fatalf("Save: %v", err)
	}

	ok, err := repo.UpdateStatusIfCurrent(task.ID, domain.StatusQueued, domain.StatusCancelled)
	if err != nil {
		t.Fatalf("UpdateStatusIfCurrent: %v", err)
	}
	if !ok {
		t.Fatal("expected transition to succeed")
	}

	ok, err = repo.UpdateStatusIfCurrent(task.ID, domain.StatusQueued, domain.StatusCompleted)
	if err != nil {
		t.Fatalf("UpdateStatusIfCurrent second transition: %v", err)
	}
	if ok {
		t.Fatal("expected second transition to fail because status no longer matches")
	}

	saved, err := repo.GetByID(task.ID)
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
	if err := repo.Save(task); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := repo.UpdateStatus("task-upd-status", domain.StatusCompleted); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	got, err := repo.GetByID("task-upd-status")
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
	if err := repo.Save(task); err != nil {
		t.Fatalf("Save: %v", err)
	}

	wantLogs := "step 1 done\nstep 2 done\n"
	if err := repo.UpdateLogs("task-upd-logs", wantLogs); err != nil {
		t.Fatalf("UpdateLogs: %v", err)
	}

	got, err := repo.GetByID("task-upd-logs")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Logs != wantLogs {
		t.Errorf("Logs: got %q, want %q", got.Logs, wantLogs)
	}
}

func TestRepository_Save_ContextFiles_NilSlice(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	task := domain.Task{
		ID:           "task-nil-ctx",
		ProjectPath:  "/projects/bar",
		TargetFile:   "foo.go",
		Instruction:  "do nothing",
		ContextFiles: nil,
		Status:       domain.StatusQueued,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Logs:         "",
	}

	if err := repo.Save(task); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetByID("task-nil-ctx")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(got.ContextFiles) != 0 {
		t.Errorf("ContextFiles: got %v, want empty slice", got.ContextFiles)
	}
}

func TestRepository_Save_DuplicateID(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	task := newTask("dup-id", domain.StatusQueued, time.Now())

	if err := repo.Save(task); err != nil {
		t.Fatalf("first Save: %v", err)
	}

	err := repo.Save(task)
	if err == nil {
		t.Fatal("expected error on duplicate Save, got nil")
	}
}

func TestRepository_UpdateStatus_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	err := repo.UpdateStatus("nonexistent-id", domain.StatusCompleted)
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

	err := repo.UpdateLogs("nonexistent-id", "some logs")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected domain.ErrNotFound, got %v", err)
	}
}

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
		if err := repo.Save(task); err != nil {
			t.Fatalf("Save %q: %v", task.ID, err)
		}
	}

	got, err := repo.GetByProjectPath("/proj/alpha")
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

	got, err := repo.GetByProjectPath("/proj/nonexistent")
	if err != nil {
		t.Fatalf("GetByProjectPath: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(got))
	}
}

func TestRepository_Save_CommandField_RoundTrip(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	now := time.Now().Truncate(time.Millisecond)
	task := domain.Task{
		ID:          "cmd-1",
		ProjectPath: "/proj/cmd",
		TargetFile:  "main.go",
		Instruction: "plan it",
		Command:     domain.CommandPlan,
		Status:      domain.StatusQueued,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := repo.Save(task); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetByID("cmd-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Command != domain.CommandPlan {
		t.Errorf("Command: got %q, want %q", got.Command, domain.CommandPlan)
	}
}

func TestRepository_Save_EmptyCommand_RoundTrip(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	now := time.Now().Truncate(time.Millisecond)
	task := domain.Task{
		ID:          "cmd-empty",
		ProjectPath: "/proj/cmd",
		TargetFile:  "main.go",
		Instruction: "auto route",
		Command:     "",
		Status:      domain.StatusQueued,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := repo.Save(task); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetByID("cmd-empty")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Command != "" {
		t.Errorf("Command: got %q, want empty", got.Command)
	}
}

// --- Backlog / new-column tests ----------------------------------------------

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
		if err := repo.Save(task); err != nil {
			t.Fatalf("Save %q: %v", task.ID, err)
		}
	}

	got, err := repo.GetByProjectPathAndStatus("/proj/filter", domain.StatusDraft, domain.StatusBacklog)
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

func TestUpdate_PersistsAllFields(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	now := time.Now()
	orig := domain.Task{
		ID:           "upd-task",
		ProjectPath:  "/proj/upd",
		TargetFile:   "old.go",
		Instruction:  "original",
		Status:       domain.StatusDraft,
		Priority:     2,
		Tags:         []string{"old"},
		ProviderName: "",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := repo.Save(orig); err != nil {
		t.Fatalf("Save: %v", err)
	}

	orig.Instruction = "updated instruction"
	orig.TargetFile = "new.go"
	orig.ProviderName = "my-provider"
	orig.Priority = 1
	orig.Tags = []string{"alpha", "beta"}
	orig.Status = domain.StatusBacklog
	if err := repo.Update(orig); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.GetByID("upd-task")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Instruction != "updated instruction" {
		t.Errorf("Instruction: want %q, got %q", "updated instruction", got.Instruction)
	}
	if got.TargetFile != "new.go" {
		t.Errorf("TargetFile: want %q, got %q", "new.go", got.TargetFile)
	}
	if got.ProviderName != "my-provider" {
		t.Errorf("ProviderName: want %q, got %q", "my-provider", got.ProviderName)
	}
	if got.Priority != 1 {
		t.Errorf("Priority: want 1, got %d", got.Priority)
	}
	if got.Status != domain.StatusBacklog {
		t.Errorf("Status: want BACKLOG, got %s", got.Status)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "alpha" || got.Tags[1] != "beta" {
		t.Errorf("Tags: want [alpha beta], got %v", got.Tags)
	}
}

func TestUpdate_ReturnsErrNotFound_ForUnknownID(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	err := repo.Update(domain.Task{
		ID:          "ghost-id",
		Instruction: "something",
	})
	if err == nil {
		t.Fatal("expected error for unknown ID, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected domain.ErrNotFound, got %v", err)
	}
}

func TestTagsRoundtrip(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	now := time.Now()
	task := domain.Task{
		ID:          "tags-task",
		ProjectPath: "/proj/tags",
		Instruction: "tag it",
		Status:      domain.StatusDraft,
		Tags:        []string{"a", "b", "c"},
		Priority:    2,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := repo.Save(task); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetByID("tags-task")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(got.Tags) != 3 {
		t.Fatalf("Tags length: want 3, got %d", len(got.Tags))
	}
	for i, want := range []string{"a", "b", "c"} {
		if got.Tags[i] != want {
			t.Errorf("Tags[%d]: want %q, got %q", i, want, got.Tags[i])
		}
	}
}
