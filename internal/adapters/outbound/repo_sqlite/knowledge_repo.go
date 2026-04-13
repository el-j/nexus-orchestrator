package repo_sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"

	"github.com/google/uuid"
)

// compile-time interface check
var _ ports.ProjectKnowledgeRepository = (*KnowledgeRepo)(nil)

// KnowledgeRepo implements ports.ProjectKnowledgeRepository using the shared SQLite database.
type KnowledgeRepo struct {
	db *sql.DB
}

// NewKnowledgeRepo returns a KnowledgeRepo backed by the same database as r.
func NewKnowledgeRepo(r *Repository) *KnowledgeRepo {
	return &KnowledgeRepo{db: r.db}
}

func migrateKnowledge(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS project_knowledge (
			id             TEXT    PRIMARY KEY,
			project_path   TEXT    NOT NULL,
			kind           TEXT    NOT NULL,
			topic          TEXT    NOT NULL,
			content        TEXT    NOT NULL,
			source         TEXT    NOT NULL DEFAULT '',
			token_count    INTEGER NOT NULL DEFAULT 0,
			relevance_score REAL   NOT NULL DEFAULT 0.5,
			created_at     INTEGER NOT NULL,
			updated_at     INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_pk_project ON project_knowledge(project_path);
		CREATE INDEX IF NOT EXISTS idx_pk_project_kind ON project_knowledge(project_path, kind);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_pk_upsert ON project_knowledge(project_path, kind, topic);
		CREATE VIRTUAL TABLE IF NOT EXISTS project_knowledge_fts USING fts5(
			topic, content,
			content='project_knowledge',
			content_rowid='rowid'
		);
		CREATE TRIGGER IF NOT EXISTS pk_ai AFTER INSERT ON project_knowledge BEGIN
			INSERT INTO project_knowledge_fts(rowid, topic, content)
			VALUES (new.rowid, new.topic, new.content);
		END;
		CREATE TRIGGER IF NOT EXISTS pk_ad AFTER DELETE ON project_knowledge BEGIN
			INSERT INTO project_knowledge_fts(project_knowledge_fts, rowid, topic, content)
			VALUES ('delete', old.rowid, old.topic, old.content);
		END;
		CREATE TRIGGER IF NOT EXISTS pk_au AFTER UPDATE ON project_knowledge BEGIN
			INSERT INTO project_knowledge_fts(project_knowledge_fts, rowid, topic, content)
			VALUES ('delete', old.rowid, old.topic, old.content);
			INSERT INTO project_knowledge_fts(rowid, topic, content)
			VALUES (new.rowid, new.topic, new.content);
		END;
		CREATE TABLE IF NOT EXISTS context_query_log (
			id            TEXT    PRIMARY KEY,
			project_path  TEXT    NOT NULL,
			query_json    TEXT    NOT NULL,
			tokens_used   INTEGER NOT NULL DEFAULT 0,
			section_count INTEGER NOT NULL DEFAULT 0,
			truncated     INTEGER NOT NULL DEFAULT 0,
			created_at    INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_cql_project ON context_query_log(project_path);
	`)
	if err != nil {
		return fmt.Errorf("knowledge_repo: migrate: %w", err)
	}
	return nil
}

// scanKnowledge scans a single row into a ProjectKnowledge value.
func scanKnowledge(row interface {
	Scan(dest ...any) error
}) (domain.ProjectKnowledge, error) {
	var k domain.ProjectKnowledge
	var createdMs, updatedMs int64
	if err := row.Scan(
		&k.ID, &k.ProjectPath, &k.Kind, &k.Topic, &k.Content,
		&k.Source, &k.TokenCount, &k.RelevanceScore,
		&createdMs, &updatedMs,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ProjectKnowledge{}, fmt.Errorf("knowledge_repo: get: %w", domain.ErrNotFound)
		}
		return domain.ProjectKnowledge{}, fmt.Errorf("knowledge_repo: scan: %w", err)
	}
	k.CreatedAt = time.UnixMilli(createdMs)
	k.UpdatedAt = time.UnixMilli(updatedMs)
	return k, nil
}

// SaveKnowledge inserts or replaces a knowledge entry (upsert by project_path, kind, topic).
func (r *KnowledgeRepo) SaveKnowledge(ctx context.Context, k domain.ProjectKnowledge) error {
	if k.ID == "" {
		k.ID = uuid.New().String()
	}
	k.ProjectPath = filepath.Clean(k.ProjectPath)
	now := time.Now().UnixMilli()
	if k.CreatedAt.IsZero() {
		k.CreatedAt = time.Now()
	}
	k.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO project_knowledge
			(id, project_path, kind, topic, content, source, token_count, relevance_score, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_path, kind, topic) DO UPDATE SET
			id             = excluded.id,
			content        = excluded.content,
			source         = excluded.source,
			token_count    = excluded.token_count,
			relevance_score = excluded.relevance_score,
			updated_at     = excluded.updated_at`,
		k.ID, k.ProjectPath, string(k.Kind), k.Topic, k.Content,
		k.Source, k.TokenCount, k.RelevanceScore,
		k.CreatedAt.UnixMilli(), now,
	)
	if err != nil {
		return fmt.Errorf("knowledge_repo: save: %w", err)
	}
	return nil
}

// GetByID returns a knowledge entry by its UUID, or domain.ErrNotFound.
func (r *KnowledgeRepo) GetByID(ctx context.Context, id string) (domain.ProjectKnowledge, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, project_path, kind, topic, content, source, token_count, relevance_score, created_at, updated_at
		FROM project_knowledge WHERE id = ?`, id)
	return scanKnowledge(row)
}

// UpdateKnowledge replaces the mutable fields of an existing entry.
func (r *KnowledgeRepo) UpdateKnowledge(ctx context.Context, k domain.ProjectKnowledge) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE project_knowledge
		SET kind = ?, topic = ?, content = ?, source = ?, token_count = ?, relevance_score = ?, updated_at = ?
		WHERE id = ?`,
		string(k.Kind), k.Topic, k.Content, k.Source,
		k.TokenCount, k.RelevanceScore, time.Now().UnixMilli(), k.ID,
	)
	if err != nil {
		return fmt.Errorf("knowledge_repo: update: %w", err)
	}
	return nil
}

