package repo_sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// plausibleMillis is the earliest Unix-millisecond timestamp (1973) a task can have. A smaller
// number in a time column was written in the wrong unit (seconds) by another tool.
const plausibleMillis = 100_000_000_000

// RepairForeignRows fixes task rows that another tool wrote straight into the database
// (an agent running sqlite3): statuses such as "in_progress" or "done" become the daemon's
// statuses, and a time column holding text, or a number that cannot be milliseconds, becomes
// Unix milliseconds again (an unusable time falls back to the row's other time, or to now).
// Without it the watchdog and the queue queries, which compare in SQL, would never see those
// rows. It returns how many rows it changed and is a no-op on a healthy database.
func (r *Repository) RepairForeignRows(ctx context.Context) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("sqlite: repair: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	fixes, err := foreignRowFixes(ctx, tx)
	if err != nil {
		return 0, err
	}
	for _, f := range fixes {
		if f.status != "" {
			if _, err := tx.ExecContext(ctx, `UPDATE tasks SET status = ? WHERE id = ?`, f.status, f.id); err != nil {
				return 0, fmt.Errorf("sqlite: repair status of %s: %w", f.id, err)
			}
		}
		if f.fixTimes {
			if _, err := tx.ExecContext(ctx, `UPDATE tasks SET created_at = ?, updated_at = ? WHERE id = ?`, f.created, f.updated, f.id); err != nil {
				return 0, fmt.Errorf("sqlite: repair times of %s: %w", f.id, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("sqlite: repair: %w", err)
	}
	if len(fixes) > 0 {
		log.Printf("repo_sqlite: repaired %d task rows written by another tool (statuses or times)", len(fixes))
	}
	return len(fixes), nil
}

// rowFix is what has to change in one task row.
type rowFix struct {
	id               string
	status           string // the daemon's status to write, empty when the status is fine
	fixTimes         bool
	created, updated int64
}

// foreignRowFixes reads every task row once and lists the ones that need repair. The rows are
// read to the end and closed before anything is written, as the transaction holds the only connection.
func foreignRowFixes(ctx context.Context, tx *sql.Tx) ([]rowFix, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id, status, created_at, updated_at FROM tasks`)
	if err != nil {
		return nil, fmt.Errorf("sqlite: repair: %w", err)
	}
	var fixes []rowFix
	for rows.Next() {
		var id, status string
		var createdRaw, updatedRaw any
		if err := rows.Scan(&id, &status, &createdRaw, &updatedRaw); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("sqlite: repair: %w", err)
		}
		fix := rowFix{id: id}
		if want := normalizeStatus(status); string(want) != status && isKnownStatus(want) {
			fix.status = string(want)
		}
		fix.created, fix.updated, fix.fixTimes = repairedTimes(createdRaw, updatedRaw)
		if fix.status != "" || fix.fixTimes {
			fixes = append(fixes, fix)
		}
	}
	iterErr := rows.Err()
	_ = rows.Close()
	if iterErr != nil {
		return nil, fmt.Errorf("sqlite: repair: %w", iterErr)
	}
	return fixes, nil
}

// repairedTimes returns the milliseconds to store for a row whose time columns are not both plain
// integers of a plausible size, and true when the row needs the repair. An unusable time falls back
// to the row's other time, or to now.
func repairedTimes(createdRaw, updatedRaw any) (created, updated int64, fix bool) {
	cInt, cIsInt := createdRaw.(int64)
	uInt, uIsInt := updatedRaw.(int64)
	if cIsInt && uIsInt && cInt >= plausibleMillis && uInt >= plausibleMillis {
		return 0, 0, false
	}
	created, cerr := millisOf(createdRaw)
	updated, uerr := millisOf(updatedRaw)
	cOK, uOK := cerr == nil && created >= plausibleMillis, uerr == nil && updated >= plausibleMillis
	switch {
	case cOK && !uOK:
		updated = created
	case !cOK && uOK:
		created = updated
	case !cOK && !uOK:
		now := time.Now().UnixMilli()
		created, updated = now, now
	}
	return created, updated, true
}

// isKnownStatus reports whether s is one of the daemon's task statuses.
func isKnownStatus(s domain.TaskStatus) bool {
	switch domain.TaskStatus(strings.ToUpper(string(s))) {
	case domain.StatusQueued, domain.StatusProcessing, domain.StatusCompleted, domain.StatusFailed,
		domain.StatusCancelled, domain.StatusTooLarge, domain.StatusNoProvider, domain.StatusWaiting, domain.StatusDraft, domain.StatusBacklog:
		return true
	}
	return false
}
