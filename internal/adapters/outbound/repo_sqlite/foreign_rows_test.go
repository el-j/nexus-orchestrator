package repo_sqlite_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// Agents sometimes edit the database directly (sqlite3 "UPDATE tasks SET status='IN_PROGRESS',
// updated_at=datetime('now')"). One such row used to make the task list fail with
// "converting driver.Value type string to int64", the daemon answered 500, and every client
// (the VS Code extension: "daemon offline") lost the whole list.
func TestTasksWrittenByHandAreStillReadable(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	for _, id := range []string{"ok", "hand", "alias", "bad"} {
		if err := r.Save(ctx, domain.Task{ID: id, ProjectPath: "/p", TargetFile: "f", Instruction: "i", Status: domain.StatusCompleted, CreatedAt: time.Now(), UpdatedAt: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := r.DB().Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`UPDATE tasks SET updated_at = '2026-10-15 12:25:52', status = 'IN_PROGRESS' WHERE id = 'hand'`)
	exec(`UPDATE tasks SET updated_at = '2026-10-15T12:25:52+02:00', created_at = '1792053845616', status = 'done' WHERE id = 'alias'`)
	exec(`UPDATE tasks SET updated_at = 'garbage' WHERE id = 'bad'`)

	hand, err := r.GetByID(ctx, "hand")
	if err != nil {
		t.Fatalf("a datetime text in updated_at must be read: %v", err)
	}
	if want := time.Date(2026, 10, 15, 12, 25, 52, 0, time.UTC); !hand.UpdatedAt.Equal(want) {
		t.Errorf("updated_at = %v, want %v (SQLite's datetime('now') is UTC)", hand.UpdatedAt, want)
	}
	if hand.Status != domain.StatusProcessing {
		t.Errorf("IN_PROGRESS is read as %s, want PROCESSING", hand.Status)
	}
	alias, err := r.GetByID(ctx, "alias")
	if err != nil || alias.Status != domain.StatusCompleted || alias.CreatedAt.UnixMilli() != 1792053845616 ||
		!alias.UpdatedAt.Equal(time.Date(2026, 10, 15, 10, 25, 52, 0, time.UTC)) {
		t.Errorf("alias row = %+v, %v", alias, err)
	}

	// One row nobody can decode must not take the list down with it.
	all, err := r.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll failed because of one bad row: %v", err)
	}
	ids := map[string]bool{}
	for _, tk := range all {
		ids[tk.ID] = true
	}
	if !ids["ok"] || !ids["hand"] || !ids["alias"] || ids["bad"] {
		t.Errorf("GetAll ids = %v; want every readable row and not the broken one", ids)
	}
	if _, err := r.GetByID(ctx, "bad"); err == nil {
		t.Error("asking for the broken row itself must still report the problem")
	}
	for name, f := range map[string]func() ([]domain.Task, error){
		"GetPending":       func() ([]domain.Task, error) { return r.GetPending(ctx) },
		"GetByProjectPath": func() ([]domain.Task, error) { return r.GetByProjectPath(ctx, "/p") },
	} {
		if _, err := f(); err != nil {
			t.Errorf("%s failed because of one bad row: %v", name, err)
		}
	}
}

func TestTaskTimestampsInOtherStorageClasses(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	for _, id := range []string{"blob", "real", "created-garbage"} {
		if err := r.Save(ctx, domain.Task{ID: id, ProjectPath: "/p", TargetFile: "f", Instruction: "i", Status: domain.StatusQueued, CreatedAt: time.Now(), UpdatedAt: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}
	for _, q := range []string{
		`UPDATE tasks SET updated_at = CAST('1792053845616' AS BLOB) WHERE id = 'blob'`,
		`UPDATE tasks SET updated_at = 1792053845616.0 WHERE id = 'real'`,
		`UPDATE tasks SET created_at = 'garbage' WHERE id = 'created-garbage'`,
	} {
		if _, err := r.DB().Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"blob", "real"} {
		got, err := r.GetByID(ctx, id)
		if err != nil || got.UpdatedAt.UnixMilli() != 1792053845616 {
			t.Errorf("%s: %v %v", id, got.UpdatedAt, err)
		}
	}
	if _, err := r.GetByID(ctx, "created-garbage"); err == nil || !strings.Contains(err.Error(), "created_at") {
		t.Errorf("a created_at that is no time must be reported: %v", err)
	}
}

// The daemon repairs rows another tool wrote, so its SQL queries (status = 'PROCESSING' for the
// watchdog, ordering by updated_at) see them too, not only the tolerant reader.
func TestRepairForeignRows(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	now := time.Now()
	for _, id := range []string{"fine", "text", "alias", "seconds", "bad"} {
		if err := r.Save(ctx, domain.Task{ID: id, ProjectPath: "/p", TargetFile: "f", Instruction: "i", Status: domain.StatusQueued, CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	for _, q := range []string{
		`UPDATE tasks SET updated_at = '2026-10-15 12:25:52' WHERE id = 'text'`,
		`UPDATE tasks SET status = 'in_progress' WHERE id = 'alias'`,
		`UPDATE tasks SET updated_at = 1792053 WHERE id = 'seconds'`, // not a plausible millisecond time
		`UPDATE tasks SET updated_at = 'garbage' WHERE id = 'bad'`,
	} {
		if _, err := r.DB().Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	n, err := r.RepairForeignRows(ctx)
	if err != nil || n != 4 {
		t.Fatalf("repaired %d rows (%v), want the 4 foreign ones", n, err)
	}
	for _, c := range []struct{ id, col, want string }{
		{"text", "typeof(updated_at)", "integer"},
		{"text", "updated_at", "1792067152000"}, // 2026-10-15 12:25:52 UTC
		{"alias", "status", "PROCESSING"},
		{"seconds", "updated_at = created_at", "1"},
		{"bad", "updated_at = created_at", "1"},
		{"fine", "typeof(updated_at)", "integer"},
	} {
		var got string
		if err := r.DB().QueryRow(`SELECT CAST(`+c.col+` AS TEXT) FROM tasks WHERE id = ?`, c.id).Scan(&got); err != nil || got != c.want {
			t.Errorf("%s %s = %q (%v), want %q", c.id, c.col, got, err, c.want)
		}
	}
	if n, _ := r.RepairForeignRows(ctx); n != 0 {
		t.Errorf("a second run repaired %d rows, want none", n)
	}
	all, err := r.GetAll(ctx)
	if err != nil || len(all) != 5 {
		t.Errorf("GetAll after repair: %d rows, %v", len(all), err)
	}
}
