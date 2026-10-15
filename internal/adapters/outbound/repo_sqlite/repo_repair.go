package repo_sqlite

import (
	"context"
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

	changed := map[string]bool{}

	rows, err := tx.QueryContext(ctx, `SELECT id, status FROM tasks`)
	if err != nil {
		return 0, fmt.Errorf("sqlite: repair: %w", err)
	}
	type fix struct{ id, status string }
	var statusFixes []fix
	for rows.Next() {
		var id, status string
		if err := rows.Scan(&id, &status); err != nil {
			_ = rows.Close()
			return 0, fmt.Errorf("sqlite: repair: %w", err)
		}
		if want := normalizeStatus(status); string(want) != status && isKnownStatus(want) {
			statusFixes = append(statusFixes, fix{id, string(want)})
		}
	}
	if err := rows.Close(); err != nil {
		return 0, fmt.Errorf("sqlite: repair: %w", err)
	}
	for _, f := range statusFixes {
		if _, err := tx.ExecContext(ctx, `UPDATE tasks SET status = ? WHERE id = ?`, f.status, f.id); err != nil {
			return 0, fmt.Errorf("sqlite: repair status of %s: %w", f.id, err)
		}
		changed[f.id] = true
	}

	times, err := tx.QueryContext(ctx, `SELECT id, created_at, updated_at FROM tasks
		WHERE typeof(created_at) != 'integer' OR typeof(updated_at) != 'integer'
		   OR created_at < ? OR updated_at < ?`, plausibleMillis, plausibleMillis)
	if err != nil {
		return 0, fmt.Errorf("sqlite: repair: %w", err)
	}
	type timeFix struct {
		id               string
		created, updated int64
	}
	var timeFixes []timeFix
	for times.Next() {
		var id string
		var createdRaw, updatedRaw any
		if err := times.Scan(&id, &createdRaw, &updatedRaw); err != nil {
			_ = times.Close()
			return 0, fmt.Errorf("sqlite: repair: %w", err)
		}
		created, cerr := millisOf(createdRaw)
		updated, uerr := millisOf(updatedRaw)
		cOK, uOK := cerr == nil && created >= plausibleMillis, uerr == nil && updated >= plausibleMillis
		now := time.Now().UnixMilli()
		switch {
		case cOK && !uOK:
			updated = created
		case !cOK && uOK:
			created = updated
		case !cOK && !uOK:
			created, updated = now, now
		}
		timeFixes = append(timeFixes, timeFix{id, created, updated})
	}
	if err := times.Close(); err != nil {
		return 0, fmt.Errorf("sqlite: repair: %w", err)
	}
	for _, f := range timeFixes {
		if _, err := tx.ExecContext(ctx, `UPDATE tasks SET created_at = ?, updated_at = ? WHERE id = ?`, f.created, f.updated, f.id); err != nil {
			return 0, fmt.Errorf("sqlite: repair times of %s: %w", f.id, err)
		}
		changed[f.id] = true
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("sqlite: repair: %w", err)
	}
	if len(changed) > 0 {
		log.Printf("repo_sqlite: repaired %d task rows written by another tool (statuses or times)", len(changed))
	}
	return len(changed), nil
}

// isKnownStatus reports whether s is one of the daemon's task statuses.
func isKnownStatus(s domain.TaskStatus) bool {
	switch domain.TaskStatus(strings.ToUpper(string(s))) {
	case domain.StatusQueued, domain.StatusProcessing, domain.StatusCompleted, domain.StatusFailed,
		domain.StatusCancelled, domain.StatusTooLarge, domain.StatusNoProvider, domain.StatusDraft, domain.StatusBacklog:
		return true
	}
	return false
}
