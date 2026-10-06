package repo_sqlite_test

import (
	"context"
	"errors"
	"testing"
	"time"

	repo_sqlite "nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
)

// closedRepo returns a Repository whose underlying database is already closed,
// so every query against it fails. Used to exercise error-wrapping branches.
func closedRepo(t *testing.T) *repo_sqlite.Repository {
	t.Helper()
	r := newTestRepo(t)
	if err := r.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return r
}

// ── ActivityRepo ─────────────────────────────────────────────────────────────

func TestActivityRepo_SaveListFilterPurge(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	repo := repo_sqlite.NewActivityRepo(r)
	ctx := context.Background()
	base := time.Now().UTC().Truncate(time.Second)

	acts := []domain.AIActivity{
		{ID: "a1", SessionID: "s1", AgentName: "claude", ActivityType: domain.ActivityTypeMessage,
			Summary: "one", ProjectPath: "/p1", Model: "m", TokensIn: 1, TokensOut: 2,
			Timestamp: base.Add(-3 * time.Hour), Metadata: map[string]string{"k": "v"}},
		{ID: "a2", AgentName: "copilot", ActivityType: domain.ActivityTypeToolUse,
			Summary: "two", ProjectPath: "/p2", Timestamp: base.Add(-2 * time.Hour)},
		{ID: "a3", AgentName: "claude", ActivityType: domain.ActivityTypeToolUse,
			Summary: "three", ProjectPath: "/p1", Timestamp: base.Add(-1 * time.Hour)},
	}
	for _, a := range acts {
		if err := repo.SaveActivity(ctx, a); err != nil {
			t.Fatalf("save %s: %v", a.ID, err)
		}
	}
	// Duplicate ID is ignored.
	if err := repo.SaveActivity(ctx, acts[0]); err != nil {
		t.Fatalf("duplicate save: %v", err)
	}

	all, err := repo.ListActivities(ctx, domain.ActivityFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 || all[0].ID != "a3" || all[2].ID != "a1" {
		t.Fatalf("expected 3 activities newest-first, got %+v", all)
	}
	if all[2].Metadata["k"] != "v" || all[2].TokensIn != 1 || all[2].Model != "m" {
		t.Errorf("fields not round-tripped: %+v", all[2])
	}

	cases := []struct {
		name   string
		filter domain.ActivityFilter
		want   int
	}{
		{"agent", domain.ActivityFilter{AgentName: "claude"}, 2},
		{"project", domain.ActivityFilter{ProjectPath: "/p2"}, 1},
		{"type", domain.ActivityFilter{Type: domain.ActivityTypeToolUse}, 2},
		{"since", domain.ActivityFilter{Since: base.Add(-150 * time.Minute)}, 2},
		{"limit", domain.ActivityFilter{Limit: 1}, 1},
		{"combined", domain.ActivityFilter{AgentName: "claude", Type: domain.ActivityTypeToolUse}, 1},
	}
	for _, tc := range cases {
		got, err := repo.ListActivities(ctx, tc.filter)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if len(got) != tc.want {
			t.Errorf("%s: got %d want %d", tc.name, len(got), tc.want)
		}
	}

	n, err := repo.PurgeOlderThan(ctx, base.Add(-150*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("purged %d, want 1", n)
	}
}

func TestActivityRepo_BadMetadataJSONIsTolerated(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	repo := repo_sqlite.NewActivityRepo(r)
	ctx := context.Background()
	if err := repo.SaveActivity(ctx, domain.AIActivity{ID: "x", Timestamp: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.DB().ExecContext(ctx, `UPDATE ai_activities SET metadata = 'not-json' WHERE id = 'x'`); err != nil {
		t.Fatal(err)
	}
	got, err := repo.ListActivities(ctx, domain.ActivityFilter{})
	if err != nil || len(got) != 1 {
		t.Fatalf("list: %v %v", got, err)
	}
	if got[0].Metadata != nil {
		t.Errorf("expected nil metadata for corrupt JSON, got %v", got[0].Metadata)
	}
}

func TestActivityRepo_ErrorsOnClosedDB(t *testing.T) {
	repo := repo_sqlite.NewActivityRepo(closedRepo(t))
	ctx := context.Background()
	if err := repo.SaveActivity(ctx, domain.AIActivity{ID: "x"}); err == nil {
		t.Error("SaveActivity: expected error")
	}
	if _, err := repo.ListActivities(ctx, domain.ActivityFilter{}); err == nil {
		t.Error("ListActivities: expected error")
	}
	if _, err := repo.PurgeOlderThan(ctx, time.Now()); err == nil {
		t.Error("PurgeOlderThan: expected error")
	}
}

// ── ModelCapabilityRepo ──────────────────────────────────────────────────────

func TestModelCapabilityRepo_CRUD(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	repo := repo_sqlite.NewModelCapabilityRepo(r)

	p := domain.ModelCapabilityProfile{
		ModelID: "m-1", ContextWindow: 8192, RecommendedMaxOutput: 1024, Notes: "n", BuiltIn: true,
	}
	if err := repo.Save(p); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByModelID("m-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ContextWindow != 8192 || got.RecommendedMaxOutput != 1024 || got.Notes != "n" || !got.BuiltIn {
		t.Errorf("round trip mismatch: %+v", got)
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Errorf("timestamps not parsed: %+v", got)
	}

	// Upsert changes fields in place.
	p.ContextWindow = 16384
	p.BuiltIn = false
	p.CreatedAt = time.Now().Add(-time.Hour)
	if err := repo.Save(p); err != nil {
		t.Fatal(err)
	}
	got, _ = repo.GetByModelID("m-1")
	if got.ContextWindow != 16384 || got.BuiltIn {
		t.Errorf("upsert not applied: %+v", got)
	}

	if err := repo.Save(domain.ModelCapabilityProfile{ModelID: "a-0", ContextWindow: 1}); err != nil {
		t.Fatal(err)
	}
	all, err := repo.GetAll()
	if err != nil || len(all) != 2 || all[0].ModelID != "a-0" {
		t.Fatalf("GetAll: %v %v", all, err)
	}

	if err := repo.Delete("m-1"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete("m-1"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("second delete: want ErrNotFound, got %v", err)
	}
	if _, err := repo.GetByModelID("m-1"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("get missing: want ErrNotFound, got %v", err)
	}
}

func TestModelCapabilityRepo_ErrorsOnClosedDB(t *testing.T) {
	repo := repo_sqlite.NewModelCapabilityRepo(closedRepo(t))
	if err := repo.Save(domain.ModelCapabilityProfile{ModelID: "x"}); err == nil {
		t.Error("Save: expected error")
	}
	if _, err := repo.GetByModelID("x"); err == nil || errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetByModelID: expected non-NotFound error, got %v", err)
	}
	if _, err := repo.GetAll(); err == nil {
		t.Error("GetAll: expected error")
	}
	if err := repo.Delete("x"); err == nil || errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Delete: expected non-NotFound error, got %v", err)
	}
}

// ── PlanFileRepo ─────────────────────────────────────────────────────────────

func TestPlanFileRepo_UpsertListDeleteStale(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	repo := repo_sqlite.NewPlanFileRepo(r)
	ctx := context.Background()
	mod := time.Unix(1_700_000_000, 0)

	f1 := domain.DiscoveredPlanFile{ID: "1", Path: "/p1/TASKS.md", Kind: domain.PlanFileKindMarkdown,
		Format: "md", ProjectPath: "/p1", Summary: "s", LastModified: mod, IsActive: true}
	f2 := domain.DiscoveredPlanFile{ID: "2", Path: "/p2/PLAN.md", Kind: domain.PlanFileKindMarkdown,
		Format: "md", ProjectPath: "/p2", LastModified: mod.Add(time.Hour)}
	for _, f := range []domain.DiscoveredPlanFile{f1, f2} {
		if err := repo.UpsertPlanFile(ctx, f); err != nil {
			t.Fatal(err)
		}
	}
	// Replace f1.
	f1.Summary = "updated"
	if err := repo.UpsertPlanFile(ctx, f1); err != nil {
		t.Fatal(err)
	}

	all, err := repo.ListPlanFiles(ctx, "")
	if err != nil || len(all) != 2 || all[0].ID != "2" {
		t.Fatalf("list all: %v %v", all, err)
	}
	one, err := repo.ListPlanFiles(ctx, "/p1")
	if err != nil || len(one) != 1 {
		t.Fatalf("list project: %v %v", one, err)
	}
	if one[0].Summary != "updated" || !one[0].IsActive || !one[0].LastModified.Equal(mod) {
		t.Errorf("round trip mismatch: %+v", one[0])
	}

	// Nothing is stale yet.
	if n, err := repo.DeleteStalePlanFiles(ctx, time.Hour); err != nil || n != 0 {
		t.Fatalf("stale (fresh): n=%d err=%v", n, err)
	}
	// Age the rows, then purge.
	if _, err := r.DB().ExecContext(ctx, `UPDATE discovered_plan_files SET updated_at = 1`); err != nil {
		t.Fatal(err)
	}
	if n, err := repo.DeleteStalePlanFiles(ctx, time.Hour); err != nil || n != 2 {
		t.Fatalf("stale (aged): n=%d err=%v", n, err)
	}
}

func TestPlanFileRepo_ErrorsOnClosedDB(t *testing.T) {
	repo := repo_sqlite.NewPlanFileRepo(closedRepo(t))
	ctx := context.Background()
	if err := repo.UpsertPlanFile(ctx, domain.DiscoveredPlanFile{ID: "x"}); err == nil {
		t.Error("Upsert: expected error")
	}
	if _, err := repo.ListPlanFiles(ctx, ""); err == nil {
		t.Error("List(all): expected error")
	}
	if _, err := repo.ListPlanFiles(ctx, "/p"); err == nil {
		t.Error("List(project): expected error")
	}
	if _, err := repo.DeleteStalePlanFiles(ctx, time.Hour); err == nil {
		t.Error("DeleteStale: expected error")
	}
}

// ── RuntimeConfigRepo ────────────────────────────────────────────────────────

func TestRuntimeConfigRepo_SaveGet(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	repo := repo_sqlite.NewRuntimeConfigRepo(r)
	ctx := context.Background()

	// Migration seeds row id=1; with it removed the repo reports ErrNotFound.
	if _, err := r.DB().ExecContext(ctx, `DELETE FROM runtime_config`); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetRuntimeConfig(ctx); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("empty store: want ErrNotFound, got %v", err)
	}

	cfg := domain.RuntimeConfig{QueueCap: 7, APIToken: "api", MCPToken: "mcp",
		UpdatedAt: time.UnixMilli(1_700_000_000_123)}
	if err := repo.SaveRuntimeConfig(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetRuntimeConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.QueueCap != 7 || got.APIToken != "api" || got.MCPToken != "mcp" || !got.UpdatedAt.Equal(cfg.UpdatedAt) {
		t.Errorf("round trip mismatch: %+v", got)
	}

	// Zero UpdatedAt is stamped with the current time; upsert replaces the row.
	if err := repo.SaveRuntimeConfig(ctx, domain.RuntimeConfig{QueueCap: 9}); err != nil {
		t.Fatal(err)
	}
	got, _ = repo.GetRuntimeConfig(ctx)
	if got.QueueCap != 9 || got.APIToken != "" || time.Since(got.UpdatedAt) > time.Minute {
		t.Errorf("upsert mismatch: %+v", got)
	}
}

func TestRuntimeConfigRepo_CorruptJSONAndEmptyJSON(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	repo := repo_sqlite.NewRuntimeConfigRepo(r)
	ctx := context.Background()

	if _, err := r.DB().ExecContext(ctx,
		`UPDATE runtime_config SET json = '', updated_at = 0`); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetRuntimeConfig(ctx)
	if err != nil || got.QueueCap != 0 || !got.UpdatedAt.IsZero() {
		t.Fatalf("empty json: %+v %v", got, err)
	}

	if _, err := r.DB().ExecContext(ctx, `UPDATE runtime_config SET json = '{bad'`); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetRuntimeConfig(ctx); err == nil {
		t.Error("corrupt json: expected error")
	}
}

func TestRuntimeConfigRepo_ErrorsOnClosedDB(t *testing.T) {
	repo := repo_sqlite.NewRuntimeConfigRepo(closedRepo(t))
	ctx := context.Background()
	if _, err := repo.GetRuntimeConfig(ctx); err == nil || errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Get: expected non-NotFound error, got %v", err)
	}
	if err := repo.SaveRuntimeConfig(ctx, domain.RuntimeConfig{}); err == nil {
		t.Error("Save: expected error")
	}
}

func TestModelCapabilityRepo_ParsesSQLiteCurrentTimestampFormat(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	repo := repo_sqlite.NewModelCapabilityRepo(r)
	if err := repo.Save(domain.ModelCapabilityProfile{ModelID: "m"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.DB().Exec(
		`UPDATE model_capabilities SET created_at = '2024-05-06 07:08:09', updated_at = 'garbage'`); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByModelID("m")
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC); !got.CreatedAt.Equal(want) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, want)
	}
	if !got.UpdatedAt.IsZero() {
		t.Errorf("unparseable UpdatedAt should be zero, got %v", got.UpdatedAt)
	}
}
