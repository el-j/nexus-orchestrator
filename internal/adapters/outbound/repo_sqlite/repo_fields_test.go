package repo_sqlite_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

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

	if err := repo.Save(context.Background(), task); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetByID(context.Background(), "task-1")
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

	_, err := repo.GetByID(context.Background(), "nonexistent-id")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected domain.ErrNotFound, got %v", err)
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

	if err := repo.Save(context.Background(), task); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetByID(context.Background(), "task-nil-ctx")
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

	if err := repo.Save(context.Background(), task); err != nil {
		t.Fatalf("first Save: %v", err)
	}

	err := repo.Save(context.Background(), task)
	if err == nil {
		t.Fatal("expected error on duplicate Save, got nil")
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
	if err := repo.Save(context.Background(), task); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetByID(context.Background(), "cmd-1")
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
	if err := repo.Save(context.Background(), task); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetByID(context.Background(), "cmd-empty")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Command != "" {
		t.Errorf("Command: got %q, want empty", got.Command)
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
	if err := repo.Save(context.Background(), orig); err != nil {
		t.Fatalf("Save: %v", err)
	}

	orig.Instruction = "updated instruction"
	orig.TargetFile = "new.go"
	orig.ProviderName = "my-provider"
	orig.Priority = 1
	orig.Tags = []string{"alpha", "beta"}
	orig.Status = domain.StatusBacklog
	if err := repo.Update(context.Background(), orig); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.GetByID(context.Background(), "upd-task")
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

	err := repo.Update(context.Background(), domain.Task{
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
	if err := repo.Save(context.Background(), task); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetByID(context.Background(), "tags-task")
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

func TestRepository_TaskVerificationFields(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()

	now := time.Now().Truncate(time.Millisecond)
	task := domain.Task{
		ID:                  "verify-task-1",
		ProjectPath:         "/projects/verify",
		TargetFile:          "calc.go",
		Instruction:         "implement Add",
		Status:              domain.StatusQueued,
		VerificationCommand: "go test -v ./...",
		MaxCorrectionTurns:  3,
		VerificationOutput:  "FAIL: TestAdd",
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := repo.Save(context.Background(), task); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetByID(context.Background(), "verify-task-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.VerificationCommand != "go test -v ./..." {
		t.Errorf("VerificationCommand: got %q, want %q", got.VerificationCommand, "go test -v ./...")
	}
	if got.MaxCorrectionTurns != 3 {
		t.Errorf("MaxCorrectionTurns: got %d, want 3", got.MaxCorrectionTurns)
	}
	if got.VerificationOutput != "FAIL: TestAdd" {
		t.Errorf("VerificationOutput: got %q, want %q", got.VerificationOutput, "FAIL: TestAdd")
	}

	// Test Update
	got.VerificationOutput = "PASS"
	got.Status = domain.StatusCompleted
	if err := repo.Update(context.Background(), got); err != nil {
		t.Fatalf("Update: %v", err)
	}

	updated, err := repo.GetByID(context.Background(), "verify-task-1")
	if err != nil {
		t.Fatalf("GetByID after update: %v", err)
	}
	if updated.VerificationOutput != "PASS" {
		t.Errorf("Updated VerificationOutput: got %q, want %q", updated.VerificationOutput, "PASS")
	}
	if updated.Status != domain.StatusCompleted {
		t.Errorf("Updated Status: got %q, want %q", updated.Status, domain.StatusCompleted)
	}
}
