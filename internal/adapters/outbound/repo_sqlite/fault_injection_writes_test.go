package repo_sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
)

func TestClaimNextQueued_ReportsAFailedClaimASkippedClaimAndARefusedCommit(t *testing.T) {
	ctx := context.Background()
	queued := newTask("q1", domain.StatusQueued, time.Now())
	for name, tc := range map[string]struct {
		inject func(t *testing.T, r *repo_sqlite.Repository)
		check  func(t *testing.T, err error)
	}{
		"the update fails": {
			func(t *testing.T, r *repo_sqlite.Repository) { abortOn(t, r, "tasks", "UPDATE") },
			func(t *testing.T, err error) {
				if err == nil || !strings.Contains(err.Error(), "injected failure") {
					t.Errorf("err = %v", err)
				}
			},
		},
		"the update touches no row (another worker took the task)": {
			func(t *testing.T, r *repo_sqlite.Repository) { skipOn(t, r, "tasks", "UPDATE") },
			func(t *testing.T, err error) {
				if !errors.Is(err, domain.ErrNotFound) {
					t.Errorf("err = %v, want ErrNotFound", err)
				}
			},
		},
		"the commit is refused": {
			func(t *testing.T, r *repo_sqlite.Repository) { refuseCommitAfter(t, r, "tasks", "UPDATE") },
			func(t *testing.T, err error) {
				if err == nil || !strings.Contains(err.Error(), "commit") {
					t.Errorf("err = %v", err)
				}
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := newTestRepo(t)
			defer r.Close()
			if err := r.Save(ctx, queued); err != nil {
				t.Fatal(err)
			}
			tc.inject(t, r)
			_, err := r.ClaimNextQueued(ctx)
			tc.check(t, err)
		})
	}
}

