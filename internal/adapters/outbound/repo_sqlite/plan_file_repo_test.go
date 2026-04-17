package repo_sqlite_test

import (
	"context"
	"testing"
	"time"

	repo_sqlite "nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
)

func newPlanFileRepo(t *testing.T) (*repo_sqlite.Repository, *repo_sqlite.PlanFileRepo) {
	t.Helper()
	r := newTestRepo(t)
	return r, repo_sqlite.NewPlanFileRepo(r)
}

func makePlanFile(id, path, project string, isActive bool, modTime time.Time) domain.DiscoveredPlanFile {
	return domain.DiscoveredPlanFile{
		ID:           id,
		Path:         path,
		Kind:         domain.PlanFileKindNexus,
		Format:       "json",
		ProjectPath:  project,
		Summary:      "summary for " + id,
		LastModified: modTime,
		IsActive:     isActive,
	}
}

func TestUpsertPlanFile_RoundTrip(t *testing.T) {
	r, repo := newPlanFileRepo(t)
	defer r.Close()
	ctx := context.Background()

	ts := time.Now().Truncate(time.Second)
	f := makePlanFile("pf-1", "/proj/a/.claude/orchestrator.json", "/proj/a", true, ts)

	if err := repo.UpsertPlanFile(ctx, f); err != nil {
		t.Fatalf("UpsertPlanFile: %v", err)
	}

	files, err := repo.ListPlanFiles(ctx, "/proj/a")
	if err != nil {
		t.Fatalf("ListPlanFiles: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	got := files[0]
	if got.ID != "pf-1" {
		t.Errorf("ID: want pf-1, got %s", got.ID)
	}
	if got.Path != f.Path {
		t.Errorf("Path: want %s, got %s", f.Path, got.Path)
	}
	if !got.IsActive {
		t.Errorf("IsActive: expected true")
	}
}

func TestUpsertPlanFile_Update(t *testing.T) {
	r, repo := newPlanFileRepo(t)
	defer r.Close()
	ctx := context.Background()

	ts := time.Now().Truncate(time.Second)
	f := makePlanFile("pf-2", "/proj/b/plans/PLAN-001.md", "/proj/b", false, ts)
	if err := repo.UpsertPlanFile(ctx, f); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	f.Summary = "updated summary"
	f.IsActive = true
	if err := repo.UpsertPlanFile(ctx, f); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	files, err := repo.ListPlanFiles(ctx, "/proj/b")
	if err != nil {
		t.Fatalf("ListPlanFiles: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file after upsert, got %d", len(files))
	}
	if files[0].Summary != "updated summary" {
		t.Errorf("Summary: want %q, got %q", "updated summary", files[0].Summary)
	}
	if !files[0].IsActive {
		t.Errorf("IsActive: expected true after update")
	}
}

func TestListPlanFiles_FilterByProjectPath(t *testing.T) {
	r, repo := newPlanFileRepo(t)
	defer r.Close()
	ctx := context.Background()

	ts := time.Now().Truncate(time.Second)
	_ = repo.UpsertPlanFile(ctx, makePlanFile("pf-3", "/proj/c/a.json", "/proj/c", false, ts))
	_ = repo.UpsertPlanFile(ctx, makePlanFile("pf-4", "/proj/d/b.json", "/proj/d", false, ts))

	files, err := repo.ListPlanFiles(ctx, "/proj/c")
	if err != nil {
		t.Fatalf("ListPlanFiles: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file for /proj/c, got %d", len(files))
	}
	if files[0].ID != "pf-3" {
		t.Errorf("expected pf-3, got %s", files[0].ID)
	}
}

func TestListPlanFiles_NoFilter(t *testing.T) {
	r, repo := newPlanFileRepo(t)
	defer r.Close()
	ctx := context.Background()

	ts := time.Now().Truncate(time.Second)
	_ = repo.UpsertPlanFile(ctx, makePlanFile("pf-5", "/proj/e/a.json", "/proj/e", false, ts))
	_ = repo.UpsertPlanFile(ctx, makePlanFile("pf-6", "/proj/f/b.json", "/proj/f", false, ts))

	files, err := repo.ListPlanFiles(ctx, "")
	if err != nil {
		t.Fatalf("ListPlanFiles with empty path: %v", err)
	}
	if len(files) < 2 {
		t.Fatalf("expected at least 2 files, got %d", len(files))
	}
}

func TestDeleteStalePlanFiles(t *testing.T) {
	r, repo := newPlanFileRepo(t)
	defer r.Close()
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	_ = repo.UpsertPlanFile(ctx, makePlanFile("pf-old", "/proj/g/old.json", "/proj/g", false, now))
	_ = repo.UpsertPlanFile(ctx, makePlanFile("pf-new", "/proj/g/new.json", "/proj/g", true, now))

	// Backdate the updated_at for pf-old so it appears stale.
	oldTS := now.Add(-48 * time.Hour).Unix()
	if _, err := r.DB().ExecContext(ctx,
		`UPDATE discovered_plan_files SET updated_at = ? WHERE id = ?`, oldTS, "pf-old"); err != nil {
		t.Fatalf("backdate updated_at: %v", err)
	}

	n, err := repo.DeleteStalePlanFiles(ctx, 24*time.Hour)
	if err != nil {
		t.Fatalf("DeleteStalePlanFiles: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 deleted, got %d", n)
	}

	files, err := repo.ListPlanFiles(ctx, "/proj/g")
	if err != nil {
		t.Fatalf("ListPlanFiles after delete: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 remaining file, got %d", len(files))
	}
	if files[0].ID != "pf-new" {
		t.Errorf("expected pf-new to remain, got %s", files[0].ID)
	}
}
