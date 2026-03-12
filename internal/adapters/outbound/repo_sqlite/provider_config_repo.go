package repo_sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"

	"github.com/google/uuid"
)

// compile-time interface check
var _ ports.ProviderConfigRepository = (*ProviderConfigRepo)(nil)

// ProviderConfigRepo implements ports.ProviderConfigRepository using the shared SQLite database.
type ProviderConfigRepo struct {
	db *sql.DB
}

// NewProviderConfigRepo returns a ProviderConfigRepo backed by the same database as r.
func NewProviderConfigRepo(r *Repository) *ProviderConfigRepo {
	return &ProviderConfigRepo{db: r.db}
}

// SaveProviderConfig inserts or updates a provider config record (upsert by ID).
// If cfg.ID is empty a new UUID is assigned.
func (p *ProviderConfigRepo) SaveProviderConfig(ctx context.Context, cfg domain.ProviderConfig) error {
	if cfg.ID == "" {
		cfg.ID = uuid.NewString()
	}
	now := time.Now()
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = now
	}
	cfg.UpdatedAt = now

	enabled := 0
	if cfg.Enabled {
		enabled = 1
	}

	_, err := p.db.ExecContext(ctx,
		`INSERT INTO provider_configs (id, name, kind, base_url, api_key, model, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   name       = excluded.name,
		   kind       = excluded.kind,
		   base_url   = excluded.base_url,
		   api_key    = excluded.api_key,
		   model      = excluded.model,
		   enabled    = excluded.enabled,
		   updated_at = excluded.updated_at`,
		cfg.ID, cfg.Name, string(cfg.Kind), cfg.BaseURL, cfg.APIKey, cfg.Model,
		enabled, cfg.CreatedAt.UnixMilli(), cfg.UpdatedAt.UnixMilli(),
	)
	if err != nil {
		return fmt.Errorf("sqlite: save provider config: %w", err)
	}
	return nil
}

// ListProviderConfigs returns all provider config records ordered by creation time.
func (p *ProviderConfigRepo) ListProviderConfigs(ctx context.Context) ([]domain.ProviderConfig, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, name, kind, base_url, api_key, model, enabled, created_at, updated_at
		 FROM provider_configs ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list provider configs: %w", err)
	}
	defer rows.Close()

	var configs []domain.ProviderConfig
	for rows.Next() {
		cfg, err := scanProviderConfig(rows)
		if err != nil {
			return nil, fmt.Errorf("sqlite: list provider configs: %w", err)
		}
		configs = append(configs, cfg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: list provider configs: %w", err)
	}
	return configs, nil
}

// GetProviderConfig returns the provider config with the given ID, or domain.ErrNotFound.
func (p *ProviderConfigRepo) GetProviderConfig(ctx context.Context, id string) (domain.ProviderConfig, error) {
	row := p.db.QueryRowContext(ctx,
		`SELECT id, name, kind, base_url, api_key, model, enabled, created_at, updated_at
		 FROM provider_configs WHERE id = ?`, id,
	)
	cfg, err := scanProviderConfig(row)
	if err != nil {
		return domain.ProviderConfig{}, fmt.Errorf("sqlite: get provider config: %w", err)
	}
	return cfg, nil
}

// DeleteProviderConfig removes the provider config with the given ID.
// Returns domain.ErrNotFound when no row with that ID exists.
func (p *ProviderConfigRepo) DeleteProviderConfig(ctx context.Context, id string) error {
	res, err := p.db.ExecContext(ctx, `DELETE FROM provider_configs WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("sqlite: delete provider config: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("sqlite: delete provider config rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("sqlite: delete provider config: %w", domain.ErrNotFound)
	}
	return nil
}

func scanProviderConfig(s scanner) (domain.ProviderConfig, error) {
	var cfg domain.ProviderConfig
	var kindStr string
	var enabledInt int
	var createdAtMs, updatedAtMs int64

	if err := s.Scan(
		&cfg.ID, &cfg.Name, &kindStr, &cfg.BaseURL, &cfg.APIKey, &cfg.Model,
		&enabledInt, &createdAtMs, &updatedAtMs,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ProviderConfig{}, fmt.Errorf("sqlite: get provider config: %w", domain.ErrNotFound)
		}
		return domain.ProviderConfig{}, fmt.Errorf("sqlite: scan provider config: %w", err)
	}

	cfg.Kind = domain.ProviderKind(kindStr)
	cfg.Enabled = enabledInt != 0
	cfg.CreatedAt = time.UnixMilli(createdAtMs).UTC()
	cfg.UpdatedAt = time.UnixMilli(updatedAtMs).UTC()
	return cfg, nil
}
