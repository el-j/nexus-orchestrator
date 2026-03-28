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

var _ ports.RuntimeConfigRepository = (*RuntimeConfigRepo)(nil)

// RuntimeConfigRepo persists runtime config in the shared SQLite database.
// Storage is a single-row JSON blob (id=1) to keep migrations additive and simple.
type RuntimeConfigRepo struct {
	db *sql.DB
}

func NewRuntimeConfigRepo(r *Repository) *RuntimeConfigRepo {
	return &RuntimeConfigRepo{db: r.db}
}

func (r *RuntimeConfigRepo) GetRuntimeConfig(ctx context.Context) (domain.RuntimeConfig, error) {
	row := r.db.QueryRowContext(ctx, `SELECT json, updated_at FROM runtime_config WHERE id = 1`)
	var raw string
	var updatedAtMS int64
	if err := row.Scan(&raw, &updatedAtMS); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.RuntimeConfig{}, fmt.Errorf("sqlite: get runtime config: %w", domain.ErrNotFound)
		}
		return domain.RuntimeConfig{}, fmt.Errorf("sqlite: get runtime config: %w", err)
	}
	var cfg domain.RuntimeConfig
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
			return domain.RuntimeConfig{}, fmt.Errorf("sqlite: unmarshal runtime config: %w", err)
		}
	}
	if updatedAtMS > 0 {
		cfg.UpdatedAt = time.UnixMilli(updatedAtMS)
	}
	return cfg, nil
}

func (r *RuntimeConfigRepo) SaveRuntimeConfig(ctx context.Context, cfg domain.RuntimeConfig) error {
	// Persist tokens and queue cap; UpdatedAt is stored separately as milliseconds.
	b, err := json.Marshal(struct {
		QueueCap int    `json:"queueCap"`
		APIToken string `json:"apiToken,omitempty"`
		MCPToken string `json:"mcpToken,omitempty"`
	}{
		QueueCap: cfg.QueueCap,
		APIToken: cfg.APIToken,
		MCPToken: cfg.MCPToken,
	})
	if err != nil {
		return fmt.Errorf("sqlite: marshal runtime config: %w", err)
	}
	updatedAt := cfg.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO runtime_config (id, json, updated_at)
		 VALUES (1, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   json = excluded.json,
		   updated_at = excluded.updated_at`,
		string(b), updatedAt.UnixMilli(),
	)
	if err != nil {
		return fmt.Errorf("sqlite: save runtime config: %w", err)
	}
	return nil
}
