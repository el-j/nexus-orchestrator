package repo_sqlite

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"nexus-ai/internal/core/domain"
)

// SessionRepo implements ports.SessionRepository using the shared SQLite database.
// It is a separate struct from Repository to avoid method-name conflicts on Save.
type SessionRepo struct {
	db *sql.DB
}

// NewSessionRepo returns a SessionRepo backed by the same database as r.
func NewSessionRepo(r *Repository) *SessionRepo {
	return &SessionRepo{db: r.db}
}

// Save persists a Session (insert or replace by project_path).
func (s *SessionRepo) Save(sess domain.Session) error {
	msgsJSON, err := json.Marshal(sess.Messages)
	if err != nil {
		return fmt.Errorf("sqlite: marshal session messages: %w", err)
	}
	_, err = s.db.Exec(
		`INSERT INTO sessions (id, project_path, messages, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(project_path) DO UPDATE SET
		   messages   = excluded.messages,
		   updated_at = excluded.updated_at`,
		sess.ID, sess.ProjectPath, string(msgsJSON),
		sess.CreatedAt.UnixMilli(), sess.UpdatedAt.UnixMilli(),
	)
	if err != nil {
		return fmt.Errorf("sqlite: save session: %w", err)
	}
	return nil
}

// GetByProjectPath returns the session for the normalized project path, or
// domain.ErrNotFound when no session exists yet.
func (s *SessionRepo) GetByProjectPath(projectPath string) (domain.Session, error) {
	clean := filepath.Clean(projectPath)
	row := s.db.QueryRow(
		`SELECT id, project_path, messages, created_at, updated_at FROM sessions WHERE project_path = ?`,
		clean,
	)
	return scanSession(row)
}

// AppendMessage adds a single message to the session identified by projectPath.
// If no session exists it is created automatically.
func (s *SessionRepo) AppendMessage(projectPath string, msg domain.Message) error {
	clean := filepath.Clean(projectPath)

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("sqlite: begin tx for append message: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	row := tx.QueryRow(
		`SELECT id, project_path, messages, created_at, updated_at FROM sessions WHERE project_path = ?`,
		clean,
	)
	sess, err := scanSession(row)
	if errors.Is(err, domain.ErrNotFound) {
		now := time.Now()
		sess = domain.Session{
			ID:          uuid.NewString(),
			ProjectPath: clean,
			Messages:    nil,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	} else if err != nil {
		return err
	}

	sess.Messages = append(sess.Messages, msg)
	sess.UpdatedAt = time.Now()

	msgsJSON, err := json.Marshal(sess.Messages)
	if err != nil {
		return fmt.Errorf("sqlite: marshal messages: %w", err)
	}
	_, err = tx.Exec(
		`INSERT INTO sessions (id, project_path, messages, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(project_path) DO UPDATE SET
		   messages   = excluded.messages,
		   updated_at = excluded.updated_at`,
		sess.ID, clean, string(msgsJSON),
		sess.CreatedAt.UnixMilli(), sess.UpdatedAt.UnixMilli(),
	)
	if err != nil {
		return fmt.Errorf("sqlite: upsert session: %w", err)
	}
	return tx.Commit()
}

func scanSession(s scanner) (domain.Session, error) {
	var sess domain.Session
	var msgsJSON string
	var createdMS, updatedMS int64

	if err := s.Scan(&sess.ID, &sess.ProjectPath, &msgsJSON, &createdMS, &updatedMS); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Session{}, fmt.Errorf("sqlite: get session: %w", domain.ErrNotFound)
		}
		return domain.Session{}, fmt.Errorf("sqlite: scan session: %w", err)
	}

	sess.CreatedAt = time.UnixMilli(createdMS)
	sess.UpdatedAt = time.UnixMilli(updatedMS)

	if err := json.Unmarshal([]byte(msgsJSON), &sess.Messages); err != nil {
		return domain.Session{}, fmt.Errorf("sqlite: unmarshal messages: %w", err)
	}
	return sess, nil
}
