// Package repo_sqlite implements the TaskRepository and SessionRepository
// ports using SQLite via mattn/go-sqlite3.
package repo_sqlite

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
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
		if _, err := db.Exec(pragma); err != nil {
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

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id            TEXT    PRIMARY KEY,
			project_path  TEXT    NOT NULL,
			target_file   TEXT    NOT NULL,
			instruction   TEXT    NOT NULL,
			context_files TEXT    NOT NULL DEFAULT '[]',
			status        TEXT    NOT NULL DEFAULT 'QUEUED',
			created_at    INTEGER NOT NULL,
			updated_at    INTEGER NOT NULL,
			logs          TEXT    NOT NULL DEFAULT ''
		);
		CREATE TABLE IF NOT EXISTS sessions (
			id           TEXT    PRIMARY KEY,
			project_path TEXT    NOT NULL UNIQUE,
			messages     TEXT    NOT NULL DEFAULT '[]',
			created_at   INTEGER NOT NULL,
			updated_at   INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS provider_configs (
			id            TEXT    PRIMARY KEY,
			name          TEXT    NOT NULL,
			kind          TEXT    NOT NULL,
			base_url      TEXT    NOT NULL DEFAULT '',
			api_key       TEXT    NOT NULL DEFAULT '',
			model         TEXT    NOT NULL DEFAULT '',
			enabled       INTEGER NOT NULL DEFAULT 1,
			created_at    INTEGER NOT NULL,
			updated_at    INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS ai_sessions (
			id               TEXT PRIMARY KEY,
			source           TEXT NOT NULL,
			external_id      TEXT NOT NULL DEFAULT '',
			agent_name       TEXT NOT NULL,
			project_path     TEXT NOT NULL DEFAULT '',
			status           TEXT NOT NULL DEFAULT 'active',
			last_activity    DATETIME NOT NULL,
			routed_task_ids  TEXT NOT NULL DEFAULT '[]',
			created_at       DATETIME NOT NULL,
			updated_at       DATETIME NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("sqlite: migrate: %w", err)
	}
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
		CREATE INDEX IF NOT EXISTS idx_tasks_project_path ON tasks(project_path);
	`)
	if err != nil {
		return fmt.Errorf("sqlite: migrate: %w", err)
	}
	// Additive column migrations — safe to re-run; errors are ignored if columns already exist.
	for _, col := range []struct{ name, def string }{
		{"model_id", "TEXT NOT NULL DEFAULT ''"},
		{"provider_hint", "TEXT NOT NULL DEFAULT ''"},
		{"command", "TEXT NOT NULL DEFAULT ''"},
		{"provider_name", "TEXT NOT NULL DEFAULT ''"},
		{"priority", "INTEGER NOT NULL DEFAULT 2"},
		{"tags", "TEXT NOT NULL DEFAULT '[]'"},
		{"retry_count", "INTEGER NOT NULL DEFAULT 0"},
	} {
		_, _ = db.Exec(fmt.Sprintf("ALTER TABLE tasks ADD COLUMN %s %s", col.name, col.def))
	}
	return nil
}

// Save inserts a new Task record.
func (r *Repository) Save(t domain.Task) error {
	t.ProjectPath = filepath.Clean(t.ProjectPath)
	ctxJSON, err := json.Marshal(t.ContextFiles)
	if err != nil {
		return fmt.Errorf("sqlite: marshal context files: %w", err)
	}
	tagsJSON := []byte("[]")
	if len(t.Tags) > 0 {
		if b, mErr := json.Marshal(t.Tags); mErr == nil {
			tagsJSON = b
		}
	}
	_, err = r.db.Exec(
		`INSERT INTO tasks (id, project_path, target_file, instruction, context_files, status, created_at, updated_at, logs, model_id, provider_hint, command, provider_name, priority, tags, retry_count)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.ProjectPath, t.TargetFile, t.Instruction,
		string(ctxJSON), string(t.Status),
		t.CreatedAt.UnixMilli(), t.UpdatedAt.UnixMilli(),
		t.Logs, t.ModelID, t.ProviderHint, string(t.Command),
		t.ProviderName, t.Priority, string(tagsJSON), t.RetryCount,
	)
	if err != nil {
		return fmt.Errorf("sqlite: insert task: %w", err)
	}
	return nil
}

// GetByID retrieves a single task by its ID.
// Returns domain.ErrNotFound when no row matches.
func (r *Repository) GetByID(id string) (domain.Task, error) {
	row := r.db.QueryRow(`SELECT id, project_path, target_file, instruction, context_files, status, created_at, updated_at, logs, model_id, provider_hint, command, provider_name, priority, tags, retry_count FROM tasks WHERE id = ?`, id)
	t, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Task{}, fmt.Errorf("sqlite: get task: %w", domain.ErrNotFound)
	}
	return t, err
}

