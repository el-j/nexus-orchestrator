---
id: TASK-514
title: Write BrainService unit tests
role: qa
planId: PLAN-066
status: todo
dependencies: [TASK-513]
createdAt: 2026-04-13T00:00:00Z
---

## Context

Unit tests for `BrainServiceImpl` covering the token-budget assembly logic, BM25 delegation, markdown ingestion, and project initialization. Tests use the real SQLite repository with `:memory:` DB (not a hand-rolled mock) to verify end-to-end service behavior with actual FTS5 search. External test package.

## Files to Read

- `internal/core/services/brain_service.go` — the implementation (TASK-513)
- `internal/core/domain/brain.go` — domain types (TASK-509)
- `internal/core/services/orchestrator_test.go` — understand service test patterns if it exists
- `internal/adapters/outbound/repo_sqlite/knowledge_repo.go` — for setupTestRepo helper

## Implementation Steps

1. Create `internal/core/services/brain_service_test.go`, package `services_test`

2. Helper `setupBrainService(t *testing.T) (*BrainServiceImpl, *KnowledgeRepo)`:
   - Open `:memory:` SQLite, run migrations, create KnowledgeRepo, create BrainServiceImpl with nil taskRepo (not needed for brain tests)
   - Note: need to import the repo package — check import path

3. Constant `sampleCLAUDEMD` — a multi-section markdown string:

   ```
   ## Architecture
   Hexagonal architecture. Core never imports adapters. Layers: domain, ports, services, adapters.

   ## Error Handling Conventions
   Always wrap errors: fmt.Errorf("package: function: %w", err). Use domain.ErrNotFound for missing entities.

   ## File Map
   internal/core/domain/ - pure types
   internal/core/ports/ - interfaces
   internal/core/services/ - business logic
   internal/adapters/ - inbound and outbound adapters

   ## Glossary
   Inbound adapter: driven by CLI, HTTP, MCP. Outbound: drives SQLite, LLM providers.
   ```

4. Test `TestBrainService_GetContext_TokenBudget`:
   - Ingest 5 entries with varying TokenCounts (50, 80, 100, 120, 200)
   - Call GetContext with MaxTokens=200
   - Assert TokensUsed <= 200
   - Assert Truncated == true (not all entries fit)
   - Assert len(Sections) >= 1

5. Test `TestBrainService_GetContext_PriorityOrder`:
   - Ingest 1 architecture, 1 convention, 1 learning entry
   - Call GetContext with MaxTokens=10000 (no truncation)
   - Assert first section Kind == KnowledgeArchitecture
   - Assert second section Kind == KnowledgeConvention
   - Assert third section Kind == KnowledgeLearning

6. Test `TestBrainService_GetContext_FocusArea`:
   - Ingest 3 conventions: topics "error handling", "HTTP routing", "database migrations"
   - Call GetContext with FocusArea="error"
   - Assert only the "error handling" convention is in Sections (others filtered)

7. Test `TestBrainService_GetContext_EmptyProject`:
   - Call GetContext on a project with no knowledge
   - Returns ContextResponse with empty Sections, TokensUsed=0, Truncated=false

8. Test `TestBrainService_GetFocusedContext`:
   - Ingest entries with content about "error wrapping" and "HTTP chi routing"
   - Call GetFocusedContext with Question="error"
   - Assert at least 1 section returned with error-related content

9. Test `TestBrainService_IngestKnowledge_TokenCount`:
   - Ingest knowledge with content of known length (e.g. 400 chars)
   - GetByID on returned entry — assert TokenCount ≈ 100 (±5)

10. Test `TestBrainService_IngestFromFile`:
    - Write sampleCLAUDEMD to a temp file (`t.TempDir()`)
    - Call IngestFromFile with the temp file path
    - Assert count >= 3 (at least 3 sections ingested)
    - GetByProject — verify at least one KnowledgeArchitecture, one KnowledgeConvention, one KnowledgeFileMap entry

11. Test `TestBrainService_InitProject_AutoDetect`:
    - Create a temp directory with `CLAUDE.md` containing sampleCLAUDEMD
    - Call InitProject(ctx, tempDir, "") — empty claudeMDPath
    - Assert returned BrainStatus.Initialized == true
    - Assert BrainStatus.EntryCount >= 3

12. Test `TestBrainService_InitProject_NoCLAUDE`:
    - Call InitProject on a directory with no CLAUDE.md
    - Should NOT error — just return BrainStatus with Initialized=false

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (all new tests pass)
- [ ] Token budget test proves GetContext never exceeds MaxTokens
- [ ] Priority order test proves Architecture comes before Convention comes before Learning
- [ ] FocusArea filter test proves filtering works
- [ ] IngestFromFile test proves markdown section splitting works

## Anti-patterns to Avoid

- NEVER mock the repository with a hand-rolled stub — use the real SQLite `:memory:` repo
- NEVER test implementation internals — test service behavior via public methods
- NEVER hardcode expected TokenCount exactly — use ranges (±10%)
