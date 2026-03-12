package repo_sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// compile-time interface check
var _ ports.AISessionRepository = (*AISessionRepo)(nil)

// AISessionRepo implements ports.AISessionRepository using the shared SQLite database.
type AISessionRepo struct {
	db *sql.DB
}

// NewAISessionRepo returns an AISessionRepo backed by the same database as r.
func NewAISessionRepo(r *Repository) *AISessionRepo {
	return &AISessionRepo{db: r.db}
}

// SaveAISession inserts or replaces an AI session record.
func (a *AISessionRepo) SaveAISession(ctx context.Context, s domain.AISession) error {
	idsJSON, err := json.Marshal(s.RoutedTaskIDs)
	if err != nil {
		return fmt.Errorf("repo_sqlite: save ai session: marshal routed task ids: %w", err)
	}

	_, err = a.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO ai_sessions
		 (id, source, external_id, agent_name, project_path, status, last_activity, routed_task_ids, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, string(s.Source), s.ExternalID, s.AgentName, s.ProjectPath,
		string(s.Status),
		s.LastActivity.UTC().Format(time.RFC3339),
		string(idsJSON),
		s.CreatedAt.UTC().Format(time.RFC3339),
		s.UpdatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("repo_sqlite: save ai session: %w", err)
	}
	return nil
}

// GetAISessionByID returns the AI session with the given ID, or domain.ErrNotFound.
func (a *AISessionRepo) GetAISessionByID(ctx context.Context, id string) (domain.AISession, error) {
	row := a.db.QueryRowContext(ctx,
		`SELECT id, source, external_id, agent_name, project_path, status, last_activity, routed_task_ids, created_at, updated_at
		 FROM ai_sessions WHERE id = ?`, id,
	)
	s, err := scanAISession(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.AISession{}, fmt.Errorf("repo_sqlite: get ai session: %w", domain.ErrNotFound)
		}
		return domain.AISession{}, fmt.Errorf("repo_sqlite: get ai session: %w", err)
	}
	return s, nil
}

// ListAISessions returns all AI session records ordered by last activity descending.
func (a *AISessionRepo) ListAISessions(ctx context.Context) ([]domain.AISession, error) {
	rows, err := a.db.QueryContext(ctx,
		`SELECT id, source, external_id, agent_name, project_path, status, last_activity, routed_task_ids, created_at, updated_at
		 FROM ai_sessions ORDER BY last_activity DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("repo_sqlite: list ai sessions: %w", err)
	}
	defer rows.Close()

	var sessions []domain.AISession
	for rows.Next() {
		s, err := scanAISession(rows)
		if err != nil {
			return nil, fmt.Errorf("repo_sqlite: list ai sessions: %w", err)
		}
		sessions = append(sessions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repo_sqlite: list ai sessions: %w", err)
	}
	return sessions, nil
}

// UpdateAISessionStatus updates the status and last_activity of an AI session.
// Returns domain.ErrNotFound when no session with the given id exists.
func (a *AISessionRepo) UpdateAISessionStatus(ctx context.Context, id string, status domain.AISessionStatus, lastActivity time.Time) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := a.db.ExecContext(ctx,
		`UPDATE ai_sessions SET status = ?, last_activity = ?, updated_at = ? WHERE id = ?`,
		string(status), lastActivity.UTC().Format(time.RFC3339), now, id,
	)
	if err != nil {
		return fmt.Errorf("repo_sqlite: update ai session status: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("repo_sqlite: update ai session status: rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("repo_sqlite: update ai session status: %w", domain.ErrNotFound)
	}
	return nil
}

// DeleteAISession removes the AI session with the given ID. Idempotent.
func (a *AISessionRepo) DeleteAISession(ctx context.Context, id string) error {
	_, err := a.db.ExecContext(ctx, `DELETE FROM ai_sessions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("repo_sqlite: delete ai session: %w", err)
	}
	return nil
}

func scanAISession(s scanner) (domain.AISession, error) {
	var sess domain.AISession
	var sourceStr, statusStr string
	var lastActivityStr, createdAtStr, updatedAtStr string
	var routedTaskIDsJSON string

	if err := s.Scan(
		&sess.ID, &sourceStr, &sess.ExternalID, &sess.AgentName, &sess.ProjectPath,
		&statusStr, &lastActivityStr, &routedTaskIDsJSON,
		&createdAtStr, &updatedAtStr,
	); err != nil {
		return domain.AISession{}, err
	}

	sess.Source = domain.AISessionSource(sourceStr)
	sess.Status = domain.AISessionStatus(statusStr)

	var parseErr error
	sess.LastActivity, parseErr = time.Parse(time.RFC3339, lastActivityStr)
	if parseErr != nil {
		return domain.AISession{}, fmt.Errorf("scan ai session: parse last_activity: %w", parseErr)
	}
	sess.CreatedAt, parseErr = time.Parse(time.RFC3339, createdAtStr)
	if parseErr != nil {
		return domain.AISession{}, fmt.Errorf("scan ai session: parse created_at: %w", parseErr)
	}
	sess.UpdatedAt, parseErr = time.Parse(time.RFC3339, updatedAtStr)
	if parseErr != nil {
		return domain.AISession{}, fmt.Errorf("scan ai session: parse updated_at: %w", parseErr)
	}

	if err := json.Unmarshal([]byte(routedTaskIDsJSON), &sess.RoutedTaskIDs); err != nil {
		return domain.AISession{}, fmt.Errorf("scan ai session: unmarshal routed_task_ids: %w", err)
	}

	return sess, nil
}