func TestSessionRepo_ReportsUnreadableRowsAndFailedWrites(t *testing.T) {
	ctx := context.Background()
	msg := domain.Message{Role: domain.RoleUser, Content: "hi"}

	t.Run("a row that cannot be scanned stops an append", func(t *testing.T) {
		r := newTestRepo(t)
		defer r.Close()
		mustExec(t, r, `INSERT INTO sessions (id, project_path, messages, created_at, updated_at) VALUES ('s', '/p', '[]', 1, 'not a number')`)
		err := repo_sqlite.NewSessionRepo(r).AppendMessage(ctx, "/p", msg)
		if err == nil || !strings.Contains(err.Error(), "scan session for append") {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("a failed upsert is reported", func(t *testing.T) {
		r := newTestRepo(t)
		defer r.Close()
		abortOn(t, r, "sessions", "INSERT")
		err := repo_sqlite.NewSessionRepo(r).AppendMessage(ctx, "/p", msg)
		if err == nil || !strings.Contains(err.Error(), "upsert session") {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("stored messages that are not JSON are reported", func(t *testing.T) {
		r := newTestRepo(t)
		defer r.Close()
		mustExec(t, r, `INSERT INTO sessions (id, project_path, messages, created_at, updated_at) VALUES ('s', '/p', '{broken', 1, 1)`)
		_, err := repo_sqlite.NewSessionRepo(r).GetByProjectPath(ctx, "/p")
		if err == nil || !strings.Contains(err.Error(), "unmarshal messages") {
			t.Errorf("err = %v", err)
		}
	})
}

func TestAppendRoutedTaskID_ReportsAnUnreadableRowAFailedUpdateAndARefusedCommit(t *testing.T) {
	ctx := context.Background()
	save := func(t *testing.T, r *repo_sqlite.Repository) {
		t.Helper()
		if err := repo_sqlite.NewAISessionRepo(r).SaveAISession(ctx, domain.AISession{ID: "a", AgentName: "x", Source: "mcp"}); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("a row without the list cannot be read", func(t *testing.T) {
		r := newTestRepo(t)
		defer r.Close()
		viewWithRow(t, r, "ai_sessions", badRow{"id": "'a'"}, false)
		if err := repo_sqlite.NewAISessionRepo(r).AppendRoutedTaskID(ctx, "a", "t1"); err == nil || errors.Is(err, domain.ErrNotFound) {
			t.Errorf("err = %v, want a scan error", err)
		}
	})
	t.Run("the update fails", func(t *testing.T) {
		r := newTestRepo(t)
		defer r.Close()
		save(t, r)
		abortOn(t, r, "ai_sessions", "UPDATE")
		if err := repo_sqlite.NewAISessionRepo(r).AppendRoutedTaskID(ctx, "a", "t1"); err == nil || !strings.Contains(err.Error(), "update") {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("the commit is refused", func(t *testing.T) {
		r := newTestRepo(t)
		defer r.Close()
		save(t, r)
		refuseCommitAfter(t, r, "ai_sessions", "UPDATE")
		if err := repo_sqlite.NewAISessionRepo(r).AppendRoutedTaskID(ctx, "a", "t1"); err == nil || !strings.Contains(err.Error(), "commit") {
			t.Errorf("err = %v", err)
		}
	})
}

func TestRepairForeignRows_ReportsEveryStepThatFails(t *testing.T) {
	ctx := context.Background()
	foreign := func(t *testing.T, r *repo_sqlite.Repository) {
		t.Helper()
		mustExec(t, r, `INSERT INTO tasks (id, project_path, target_file, instruction, status, created_at, updated_at) VALUES ('f', '/p', 'f.go', 'x', 'in_progress', 'junk', 1)`)
	}
	for name, tc := range map[string]struct {
		prepare func(t *testing.T, r *repo_sqlite.Repository)
		want    string
	}{
		"the transaction cannot start": {func(t *testing.T, r *repo_sqlite.Repository) { _ = r.Close() }, "repair"},
		"the table cannot be read":     {func(t *testing.T, r *repo_sqlite.Repository) { mustExec(t, r, "DROP TABLE tasks") }, "repair"},
		"a row cannot be scanned": {func(t *testing.T, r *repo_sqlite.Repository) {
			mustExec(t, r, "DROP TABLE tasks")
			mustExec(t, r, "CREATE TABLE tasks (id TEXT, status TEXT, created_at, updated_at)")
			mustExec(t, r, "INSERT INTO tasks VALUES ('x', NULL, 1, 1)")
		}, "repair"},
		"reading stops halfway": {func(t *testing.T, r *repo_sqlite.Repository) {
			viewWithRow(t, r, "tasks", nil, true)
		}, "repair"},
		"the status cannot be written": {func(t *testing.T, r *repo_sqlite.Repository) {
			foreign(t, r)
			abortOn(t, r, "tasks", "UPDATE")
		}, "repair status of f"},
		"the times cannot be written": {func(t *testing.T, r *repo_sqlite.Repository) {
			mustExec(t, r, `INSERT INTO tasks (id, project_path, target_file, instruction, status, created_at, updated_at) VALUES ('f', '/p', 'f.go', 'x', 'QUEUED', 'junk', 1)`)
			abortOn(t, r, "tasks", "UPDATE")
		}, "repair times of f"},
		"the commit is refused": {func(t *testing.T, r *repo_sqlite.Repository) {
			foreign(t, r)
			refuseCommitAfter(t, r, "tasks", "UPDATE")
		}, "repair"},
	} {
		t.Run(name, func(t *testing.T) {
			r := newTestRepo(t)
			defer r.Close()
			tc.prepare(t, r)
			if _, err := r.RepairForeignRows(ctx); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want it to mention %q", err, tc.want)
			}
		})
	}
}

func TestColumnReaders_HandleEveryStorageClassAndSpelling(t *testing.T) {
	when := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	for name, tc := range map[string]struct {
		in   any
		want int64
		bad  bool
	}{
		"float":  {1.5e12, 1_500_000_000_000, false},
		"time":   {when, when.UnixMilli(), false},
		"bytes":  {[]byte("1700000000000"), 1_700_000_000_000, false},
		"text":   {"2026-10-31 12:00:00", when.UnixMilli(), false},
		"junk":   {"yesterday", 0, true},
		"a bool": {true, 0, true},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := repo_sqlite.MillisOf(tc.in)
			if (err != nil) != tc.bad || (!tc.bad && got != tc.want) {
				t.Errorf("MillisOf(%v) = %d, %v", tc.in, got, err)
			}
		})
	}
	if got := repo_sqlite.NormalizeStatus("Mystery"); got != "Mystery" {
		t.Errorf("an unknown status is kept as written, got %q", got)
	}
	if !repo_sqlite.ParseSQLiteTime("").IsZero() || !repo_sqlite.ParseSQLiteTime("not a time").IsZero() {
		t.Error("an empty or unreadable time is the zero time")
	}
}

func TestNew_ReportsAHandleThatCannotBeOpenedAndAMigrationThatFails(t *testing.T) {
	t.Run("opening fails", func(t *testing.T) {
		restore := repo_sqlite.SetOpenDatabase(func(string, string) (*sql.DB, error) { return nil, errors.New("no driver") })
		defer restore()
		if _, err := repo_sqlite.New(filepath.Join(t.TempDir(), "x.db")); err == nil || !strings.Contains(err.Error(), "open") {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("a view where the usage table belongs", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "x.db")
		db, err := sql.Open("sqlite3", path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec("CREATE VIEW llm_usage AS SELECT 1 AS at"); err != nil {
			t.Fatal(err)
		}
		_ = db.Close()
		if _, err := repo_sqlite.New(path); err == nil || !strings.Contains(err.Error(), "migrate llm_usage") {
			t.Errorf("err = %v", err)
		}
	})
}

func TestRepairForeignRows_AnUnusableCreatedTimeTakesTheUpdatedTime(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	mustExec(t, r, `INSERT INTO tasks (id, project_path, target_file, instruction, status, created_at, updated_at) VALUES ('f', '/p', 'f.go', 'x', 'QUEUED', 'junk', 1700000000000)`)
	if n, err := r.RepairForeignRows(context.Background()); err != nil || n != 1 {
		t.Fatalf("repaired %d rows: %v", n, err)
	}
	var created int64
	if err := r.DB().QueryRow(`SELECT created_at FROM tasks WHERE id = 'f'`).Scan(&created); err != nil || created != 1700000000000 {
		t.Errorf("created_at = %d, %v", created, err)
	}
}

func TestListDiscoveredAgents_ASavedTimeThatCannotBeReadLeavesTheTimeZero(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	mustExec(t, r, `INSERT INTO discovered_agents (id, kind, name, detection_method, process_name, cli_path, config_path, mcp_endpoint, is_running, pid, last_seen) VALUES ('a', 'k', 'n', 'm', '', '', '', '', 0, 0, 'not a time')`)
	agents, err := repo_sqlite.NewDiscoveredAgentRepo(r).ListDiscoveredAgents(context.Background())
	if err != nil || len(agents) != 1 || !agents[0].LastSeen.IsZero() {
		t.Errorf("agents = %+v, %v", agents, err)
	}
}

func TestListDiscoveredAgents_ReadsTheSavedTime(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	seen := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	repo := repo_sqlite.NewDiscoveredAgentRepo(r)
	if err := repo.UpsertDiscoveredAgent(context.Background(), domain.DiscoveredAgent{ID: "a", Kind: "copilot", Name: "n", LastSeen: seen}); err != nil {
		t.Fatal(err)
	}
	agents, err := repo.ListDiscoveredAgents(context.Background())
	if err != nil || len(agents) != 1 || !agents[0].LastSeen.Equal(seen) {
		t.Errorf("agents = %+v, %v", agents, err)
	}
}
