// Package ports defines the hexagonal architecture port interfaces for nexus-orchestrator.
// This file contains the brain / context-intelligence port interfaces.
package ports

import (
	"context"

	"nexus-orchestrator/internal/core/domain"
)

// ProjectKnowledgeRepository is the outbound port for persisting and querying project knowledge entries.
// Implementations use SQLite with FTS5 for BM25 full-text search.
type ProjectKnowledgeRepository interface {
	// SaveKnowledge inserts or replaces a knowledge entry (upsert by project_path, kind, topic).
	SaveKnowledge(ctx context.Context, k domain.ProjectKnowledge) error
	// GetByID returns a knowledge entry by its UUID, or domain.ErrNotFound.
	GetByID(ctx context.Context, id string) (domain.ProjectKnowledge, error)
	// UpdateKnowledge replaces the mutable fields of an existing entry.
	UpdateKnowledge(ctx context.Context, k domain.ProjectKnowledge) error
	// DeleteKnowledge removes an entry by ID, or returns domain.ErrNotFound.
	DeleteKnowledge(ctx context.Context, id string) error
	// GetByProject returns all knowledge entries for a project, ordered by kind then relevance.
	GetByProject(ctx context.Context, projectPath string) ([]domain.ProjectKnowledge, error)
	// GetByProjectAndKind returns entries filtered by project and kind, ordered by relevance.
	GetByProjectAndKind(ctx context.Context, projectPath string, kind domain.KnowledgeKind) ([]domain.ProjectKnowledge, error)
	// SearchFTS performs BM25 full-text search within a project up to maxResults.
	SearchFTS(ctx context.Context, projectPath, query string, maxResults int) ([]domain.ProjectKnowledge, error)
	// GetStatus aggregates entry counts, token sums, and kind breakdown for a project.
	GetStatus(ctx context.Context, projectPath string) (domain.BrainStatus, error)
	// DeleteByProject removes all knowledge entries for a project.
	DeleteByProject(ctx context.Context, projectPath string) error
}

// BrainService is the inbound port for project context intelligence.
// It assembles minimal token-budget-constrained context responses for AI agents.
type BrainService interface {
	// GetContext assembles a ContextResponse in priority order within the MaxTokens budget (default 400).
	GetContext(ctx context.Context, q domain.ContextQuery) (domain.ContextResponse, error)
	// GetFocusedContext uses BM25 search to assemble context relevant to a specific question.
	GetFocusedContext(ctx context.Context, q domain.ContextQuery) (domain.ContextResponse, error)
	// IngestKnowledge stores a knowledge entry, computing token count and setting timestamps.
	IngestKnowledge(ctx context.Context, k domain.ProjectKnowledge) (domain.ProjectKnowledge, error)
	// IngestFromFile parses a CLAUDE.md-style markdown file and ingests each ## section.
	IngestFromFile(ctx context.Context, projectPath, filePath string) (int, error)
	// SearchKnowledge performs BM25 search and returns ContextSection slices within budget.
	SearchKnowledge(ctx context.Context, projectPath, query string, maxTokens int) ([]domain.ContextSection, error)
	// GetFileMap returns file path strings from file_map knowledge entries, optionally filtered.
	GetFileMap(ctx context.Context, projectPath, focusArea string) ([]string, error)
	// InitProject ingests the project CLAUDE.md (auto-detected if claudeMDPath is empty).
	InitProject(ctx context.Context, projectPath, claudeMDPath string) (domain.BrainStatus, error)
	// GetStatus returns the current brain status for a project.
	GetStatus(ctx context.Context, projectPath string) (domain.BrainStatus, error)
	// ListKnowledge returns all knowledge entries for a project, optionally filtered by kind.
	ListKnowledge(ctx context.Context, projectPath, kind string) ([]domain.ProjectKnowledge, error)
	// DeleteKnowledge removes a knowledge entry by ID.
	DeleteKnowledge(ctx context.Context, id string) error
}
