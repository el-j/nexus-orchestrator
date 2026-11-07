package repo_sqlite_test

import (
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
