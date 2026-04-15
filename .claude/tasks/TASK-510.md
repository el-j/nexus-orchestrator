---
id: TASK-510
title: Define brain port interfaces
role: backend
planId: PLAN-066
status: done
dependencies: [TASK-509]
createdAt: 2026-04-13T00:00:00Z
---

## Context

Port interfaces are the contracts between the core services and their adapters. Two new interfaces are needed: one for the storage layer (outbound port) and one for the brain service itself (inbound port, consumed by MCP/HTTP/CLI adapters). These live in a new file `internal/core/ports/brain.go` to avoid growing `ports.go` further.

## Files to Read

- `internal/core/ports/ports.go` — understand the port style: interface naming, method signatures, context.Context as first arg, error as last return
- `internal/core/domain/brain.go` — the types defined in TASK-509

## Implementation Steps

1. Create `internal/core/ports/brain.go` with package `ports`

2. Imports: `"context"`, `"nexus-orchestrator/internal/core/domain"` (check actual module name in go.mod)

3. Define `ProjectKnowledgeRepository` interface (8 methods):

   ```go
   SaveKnowledge(ctx context.Context, k domain.ProjectKnowledge) error
   GetByID(ctx context.Context, id string) (domain.ProjectKnowledge, error)
   UpdateKnowledge(ctx context.Context, k domain.ProjectKnowledge) error
   DeleteKnowledge(ctx context.Context, id string) error
   GetByProject(ctx context.Context, projectPath string) ([]domain.ProjectKnowledge, error)
   GetByProjectAndKind(ctx context.Context, projectPath string, kind domain.KnowledgeKind) ([]domain.ProjectKnowledge, error)
   SearchFTS(ctx context.Context, projectPath, query string, maxResults int) ([]domain.ProjectKnowledge, error)
   GetStatus(ctx context.Context, projectPath string) (domain.BrainStatus, error)
   DeleteByProject(ctx context.Context, projectPath string) error
   ```

4. Add doc comment to `ProjectKnowledgeRepository`: "ProjectKnowledgeRepository is the outbound port for persisting and querying project knowledge entries. Implementations use SQLite with FTS5 for BM25 full-text search."

5. Define `BrainService` interface (8 methods):

   ```go
   GetContext(ctx context.Context, q domain.ContextQuery) (domain.ContextResponse, error)
   GetFocusedContext(ctx context.Context, q domain.ContextQuery) (domain.ContextResponse, error)
   IngestKnowledge(ctx context.Context, k domain.ProjectKnowledge) (domain.ProjectKnowledge, error)
   IngestFromFile(ctx context.Context, projectPath, filePath string) (int, error)
   SearchKnowledge(ctx context.Context, projectPath, query string, maxTokens int) ([]domain.ContextSection, error)
   GetFileMap(ctx context.Context, projectPath, focusArea string) ([]string, error)
   InitProject(ctx context.Context, projectPath, claudeMDPath string) (domain.BrainStatus, error)
   GetStatus(ctx context.Context, projectPath string) (domain.BrainStatus, error)
   ```

6. Add doc comment to `BrainService`: "BrainService is the inbound port for project context intelligence. It assembles minimal token-budget-constrained context responses for AI agents."

7. Add doc comments to each method (1 line each) describing what it does and the key constraint (e.g. "GetContext assembles a ContextResponse in priority order within the MaxTokens budget (default 400).").

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `internal/core/ports/brain.go` exists with both interfaces
- [ ] `ProjectKnowledgeRepository` has exactly 9 methods
- [ ] `BrainService` has exactly 8 methods
- [ ] No method uses `interface{}` or `any`

## Anti-patterns to Avoid

- NEVER import adapter packages from ports
- NEVER add concrete types to ports — interfaces only
- NEVER use goroutines or channels in port definitions
