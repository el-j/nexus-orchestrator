package repo_sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// compile-time interface check
var _ ports.ModelCapabilityRepository = (*ModelCapabilityRepo)(nil)

// ModelCapabilityRepo implements ports.ModelCapabilityRepository using the shared SQLite database.
type ModelCapabilityRepo struct {
	db *sql.DB
}

// NewModelCapabilityRepo returns a ModelCapabilityRepo backed by the same database as r.
func NewModelCapabilityRepo(r *Repository) *ModelCapabilityRepo {
	return &ModelCapabilityRepo{db: r.db}
}

// Save inserts or updates a model capability profile (upsert by model_id).
func (m *ModelCapabilityRepo) Save(p domain.ModelCapabilityProfile) error {
	now := time.Now()
	createdAt := p.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}

	builtIn := 0
	if p.BuiltIn {
		builtIn = 1
	}

	_, err := m.db.Exec(
		`INSERT INTO model_capabilities (model_id, context_window, recommended_max_output, notes, built_in, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(model_id) DO UPDATE SET
		   context_window       = excluded.context_window,
		   recommended_max_output = excluded.recommended_max_output,
		   notes                = excluded.notes,
		   built_in             = excluded.built_in,
		   updated_at           = excluded.updated_at`,
		p.ModelID, p.ContextWindow, p.RecommendedMaxOutput, p.Notes,
		builtIn, createdAt, now,
	)
	if err != nil {
		return fmt.Errorf("sqlite: save model capability: %w", err)
	}
	return nil
}

// GetByModelID returns the model capability profile with the given model ID, or domain.ErrNotFound.
func (m *ModelCapabilityRepo) GetByModelID(modelID string) (domain.ModelCapabilityProfile, error) {
	row := m.db.QueryRow(
		`SELECT model_id, context_window, recommended_max_output, notes, built_in, created_at, updated_at
		 FROM model_capabilities WHERE model_id = ?`, modelID,
	)
	p, err := scanModelCapability(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ModelCapabilityProfile{}, fmt.Errorf("sqlite: get model capability: %w", domain.ErrNotFound)
		}
		return domain.ModelCapabilityProfile{}, fmt.Errorf("sqlite: get model capability: %w", err)
	}
	return p, nil
}

// GetAll returns all model capability profiles ordered by model_id.
func (m *ModelCapabilityRepo) GetAll() ([]domain.ModelCapabilityProfile, error) {
	rows, err := m.db.Query(
		`SELECT model_id, context_window, recommended_max_output, notes, built_in, created_at, updated_at
		 FROM model_capabilities ORDER BY model_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list model capabilities: %w", err)
	}
	defer rows.Close()

	profiles := []domain.ModelCapabilityProfile{}
	for rows.Next() {
		p, err := scanModelCapability(rows)
		if err != nil {
			return nil, fmt.Errorf("sqlite: list model capabilities: %w", err)
		}
		profiles = append(profiles, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: list model capabilities: %w", err)
	}
	return profiles, nil
}

// Delete removes the model capability profile with the given model ID.
// Returns domain.ErrNotFound when no row with that ID exists.
func (m *ModelCapabilityRepo) Delete(modelID string) error {
	res, err := m.db.Exec(`DELETE FROM model_capabilities WHERE model_id = ?`, modelID)
	if err != nil {
		return fmt.Errorf("sqlite: delete model capability: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("sqlite: delete model capability rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("sqlite: delete model capability: %w", domain.ErrNotFound)
	}
	return nil
}

func scanModelCapability(s scanner) (domain.ModelCapabilityProfile, error) {
	var p domain.ModelCapabilityProfile
	var builtInInt int
	var createdAtStr, updatedAtStr string

	if err := s.Scan(
		&p.ModelID, &p.ContextWindow, &p.RecommendedMaxOutput, &p.Notes,
		&builtInInt, &createdAtStr, &updatedAtStr,
	); err != nil {
		return domain.ModelCapabilityProfile{}, fmt.Errorf("sqlite: scan model capability: %w", err)
	}

	p.BuiltIn = builtInInt != 0

	// Parse DATETIME strings returned from SQLite
	if createdAtStr != "" {
		if parsed, err := time.Parse("2006-01-02 15:04:05", createdAtStr); err == nil {
			p.CreatedAt = parsed.UTC()
		}
	}
	if updatedAtStr != "" {
		if parsed, err := time.Parse("2006-01-02 15:04:05", updatedAtStr); err == nil {
			p.UpdatedAt = parsed.UTC()
		}
	}

	return p, nil
}
