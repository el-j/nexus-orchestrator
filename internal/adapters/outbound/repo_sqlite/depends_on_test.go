package repo_sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
)

func TestRepository_DependsOn_SaveClaimAndCorruptJSON(t *testing.T) {
	ctx := context.Background()
	repo, err := repo_sqlite.New(filepath.Join(t.TempDir(), "deps.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	now := time.Now()
	t1 := domain.Task{
		ID:          "t1",
		ProjectPath: "/proj",
		TargetFile:  "a.go",
		Instruction: "first",
		Status:      domain.StatusQueued,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	t2 := domain.Task{
		ID:          "t2",
		ProjectPath: "/proj",
		TargetFile:  "b.go",
		Instruction: "second depends on t1",
		DependsOn:   []string{"t1"},
		Status:      domain.StatusQueued,
		CreatedAt:   now.Add(-time.Second), // older than t1, but blocked!
		UpdatedAt:   now,
	}
	t3 := domain.Task{
		ID:          "t3",
		ProjectPath: "/proj",
		TargetFile:  "c.go",
		Instruction: "depends on missing",
		DependsOn:   []string{"missing-id"},
		Status:      domain.StatusQueued,
		CreatedAt:   now.Add(-2 * time.Second),
		UpdatedAt:   now,
	}
	for _, task := range []domain.Task{t1, t2, t3} {
		if err := repo.Save(ctx, task); err != nil {
			t.Fatal(err)
		}
	}

	// ClaimNextQueued must skip t3 (missing dep) and t2 (t1 not completed), claiming t1 first.
	claimed, err := repo.ClaimNextQueued(ctx)
	if err != nil || claimed.ID != "t1" {
		t.Fatalf("expected t1, got %q err=%v", claimed.ID, err)
	}

	// While t1 is PROCESSING, nothing is claimable.
	if _, err := repo.ClaimNextQueued(ctx); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound while t1 processing, got %v", err)
	}

	// Complete t1 -> now t2 becomes claimable!
	if err := repo.UpdateStatus(ctx, "t1", domain.StatusCompleted); err != nil {
		t.Fatal(err)
	}
	claimed2, err := repo.ClaimNextQueued(ctx)
	if err != nil || claimed2.ID != "t2" {
		t.Fatalf("expected t2 after t1 completed, got %q err=%v", claimed2.ID, err)
	}
	if len(claimed2.DependsOn) != 1 || claimed2.DependsOn[0] != "t1" {
		t.Fatalf("expected DependsOn=[t1], got %v", claimed2.DependsOn)
	}

	// Corrupt depends_on JSON should log and still return the task.
	dbPath := filepath.Join(t.TempDir(), "corrupt.db")
	repo2, err := repo_sqlite.New(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repo2.Close() })
	if err := repo2.Save(ctx, t1); err != nil {
		t.Fatal(err)
	}
	rawDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rawDB.Close() })
	if _, err := rawDB.Exec(`UPDATE tasks SET depends_on = 'not-json' WHERE id = 't1'`); err != nil {
		t.Fatal(err)
	}
	got, err := repo2.GetByID(ctx, "t1")
	if err != nil || got.ID != "t1" {
		t.Fatalf("expected task despite corrupt depends_on, got %+v err=%v", got, err)
	}

	// Drop tasks table on repo2's connection to exercise ClaimNextQueued query error path.
	if _, err := repo2.DB().Exec(`DROP TABLE tasks`); err != nil {
		t.Fatal(err)
	}
	if _, err := repo2.ClaimNextQueued(ctx); err == nil || errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected SQL error when tasks table is missing, got %v", err)
	}
}