// DeleteKnowledge removes an entry by ID, returning domain.ErrNotFound if not found.
func (r *KnowledgeRepo) DeleteKnowledge(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM project_knowledge WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("knowledge_repo: delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("knowledge_repo: delete: %w", domain.ErrNotFound)
	}
	return nil
}

// GetByProject returns all knowledge entries for a project, ordered by kind then relevance.
func (r *KnowledgeRepo) GetByProject(ctx context.Context, projectPath string) ([]domain.ProjectKnowledge, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_path, kind, topic, content, source, token_count, relevance_score, created_at, updated_at
		FROM project_knowledge WHERE project_path = ?
		ORDER BY kind, relevance_score DESC`,
		filepath.Clean(projectPath),
	)
	if err != nil {
		return nil, fmt.Errorf("knowledge_repo: get by project: %w", err)
	}
	defer rows.Close()
	return collectKnowledge(rows)
}

// GetByProjectAndKind returns entries filtered by project and kind, ordered by relevance.
func (r *KnowledgeRepo) GetByProjectAndKind(ctx context.Context, projectPath string, kind domain.KnowledgeKind) ([]domain.ProjectKnowledge, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_path, kind, topic, content, source, token_count, relevance_score, created_at, updated_at
		FROM project_knowledge WHERE project_path = ? AND kind = ?
		ORDER BY relevance_score DESC`,
		filepath.Clean(projectPath), string(kind),
	)
	if err != nil {
		return nil, fmt.Errorf("knowledge_repo: get by kind: %w", err)
	}
	defer rows.Close()
	return collectKnowledge(rows)
}

// SearchFTS performs BM25 full-text search within a project up to maxResults.
func (r *KnowledgeRepo) SearchFTS(ctx context.Context, projectPath, query string, maxResults int) ([]domain.ProjectKnowledge, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT pk.id, pk.project_path, pk.kind, pk.topic, pk.content, pk.source,
		       pk.token_count, pk.relevance_score, pk.created_at, pk.updated_at
		FROM project_knowledge pk
		JOIN project_knowledge_fts fts ON pk.rowid = fts.rowid
		WHERE project_knowledge_fts MATCH ?
		  AND pk.project_path = ?
		ORDER BY bm25(project_knowledge_fts)
		LIMIT ?`,
		query, filepath.Clean(projectPath), maxResults,
	)
	if err != nil {
		return nil, fmt.Errorf("knowledge_repo: search fts: %w", err)
	}
	defer rows.Close()
	return collectKnowledge(rows)
}

// GetStatus aggregates entry counts, token sums, and kind breakdown for a project.
func (r *KnowledgeRepo) GetStatus(ctx context.Context, projectPath string) (domain.BrainStatus, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT kind, COUNT(*), SUM(token_count), MAX(updated_at)
		FROM project_knowledge WHERE project_path = ?
		GROUP BY kind`,
		filepath.Clean(projectPath),
	)
	if err != nil {
		return domain.BrainStatus{}, fmt.Errorf("knowledge_repo: get status: %w", err)
	}
	defer rows.Close()

	status := domain.BrainStatus{
		ProjectPath: filepath.Clean(projectPath),
		KindCounts:  make(map[string]int),
	}
	var maxUpdatedMs int64
	for rows.Next() {
		var kind string
		var count, tokenSum int
		var updatedMs int64
		if err := rows.Scan(&kind, &count, &tokenSum, &updatedMs); err != nil {
			return domain.BrainStatus{}, fmt.Errorf("knowledge_repo: scan status: %w", err)
		}
		status.EntryCount += count
		status.KindCounts[kind] = count
		status.TotalTokens += tokenSum
		if updatedMs > maxUpdatedMs {
			maxUpdatedMs = updatedMs
		}
	}
	if err := rows.Err(); err != nil {
		return domain.BrainStatus{}, fmt.Errorf("knowledge_repo: status rows: %w", err)
	}
	status.Initialized = status.EntryCount > 0
	if maxUpdatedMs > 0 {
		status.LastUpdated = time.UnixMilli(maxUpdatedMs)
	}
	return status, nil
}

// DeleteByProject removes all knowledge entries for a project.
func (r *KnowledgeRepo) DeleteByProject(ctx context.Context, projectPath string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM project_knowledge WHERE project_path = ?`,
		filepath.Clean(projectPath))
	if err != nil {
		return fmt.Errorf("knowledge_repo: delete by project: %w", err)
	}
	return nil
}

// collectKnowledge drains sql.Rows into a slice of ProjectKnowledge values.
func collectKnowledge(rows *sql.Rows) ([]domain.ProjectKnowledge, error) {
	var out []domain.ProjectKnowledge
	for rows.Next() {
		k, err := scanKnowledge(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("knowledge_repo: rows: %w", err)
	}
	return out, nil
}
