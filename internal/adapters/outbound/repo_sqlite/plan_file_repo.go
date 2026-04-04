package repo_sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// compile-time interface check
var _ ports.DiscoveredPlanFileRepo = (*PlanFileRepo)(nil)

// PlanFileRepo persists domain.DiscoveredPlanFile records using the shared SQLite database.
type PlanFileRepo struct {
	db *sql.DB
}

// NewPlanFileRepo returns a PlanFileRepo backed by the same database as r.
func NewPlanFileRepo(r *Repository) *PlanFileRepo {
	return &PlanFileRepo{db: r.db}
}

// UpsertPlanFile inserts or replaces a discovered plan file record.
func (p *PlanFileRepo) UpsertPlanFile(ctx context.Context, f domain.DiscoveredPlanFile) error {
	now := time.Now().Unix()
	_, err := p.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO discovered_plan_files
			(id, path, kind, format, project_path, summary, last_modified, is_active, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.ID, f.Path, string(f.Kind), f.Format, f.ProjectPath,
		f.Summary, f.LastModified.Unix(), boolToInt(f.IsActive), now,
	)
	if err != nil {
		return fmt.Errorf("repo_sqlite: upsert plan file: %w", err)
	}
	return nil
}

// ListPlanFiles returns all discovered plan file records for the given project path.
// If projectPath is empty, all plan files across all projects are returned.
func (p *PlanFileRepo) ListPlanFiles(ctx context.Context, projectPath string) ([]domain.DiscoveredPlanFile, error) {
	var rows *sql.Rows
	var err error
	if projectPath == "" {
		rows, err = p.db.QueryContext(ctx, `
			SELECT id, path, kind, format, project_path, summary, last_modified, is_active
			FROM discovered_plan_files
			ORDER BY last_modified DESC`)
	} else {
		rows, err = p.db.QueryContext(ctx, `
			SELECT id, path, kind, format, project_path, summary, last_modified, is_active
			FROM discovered_plan_files
			WHERE project_path = ?
			ORDER BY last_modified DESC`, projectPath)
	}
	if err != nil {
		return nil, fmt.Errorf("repo_sqlite: list plan files: %w", err)
	}
	defer rows.Close()

	var files []domain.DiscoveredPlanFile
	for rows.Next() {
		var f domain.DiscoveredPlanFile
		var kindStr string
		var lastModifiedUnix int64
		var isActiveInt int
		if err := rows.Scan(&f.ID, &f.Path, &kindStr, &f.Format, &f.ProjectPath,
			&f.Summary, &lastModifiedUnix, &isActiveInt); err != nil {
			return nil, fmt.Errorf("repo_sqlite: scan plan file: %w", err)
		}
		f.Kind = domain.PlanFileKind(kindStr)
		f.LastModified = time.Unix(lastModifiedUnix, 0)
		f.IsActive = isActiveInt != 0
		files = append(files, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repo_sqlite: list plan files: %w", err)
	}
	return files, nil
}

// DeleteStalePlanFiles deletes plan file records that have not been updated within olderThan.
// Returns the number of rows deleted.
func (p *PlanFileRepo) DeleteStalePlanFiles(ctx context.Context, olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan).Unix()
	res, err := p.db.ExecContext(ctx,
		`DELETE FROM discovered_plan_files WHERE updated_at < ?`, cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("repo_sqlite: delete stale plan files: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("repo_sqlite: delete stale plan files: rows affected: %w", err)
	}
	return int(n), nil
}