// GetPending returns all tasks in QUEUED or PROCESSING state.
func (r *Repository) GetPending() ([]domain.Task, error) {
	rows, err := r.db.Query(
		`SELECT id, project_path, target_file, instruction, context_files, status, created_at, updated_at, logs, model_id, provider_hint, command, provider_name, priority, tags, retry_count
		 FROM tasks WHERE status IN ('QUEUED','PROCESSING') ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("sqlite: query pending: %w", err)
	}
	defer rows.Close()

	var tasks []domain.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// UpdateStatus changes the status of a task identified by id.
func (r *Repository) UpdateStatus(id string, status domain.TaskStatus) error {
	res, err := r.db.Exec(
		`UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?`,
		string(status), time.Now().UnixMilli(), id,
	)
	if err != nil {
		return fmt.Errorf("sqlite: update status: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("sqlite: rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("sqlite: update status: %w", domain.ErrNotFound)
	}
	return nil
}

// UpdateLogs replaces the logs field for the task identified by id.
func (r *Repository) UpdateLogs(id, logs string) error {
	res, err := r.db.Exec(
		`UPDATE tasks SET logs = ?, updated_at = ? WHERE id = ?`,
		logs, time.Now().UnixMilli(), id,
	)
	if err != nil {
		return fmt.Errorf("sqlite: update logs: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("sqlite: rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("sqlite: update logs: %w", domain.ErrNotFound)
	}
	return nil
}

// GetByProjectPath returns all tasks for the given project path, ordered by creation time descending.
func (r *Repository) GetByProjectPath(projectPath string) ([]domain.Task, error) {
	projectPath = filepath.Clean(projectPath)
	rows, err := r.db.Query(
		`SELECT id, project_path, target_file, instruction, context_files, status, created_at, updated_at, logs, model_id, provider_hint, command, provider_name, priority, tags, retry_count
		 FROM tasks WHERE project_path = ? ORDER BY created_at DESC`,
		projectPath,
	)
	if err != nil {
		return nil, fmt.Errorf("sqlite: query by project path: %w", err)
	}
	defer rows.Close()

	var tasks []domain.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// GetByProjectPathAndStatus returns tasks for a project filtered by one or more statuses,
// ordered by priority ASC then created_at ASC. Returns an empty slice (not an error) if none match.
func (r *Repository) GetByProjectPathAndStatus(projectPath string, statuses ...domain.TaskStatus) ([]domain.Task, error) {
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
	query := fmt.Sprintf(
		`SELECT id, project_path, target_file, instruction, context_files, status, created_at, updated_at, logs, model_id, provider_hint, command, provider_name, priority, tags, retry_count
		 FROM tasks WHERE project_path = ? AND status IN (%s) ORDER BY priority ASC, created_at ASC`,
		strings.Join(placeholders, ","),
	)
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite: query by project path and status: %w", err)
	}
	defer rows.Close()
	tasks := []domain.Task{}
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// Update persists changes to an existing task's mutable fields.
// Returns domain.ErrNotFound if no row with t.ID exists.
func (r *Repository) Update(t domain.Task) error {
	t.UpdatedAt = time.Now()
	tagsJSON := []byte("[]")
	if len(t.Tags) > 0 {
		if b, mErr := json.Marshal(t.Tags); mErr == nil {
			tagsJSON = b
		}
	}
	res, err := r.db.Exec(
		`UPDATE tasks SET instruction=?, target_file=?, provider_name=?, priority=?, tags=?, status=?, retry_count=?, updated_at=? WHERE id=?`,
		t.Instruction, t.TargetFile, t.ProviderName, t.Priority, string(tagsJSON), string(t.Status), t.RetryCount, t.UpdatedAt.UnixMilli(), t.ID,
	)
	if err != nil {
		return fmt.Errorf("sqlite: update task: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("sqlite: rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("sqlite: update task: %w", domain.ErrNotFound)
	}
	return nil
}

// Close releases the underlying database connection.
func (r *Repository) Close() error { return r.db.Close() }

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...interface{}) error
}

func scanTask(s scanner) (domain.Task, error) {
	var t domain.Task
	var status string
	var command string
	var ctxJSON string
	var tagsJSON string
	var createdMS, updatedMS int64

	if err := s.Scan(
		&t.ID, &t.ProjectPath, &t.TargetFile, &t.Instruction,
		&ctxJSON, &status, &createdMS, &updatedMS, &t.Logs,
		&t.ModelID, &t.ProviderHint, &command, &t.ProviderName, &t.Priority, &tagsJSON, &t.RetryCount,
	); err != nil {
		return t, fmt.Errorf("sqlite: scan task: %w", err)
	}

	t.Status = domain.TaskStatus(status)
	t.Command = domain.CommandType(command)
	t.CreatedAt = time.UnixMilli(createdMS)
	t.UpdatedAt = time.UnixMilli(updatedMS)

	if err := json.Unmarshal([]byte(ctxJSON), &t.ContextFiles); err != nil {
		return t, fmt.Errorf("sqlite: unmarshal context files: %w", err)
	}
	if tagsJSON != "" && tagsJSON != "[]" {
		_ = json.Unmarshal([]byte(tagsJSON), &t.Tags) // treat parse errors as empty slice
	}
	return t, nil
}
