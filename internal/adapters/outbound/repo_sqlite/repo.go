// Package repo_sqlite implements the TaskRepository and SessionRepository
// ports using SQLite via mattn/go-sqlite3.
package repo_sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"nexus-orchestrator/internal/core/domain"

	_ "github.com/mattn/go-sqlite3" // register the sqlite3 driver via its init() side-effect
)

// Repository implements ports.TaskRepository using a local SQLite database.
type Repository struct {
	db *sql.DB
}

// New opens (or creates) a SQLite database at the given path and runs migrations.
func New(dbPath string) (*Repository, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open: %w", err)
	}
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
	} {
		if _, err := db.ExecContext(context.Background(), pragma); err != nil {
			return nil, fmt.Errorf("sqlite: %s: %w", pragma, err)
		}
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := migrate(db); err != nil {
		return nil, err
	}
	return &Repository{db: db}, nil
}

const taskColumns = `id, project_path, target_file, instruction, context_files, status, created_at, updated_at, logs, model_id, provider_hint, command, provider_name, priority, tags, retry_count, ai_session_id, verification_command, max_correction_turns, verification_output, role, agentic, max_agent_turns`

// Save inserts a new Task record.
func (r *Repository) Save(ctx context.Context, t domain.Task) error {
	t.ProjectPath = filepath.Clean(t.ProjectPath)
	ctxJSON := marshal(t.ContextFiles)
	tagsJSON := marshalTags(t.Tags)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tasks (`+taskColumns+`)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.ProjectPath, t.TargetFile, t.Instruction,
		string(ctxJSON), string(t.Status),
		t.CreatedAt.UnixMilli(), t.UpdatedAt.UnixMilli(),
		t.Logs, t.ModelID, t.ProviderHint, string(t.Command),
		t.ProviderName, t.Priority, string(tagsJSON), t.RetryCount, t.AISessionID,
		t.VerificationCommand, t.MaxCorrectionTurns, t.VerificationOutput,
		t.Role, t.Agentic, t.MaxAgentTurns,
	)
	if err != nil {
		return fmt.Errorf("sqlite: insert task: %w", err)
	}
	return nil
}

// GetByID retrieves a single task by its ID.
// Returns domain.ErrNotFound when no row matches.
func (r *Repository) GetByID(ctx context.Context, id string) (domain.Task, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+taskColumns+` FROM tasks WHERE id = ?`, id)
	t, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Task{}, fmt.Errorf("sqlite: get task: %w", domain.ErrNotFound)
	}
	return t, err
}

// GetPending returns all tasks in QUEUED or PROCESSING state.
func (r *Repository) GetPending(ctx context.Context) ([]domain.Task, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+taskColumns+` FROM tasks WHERE status IN ('QUEUED','PROCESSING') ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("sqlite: query pending: %w", err)
	}
	defer rows.Close()

	return collectTasks(rows)
}

// ClaimNextQueued atomically claims the oldest queued task and marks it PROCESSING.
func (r *Repository) ClaimNextQueued(ctx context.Context) (domain.Task, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Task{}, fmt.Errorf("sqlite: begin claim next queued: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	row := tx.QueryRowContext(ctx,
		`SELECT `+taskColumns+` FROM tasks WHERE status = 'QUEUED' ORDER BY created_at ASC LIMIT 1`,
	)
	task, err := scanTask(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Task{}, fmt.Errorf("sqlite: claim next queued: %w", domain.ErrNotFound)
		}
		return domain.Task{}, fmt.Errorf("sqlite: claim next queued: %w", err)
	}

	res, err := tx.ExecContext(ctx,
		`UPDATE tasks SET status = ?, updated_at = ? WHERE id = ? AND status = 'QUEUED'`,
		string(domain.StatusProcessing), time.Now().UnixMilli(), task.ID,
	)
	if err != nil {
		return domain.Task{}, fmt.Errorf("sqlite: claim next queued: %w", err)
	}
	n := affected(res)
	if n == 0 {
		return domain.Task{}, fmt.Errorf("sqlite: claim next queued: %w", domain.ErrNotFound)
	}
	if err := tx.Commit(); err != nil {
		return domain.Task{}, fmt.Errorf("sqlite: claim next queued commit: %w", err)
	}
	task.Status = domain.StatusProcessing
	task.UpdatedAt = time.Now()
	return task, nil
}

// UpdateStatus changes the status of a task identified by id.
func (r *Repository) UpdateStatus(ctx context.Context, id string, status domain.TaskStatus) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?`,
		string(status), time.Now().UnixMilli(), id,
	)
	if err != nil {
		return fmt.Errorf("sqlite: update status: %w", err)
	}
	n := affected(res)
	if n == 0 {
		return fmt.Errorf("sqlite: update status: %w", domain.ErrNotFound)
	}
	return nil
}

