package repo_sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// Compile-time interface check.
var _ ports.AIActivityRepository = (*ActivityRepo)(nil)

// ActivityRepo implements ports.AIActivityRepository using the shared SQLite database.
type ActivityRepo struct {
	db *sql.DB
}

// NewActivityRepo returns an ActivityRepo backed by the same database as r.
func NewActivityRepo(r *Repository) *ActivityRepo {
	return &ActivityRepo{db: r.db}
}

// SaveActivity inserts an AIActivity record, ignoring duplicates by ID.
func (a *ActivityRepo) SaveActivity(ctx context.Context, act domain.AIActivity) error {
	metaJSON, err := json.Marshal(act.Metadata)
	if err != nil {
		return fmt.Errorf("repo_sqlite: marshal activity metadata: %w", err)
	}
	_, err = a.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO ai_activities
		 (id, session_id, agent_name, activity_type, summary, project_path, model, tokens_in, tokens_out, timestamp, metadata)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		act.ID, act.SessionID, act.AgentName, string(act.ActivityType),
		act.Summary, act.ProjectPath, act.Model,
		act.TokensIn, act.TokensOut,
		act.Timestamp.UTC(),
		string(metaJSON),
	)
	if err != nil {
		return fmt.Errorf("repo_sqlite: save activity: %w", err)
	}
	return nil
}

// ListActivities returns activities matching the filter, ordered by timestamp DESC.
func (a *ActivityRepo) ListActivities(ctx context.Context, f domain.ActivityFilter) ([]domain.AIActivity, error) {
	limit := f.Limit
	if limit == 0 {
		limit = 100
	}

	var clauses []string
	var args []any

	if f.AgentName != "" {
		clauses = append(clauses, "agent_name = ?")
		args = append(args, f.AgentName)
	}
	if f.ProjectPath != "" {
		clauses = append(clauses, "project_path = ?")
		args = append(args, f.ProjectPath)
	}
	if f.Type != "" {
		clauses = append(clauses, "activity_type = ?")
		args = append(args, string(f.Type))
	}
	if !f.Since.IsZero() {
		clauses = append(clauses, "timestamp >= ?")
		args = append(args, f.Since.UTC())
	}

	query := `SELECT id, session_id, agent_name, activity_type, summary, project_path, model, tokens_in, tokens_out, timestamp, metadata
	          FROM ai_activities`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY timestamp DESC LIMIT ?"
	args = append(args, limit)

	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("repo_sqlite: list activities: %w", err)
	}
	defer rows.Close()

	var activities []domain.AIActivity
	for rows.Next() {
		var act domain.AIActivity
		var actType string
		var metaJSON string
		var ts time.Time

		if err := rows.Scan(
			&act.ID, &act.SessionID, &act.AgentName, &actType,
			&act.Summary, &act.ProjectPath, &act.Model,
			&act.TokensIn, &act.TokensOut, &ts, &metaJSON,
		); err != nil {
			return nil, fmt.Errorf("repo_sqlite: scan activity: %w", err)
		}
		act.ActivityType = domain.ActivityType(actType)
		act.Timestamp = ts

		if err := json.Unmarshal([]byte(metaJSON), &act.Metadata); err != nil {
			act.Metadata = nil
		}
		activities = append(activities, act)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repo_sqlite: iterate activities: %w", err)
	}
	return activities, nil
}

// PurgeOlderThan deletes activities with a timestamp before cutoff and returns the row count.
func (a *ActivityRepo) PurgeOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := a.db.ExecContext(ctx,
		`DELETE FROM ai_activities WHERE timestamp < ?`, cutoff.UTC(),
	)
	if err != nil {
		return 0, fmt.Errorf("repo_sqlite: purge activities: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("repo_sqlite: rows affected after purge: %w", err)
	}
	return n, nil
}
