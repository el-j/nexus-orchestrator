---
id: TASK-347
title: SQLite model_capabilities table + ModelCapabilityRepository adapter
role: backend
planId: PLAN-051
status: done
dependencies: [TASK-346]
createdAt: 2026-03-28T00:00:00Z
---

## Context

TASK-346 defines the `ModelCapabilityProfile` domain type and port interface. This task wires in the SQLite adapter and migration so that user-defined (non-built-in) profiles are persisted across restarts.

## Files to Read

- `internal/adapters/outbound/repo_sqlite/repo.go` — migration pattern, DB setup
- `internal/adapters/outbound/repo_sqlite/provider_config_repo.go` — CRUD pattern to copy
- `internal/core/ports/ports.go` — `ModelCapabilityRepository` interface (added in TASK-346)

## Implementation Steps

1. Add migration to `repo_sqlite/repo.go` `migrations` slice:

```go
`CREATE TABLE IF NOT EXISTS model_capabilities (
    model_id TEXT PRIMARY KEY,
    context_window INTEGER NOT NULL DEFAULT 0,
    recommended_max_output INTEGER NOT NULL DEFAULT 0,
    notes TEXT NOT NULL DEFAULT '',
    built_in INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
)`,
```

2. Create `internal/adapters/outbound/repo_sqlite/model_capability_repo.go`:

```go
package repo_sqlite

import (
    "database/sql"
    "errors"
    "fmt"
    "time"

    "nexus-orchestrator/internal/core/domain"
)

// ModelCapabilityRepo implements ports.ModelCapabilityRepository using SQLite.
type ModelCapabilityRepo struct{ db *sql.DB }

func NewModelCapabilityRepo(db *sql.DB) *ModelCapabilityRepo {
    return &ModelCapabilityRepo{db: db}
}

func (r *ModelCapabilityRepo) Save(p domain.ModelCapabilityProfile) error {
    now := time.Now().UTC()
    if p.CreatedAt.IsZero() {
        p.CreatedAt = now
    }
    p.UpdatedAt = now
    _, err := r.db.Exec(`
        INSERT INTO model_capabilities
            (model_id, context_window, recommended_max_output, notes, built_in, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(model_id) DO UPDATE SET
            context_window=excluded.context_window,
            recommended_max_output=excluded.recommended_max_output,
            notes=excluded.notes,
            updated_at=excluded.updated_at`,
        p.ModelID, p.ContextWindow, p.RecommendedMaxOutput, p.Notes,
        boolToInt(p.BuiltIn), p.CreatedAt, p.UpdatedAt,
    )
    if err != nil {
        return fmt.Errorf("model_capability_repo: save %q: %w", p.ModelID, err)
    }
    return nil
}

func (r *ModelCapabilityRepo) GetByModelID(modelID string) (domain.ModelCapabilityProfile, error) {
    row := r.db.QueryRow(`
        SELECT model_id, context_window, recommended_max_output, notes, built_in, created_at, updated_at
        FROM model_capabilities WHERE model_id = ?`, modelID)
    return scanModelCapability(row)
}

func (r *ModelCapabilityRepo) GetAll() ([]domain.ModelCapabilityProfile, error) {
    rows, err := r.db.Query(`
        SELECT model_id, context_window, recommended_max_output, notes, built_in, created_at, updated_at
        FROM model_capabilities ORDER BY model_id`)
    if err != nil {
        return nil, fmt.Errorf("model_capability_repo: get all: %w", err)
    }
    defer rows.Close()
    var out []domain.ModelCapabilityProfile
    for rows.Next() {
        if p, err := scanModelCapability(rows); err == nil {
            out = append(out, p)
        }
    }
    return out, rows.Err()
}

func (r *ModelCapabilityRepo) Delete(modelID string) error {
    _, err := r.db.Exec(`DELETE FROM model_capabilities WHERE model_id = ?`, modelID)
    return err
}

type scannable interface {
    Scan(dest ...any) error
}

func scanModelCapability(s scannable) (domain.ModelCapabilityProfile, error) {
    var p domain.ModelCapabilityProfile
    var builtIn int
    err := s.Scan(&p.ModelID, &p.ContextWindow, &p.RecommendedMaxOutput, &p.Notes,
        &builtIn, &p.CreatedAt, &p.UpdatedAt)
    if errors.Is(err, sql.ErrNoRows) {
        return p, fmt.Errorf("model_capability_repo: not found")
    }
    p.BuiltIn = builtIn != 0
    return p, err
}
```

Note: `boolToInt` already exists in the package — do not redeclare it.

3. Wire into `repo_sqlite/repo.go` `Repository` struct if a central struct is used, or expose `NewModelCapabilityRepo` standalone (same pattern as other repos).

## Acceptance Criteria

- `go build ./internal/adapters/outbound/repo_sqlite/...` passes
- `go vet ./...` clean
- `NewModelCapabilityRepo` satisfies `ports.ModelCapabilityRepository` (confirmed by adding a compile-time assertion)