// UpdateStatusIfCurrent changes the status only when the current status matches from.
func (r *Repository) UpdateStatusIfCurrent(ctx context.Context, id string, from, to domain.TaskStatus) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE tasks SET status = ?, updated_at = ? WHERE id = ? AND status = ?`,
		string(to), time.Now().UnixMilli(), id, string(from),
	)
	if err != nil {
		return false, fmt.Errorf("sqlite: update status if current: %w", err)
	}
	n := affected(res)
	return n == 1, nil
}

// UpdateLogs replaces the logs field for the task identified by id.
func (r *Repository) UpdateLogs(ctx context.Context, id, logs string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE tasks SET logs = ?, updated_at = ? WHERE id = ?`,
		logs, time.Now().UnixMilli(), id,
	)
	if err != nil {
		return fmt.Errorf("sqlite: update logs: %w", err)
	}
	n := affected(res)
	if n == 0 {
		return fmt.Errorf("sqlite: update logs: %w", domain.ErrNotFound)
	}
	return nil
}

// GetAll returns every task, ordered by creation time descending.
func (r *Repository) GetAll(ctx context.Context) ([]domain.Task, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+taskColumns+` FROM tasks ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("sqlite: query all tasks: %w", err)
	}
	defer rows.Close()

	return collectTasks(rows)
}

// GetByProjectPath returns all tasks for the given project path, ordered by creation time descending.
func (r *Repository) GetByProjectPath(ctx context.Context, projectPath string) ([]domain.Task, error) {
	projectPath = filepath.Clean(projectPath)
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+taskColumns+` FROM tasks WHERE project_path = ? ORDER BY created_at DESC`,
		projectPath,
	)
	if err != nil {
		return nil, fmt.Errorf("sqlite: query by project path: %w", err)
	}
	defer rows.Close()

	return collectTasks(rows)
}

// GetByProjectPathAndStatus returns tasks for a project filtered by one or more statuses,
// ordered by priority ASC then created_at ASC. Returns an empty slice (not an error) if none match.
func (r *Repository) GetByProjectPathAndStatus(ctx context.Context, projectPath string, statuses ...domain.TaskStatus) ([]domain.Task, error) {
	if len(statuses) == 0 {
		return []domain.Task{}, nil
	}
	projectPath = filepath.Clean(projectPath)
	placeholders := make([]string, len(statuses))
	args := make([]interface{}, 0, 1+len(statuses))
	args = append(args, projectPath)
	for i, s := range statuses {
		placeholders[i] = "?"
		args = append(args, string(s))
	}
	// Only "?" placeholders are interpolated; every status is bound via args.
	query := fmt.Sprintf( //nolint:gosec // G201: no user input in the SQL text
		`SELECT `+taskColumns+` FROM tasks WHERE project_path = ? AND status IN (%s) ORDER BY priority ASC, created_at ASC`,
		strings.Join(placeholders, ","),
	)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite: query by project path and status: %w", err)
	}
	defer rows.Close()
	return collectTasks(rows)
}

// Update persists changes to an existing task's mutable fields.
// Returns domain.ErrNotFound if no row with t.ID exists.
func (r *Repository) Update(ctx context.Context, t domain.Task) error {
	t.UpdatedAt = time.Now()
	tagsJSON := []byte("[]")
	if len(t.Tags) > 0 {
		if b, mErr := json.Marshal(t.Tags); mErr == nil {
			tagsJSON = b
		}
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE tasks SET instruction=?, target_file=?, provider_name=?, priority=?, tags=?, status=?, retry_count=?, ai_session_id=?, verification_command=?, max_correction_turns=?, verification_output=?, updated_at=? WHERE id=?`,
		t.Instruction, t.TargetFile, t.ProviderName, t.Priority, string(tagsJSON), string(t.Status), t.RetryCount, t.AISessionID, t.VerificationCommand, t.MaxCorrectionTurns, t.VerificationOutput, t.UpdatedAt.UnixMilli(), t.ID,
	)
	if err != nil {
		return fmt.Errorf("sqlite: update task: %w", err)
	}
	n := affected(res)
	if n == 0 {
		return fmt.Errorf("sqlite: update task: %w", domain.ErrNotFound)
	}
	return nil
}

// Close releases the underlying database connection.
func (r *Repository) Close() error { return r.db.Close() }

