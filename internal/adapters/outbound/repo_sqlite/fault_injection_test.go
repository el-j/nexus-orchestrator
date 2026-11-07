package repo_sqlite_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
)

// The helpers below break one table in a controlled way so the error branches of the repositories
// run: a row the code cannot scan, a row whose evaluation fails halfway through a result set, a
// statement that aborts, a statement that is silently skipped, and a commit that is refused.

// badRow gives the SQL expression for chosen columns of an injected row; every other column is
// NULL (a value no repository can scan into a string or number).
type badRow map[string]string

// columnsOf returns the column names of a table.
func columnsOf(t *testing.T, r *repo_sqlite.Repository, table string) []string {
	t.Helper()
	rows, err := r.DB().Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var cid, notnull, pk int
		var name, typ string
		var dflt any
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			t.Fatal(err)
		}
		cols = append(cols, name)
	}
	return cols
}

// mustExec runs SQL against the repository's database.
func mustExec(t *testing.T, r *repo_sqlite.Repository, q string) {
	t.Helper()
	if _, err := r.DB().Exec(q); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
}

// viewWithRow replaces the table by a view that returns its rows and one more, built from row.
// Columns not named in row are NULL, or, when failing is true, an expression that raises an
// integer-overflow error as soon as the row is read.
func viewWithRow(t *testing.T, r *repo_sqlite.Repository, table string, row badRow, failing bool) {
	t.Helper()
	cols := columnsOf(t, r, table)
	var exprs []string
	for _, c := range cols {
		switch expr, ok := row[c]; {
		case ok:
			exprs = append(exprs, expr)
		case failing:
			exprs = append(exprs, "abs(-9223372036854775808)")
		default:
			exprs = append(exprs, "NULL")
		}
	}
	mustExec(t, r, fmt.Sprintf("ALTER TABLE %s RENAME TO %s_real", table, table))
	mustExec(t, r, fmt.Sprintf("CREATE VIEW %s AS SELECT * FROM %s_real UNION ALL SELECT %s", table, table, strings.Join(exprs, ", ")))
}

// abortOn makes every statement of the given kind (INSERT, UPDATE, DELETE) on the table fail.
func abortOn(t *testing.T, r *repo_sqlite.Repository, table, event string) {
	t.Helper()
	mustExec(t, r, fmt.Sprintf("CREATE TRIGGER inject_%s_%s BEFORE %s ON %s BEGIN SELECT RAISE(ABORT, 'injected failure'); END", table, event, event, table))
}

// skipOn makes statements of the given kind on the table do nothing (zero rows affected, no error).
func skipOn(t *testing.T, r *repo_sqlite.Repository, table, event string) {
	t.Helper()
	mustExec(t, r, fmt.Sprintf("CREATE TRIGGER skip_%s_%s BEFORE %s ON %s BEGIN SELECT RAISE(IGNORE); END", table, event, event, table))
}

// refuseCommitAfter lets the statement run but makes the transaction's COMMIT fail: the trigger
// leaves a deferred foreign key violation behind.
func refuseCommitAfter(t *testing.T, r *repo_sqlite.Repository, table, event string) {
	t.Helper()
	mustExec(t, r, fmt.Sprintf("CREATE TABLE ghost_%s (ref TEXT REFERENCES %s(id) DEFERRABLE INITIALLY DEFERRED)", table, table))
	mustExec(t, r, fmt.Sprintf("CREATE TRIGGER ghost_%s_%s AFTER %s ON %s BEGIN INSERT INTO ghost_%s(ref) VALUES ('no such row'); END", table, event, event, table, table))
}

func TestListQueries_ReportRowsTheyCannotScanAndFailuresMidway(t *testing.T) {
	ctx := context.Background()
	for name, tc := range map[string]struct {
		table string
		row   badRow
		call  func(r *repo_sqlite.Repository) error
	}{
		"activities": {"ai_activities", nil, func(r *repo_sqlite.Repository) error {
			_, err := repo_sqlite.NewActivityRepo(r).ListActivities(ctx, domain.ActivityFilter{})
			return err
		}},
		"ai sessions": {"ai_sessions", nil, func(r *repo_sqlite.Repository) error {
			_, err := repo_sqlite.NewAISessionRepo(r).ListAISessions(ctx)
			return err
		}},
		"discovered agents": {"discovered_agents", nil, func(r *repo_sqlite.Repository) error {
			_, err := repo_sqlite.NewDiscoveredAgentRepo(r).ListDiscoveredAgents(ctx)
			return err
		}},
		"knowledge status": {"project_knowledge", badRow{"project_path": "'/p'"}, func(r *repo_sqlite.Repository) error {
			_, err := repo_sqlite.NewKnowledgeRepo(r).GetStatus(ctx, "/p")
			return err
		}},
		"knowledge by project": {"project_knowledge", badRow{"project_path": "'/p'"}, func(r *repo_sqlite.Repository) error {
			_, err := repo_sqlite.NewKnowledgeRepo(r).GetByProject(ctx, "/p")
			return err
		}},
		"model capabilities": {"model_capabilities", nil, func(r *repo_sqlite.Repository) error {
			_, err := repo_sqlite.NewModelCapabilityRepo(r).GetAll(ctx)
			return err
		}},
		"plan files": {"discovered_plan_files", nil, func(r *repo_sqlite.Repository) error {
			_, err := repo_sqlite.NewPlanFileRepo(r).ListPlanFiles(ctx, "")
			return err
		}},
		"provider configs": {"provider_configs", nil, func(r *repo_sqlite.Repository) error {
			_, err := repo_sqlite.NewProviderConfigRepo(r).ListProviderConfigs(ctx)
			return err
		}},
		"usage": {"llm_usage", badRow{"at": "1"}, func(r *repo_sqlite.Repository) error {
			_, err := repo_sqlite.NewUsageRepo(r).Summarize(ctx, time.Time{})
			return err
		}},
	} {
		for _, failing := range []bool{false, true} {
			kind := "a row that cannot be scanned"
			if failing {
				kind = "a failure while reading the rows"
			}
			t.Run(name+": "+kind, func(t *testing.T) {
				r := newTestRepo(t)
				defer r.Close()
				viewWithRow(t, r, tc.table, tc.row, failing)
				if err := tc.call(r); err == nil {
					t.Fatal("want an error, got none")
				}
			})
		}
	}
}
