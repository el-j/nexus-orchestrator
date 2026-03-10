package repo_sqlite

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"nexus-orchestrator/internal/core/domain"

	_ "github.com/mattn/go-sqlite3"
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
		return nil, fmt.Errorf("sqlite: migrate: %w", err)
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
	`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
		CREATE INDEX IF NOT EXISTS idx_tasks_project_path ON tasks(project_path);
	`)
	if err != nil {
		return err
	}
	// Additive column migrations — safe to re-run; errors are ignored if columns already exist.
	for _, col := range []struct{ name, def string }{
		{"model_id", "TEXT NOT NULL DEFAULT ''"},
		{"provider_hint", "TEXT NOT NULL DEFAULT ''"},
		{"command", "TEXT NOT NULL DEFAULT ''"},
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
	_, err = r.db.Exec(
		`INSERT INTO tasks (id, project_path, target_file, instruction, context_files, status, created_at, updated_at, logs, model_id, provider_hint, command)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.ProjectPath, t.TargetFile, t.Instruction,
		string(ctxJSON), string(t.Status),
		t.CreatedAt.UnixMilli(), t.UpdatedAt.UnixMilli(),
		t.Logs, t.ModelID, t.ProviderHint, string(t.Command),
	)
	if err != nil {
		return fmt.Errorf("sqlite: insert task: %w", err)
	}
	return nil
}

// GetByID retrieves a single task by its ID.
// Returns domain.ErrNotFound when no row matches.
func (r *Repository) GetByID(id string) (domain.Task, error) {
	row := r.db.QueryRow(`SELECT id, project_path, target_file, instruction, context_files, status, created_at, updated_at, logs, model_id, provider_hint, command FROM tasks WHERE id = ?`, id)
	t, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Task{}, fmt.Errorf("sqlite: get task: %w", domain.ErrNotFound)
	}
	return t, err
}

// GetPending returns all tasks in QUEUED or PROCESSING state.
func (r *Repository) GetPending() ([]domain.Task, error) {
	rows, err := r.db.Query(
		`SELECT id, project_path, target_file, instruction, context_files, status, created_at, updated_at, logs, model_id, provider_hint, command
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
	n, _ := res.RowsAffected()
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
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("sqlite: update logs: %w", domain.ErrNotFound)
	}
	return nil
}

// GetByProjectPath returns all tasks for the given project path, ordered by creation time descending.
func (r *Repository) GetByProjectPath(projectPath string) ([]domain.Task, error) {
	projectPath = filepath.Clean(projectPath)
	rows, err := r.db.Query(
		`SELECT id, project_path, target_file, instruction, context_files, status, created_at, updated_at, logs, model_id, provider_hint, command
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
	var createdMS, updatedMS int64

	if err := s.Scan(
		&t.ID, &t.ProjectPath, &t.TargetFile, &t.Instruction,
		&ctxJSON, &status, &createdMS, &updatedMS, &t.Logs,
		&t.ModelID, &t.ProviderHint, &command,
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
	return t, nil
}