// GetTasksBySessionID returns all tasks claimed by the given AI session, ordered by creation time descending.
func (r *Repository) GetTasksBySessionID(ctx context.Context, sessionID string) ([]domain.Task, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+taskColumns+` FROM tasks WHERE ai_session_id = ? ORDER BY created_at DESC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("sqlite: query tasks by session id: %w", err)
	}
	defer rows.Close()

	return collectTasks(rows)
}

// GetStaleProcessing returns tasks in PROCESSING status that haven't been updated for the threshold duration.
func (r *Repository) GetStaleProcessing(ctx context.Context, threshold time.Duration) ([]domain.Task, error) {
	cutoff := time.Now().Add(-threshold).UnixMilli()
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+taskColumns+` FROM tasks WHERE status = 'PROCESSING' AND updated_at < ? ORDER BY updated_at ASC`,
		cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("sqlite: query stale processing: %w", err)
	}
	defer rows.Close()

	return collectTasks(rows)
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanTask(s scanner) (domain.Task, error) {
	var t domain.Task
	var status string
	var command string
	var ctxJSON string
	var tagsJSON string
	var createdRaw, updatedRaw any

	if err := s.Scan(
		&t.ID, &t.ProjectPath, &t.TargetFile, &t.Instruction,
		&ctxJSON, &status, &createdRaw, &updatedRaw, &t.Logs,
		&t.ModelID, &t.ProviderHint, &command, &t.ProviderName, &t.Priority, &tagsJSON, &t.RetryCount, &t.AISessionID,
		&t.VerificationCommand, &t.MaxCorrectionTurns, &t.VerificationOutput,
		&t.Role, &t.Agentic, &t.MaxAgentTurns,
	); err != nil {
		return t, fmt.Errorf("sqlite: scan task: %w", err)
	}

	createdMS, err := millisOf(createdRaw)
	if err != nil {
		return t, fmt.Errorf("sqlite: task %s: created_at: %w", t.ID, err)
	}
	updatedMS, err := millisOf(updatedRaw)
	if err != nil {
		return t, fmt.Errorf("sqlite: task %s: updated_at: %w", t.ID, err)
	}
	t.Status = normalizeStatus(status)
	t.Command = domain.CommandType(command)
	t.CreatedAt = time.UnixMilli(createdMS)
	t.UpdatedAt = time.UnixMilli(updatedMS)

	if err := json.Unmarshal([]byte(ctxJSON), &t.ContextFiles); err != nil {
		return t, fmt.Errorf("sqlite: unmarshal context files: %w", err)
	}
	if tagsJSON != "" && tagsJSON != "[]" {
		if err := json.Unmarshal([]byte(tagsJSON), &t.Tags); err != nil {
			log.Printf("repo_sqlite: task %s: unmarshal tags: %v", t.ID, err)
		}
	}
	return t, nil
}

// collectTasks reads every row of a task query. A row that cannot be decoded is logged and
// skipped: one damaged or hand-edited row must not make the whole list unavailable (every client
// would show the daemon as offline). Asking for that task by id still reports the problem.
func collectTasks(rows *sql.Rows) ([]domain.Task, error) {
	tasks := []domain.Task{}
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			log.Printf("repo_sqlite: skipping an unreadable task row: %v", err)
			continue
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// timeLayouts are the text forms of a time that tools other than this daemon write: SQLite's own
// datetime('now') (UTC, no zone), and ISO 8601.
var timeLayouts = []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05", time.RFC3339Nano, time.RFC3339}

// millisOf reads a timestamp column. The daemon stores Unix milliseconds; a column edited by hand
// may hold the number as text or a datetime text, which is read as UTC when it names no zone.
func millisOf(v any) (int64, error) {
	switch x := v.(type) {
	case int64:
		return x, nil
	case float64:
		return int64(x), nil
	case time.Time:
		return x.UnixMilli(), nil
	case []byte:
		return millisOf(string(x))
	case string:
		x = strings.TrimSpace(x)
		if n, err := strconv.ParseInt(x, 10, 64); err == nil {
			return n, nil
		}
		for _, layout := range timeLayouts {
			if t, err := time.ParseInLocation(layout, x, time.UTC); err == nil {
				return t.UnixMilli(), nil
			}
		}
		return 0, fmt.Errorf("cannot read %q as a time", x)
	default:
		return 0, fmt.Errorf("cannot read a %T as a time", v)
	}
}

// statusAliases maps the words other tools use for a task's state to the daemon's statuses.
var statusAliases = map[string]domain.TaskStatus{
	"IN_PROGRESS": domain.StatusProcessing, "RUNNING": domain.StatusProcessing,
	"DONE": domain.StatusCompleted, "COMPLETE": domain.StatusCompleted,
	"ERROR": domain.StatusFailed, "PENDING": domain.StatusQueued, "TODO": domain.StatusQueued,
}

// normalizeStatus reads a status column, tolerating case and the aliases above.
func normalizeStatus(s string) domain.TaskStatus {
	if alias, ok := statusAliases[strings.ToUpper(strings.TrimSpace(s))]; ok {
		return alias
	}
	return domain.TaskStatus(s)
}
