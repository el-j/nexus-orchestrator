---
id: TASK-512
title: Write knowledge repository tests
role: qa
planId: PLAN-066
status: todo
dependencies: [TASK-511]
createdAt: 2026-04-13T00:00:00Z
---

## Context

Comprehensive tests for `KnowledgeRepo` verify correctness of CRUD, upsert semantics, FTS5 search, status aggregation, and cascade delete. All tests use in-memory SQLite (`:memory:`) with CGO_ENABLED=1. Follow the pattern of existing repo tests.

## Files to Read

- `internal/adapters/outbound/repo_sqlite/knowledge_repo.go` — the implementation (TASK-511)
- `internal/adapters/outbound/repo_sqlite/repo_test.go` — test patterns (in-memory DB setup, helper functions)
- `internal/adapters/outbound/repo_sqlite/ai_session_repo_test.go` — secondary repo test patterns
- `internal/core/domain/brain.go` — types used in tests

## Implementation Steps

1. Create `internal/adapters/outbound/repo_sqlite/knowledge_repo_test.go`, package `repo_sqlite_test`

2. Helper: `setupKnowledgeRepo(t *testing.T) *KnowledgeRepo` — opens `:memory:` DB, runs migrations, returns repo

3. Helper: `makeKnowledge(projectPath, kind, topic, content string) domain.ProjectKnowledge` — creates a ProjectKnowledge with uuid ID, RelevanceScore 0.5, now timestamps

4. Test `TestKnowledgeRepo_SaveAndGetByID`:
   - Save a knowledge entry
   - GetByID returns exact same data (all fields round-trip)
   - GetByID with unknown ID returns error wrapping `domain.ErrNotFound`

5. Test `TestKnowledgeRepo_Upsert`:
   - Save entry with (project, kind, topic) = ("path", "convention", "error handling")
   - Save DIFFERENT content with same (project, kind, topic) — must succeed (INSERT OR REPLACE)
   - GetByProject returns exactly 1 entry with updated content
   - No duplicate entries

6. Test `TestKnowledgeRepo_GetByProject`:
   - Save 3 entries for projectA, 1 entry for projectB
   - GetByProject(projectA) returns exactly 3
   - GetByProject(projectB) returns exactly 1
   - GetByProject("nonexistent") returns empty slice (not error)

7. Test `TestKnowledgeRepo_GetByProjectAndKind`:
   - Save 2 conventions and 1 architecture for the same project
   - GetByProjectAndKind(project, KnowledgeConvention) returns exactly 2
   - GetByProjectAndKind(project, KnowledgeGlossary) returns empty slice

8. Test `TestKnowledgeRepo_SearchFTS`:
   - Save 3 entries: one with content "error wrapping fmt.Errorf package", one with "HTTP chi router middleware", one with "sqlite migration schema"
   - SearchFTS(project, "error", 10) returns the first entry
   - SearchFTS(project, "chi", 10) returns the second entry
   - SearchFTS(project, "nonexistent_xyz", 10) returns empty slice (not error)
   - Results are ordered (at least no panic, BM25 ranking applied)

9. Test `TestKnowledgeRepo_GetStatus`:
   - Empty project: Initialized=false, EntryCount=0
   - After saving 2 conventions + 1 architecture: Initialized=true, EntryCount=3, KindCounts["convention"]=2, KindCounts["architecture"]=1
   - TotalTokens equals sum of saved TokenCounts

10. Test `TestKnowledgeRepo_DeleteKnowledge`:
    - Save entry, delete by ID → GetByID returns not-found error
    - Delete non-existent ID → returns not-found error

11. Test `TestKnowledgeRepo_DeleteByProject`:
    - Save 3 entries for projectA, 2 for projectB
    - DeleteByProject(projectA) — GetByProject(projectA) returns empty, GetByProject(projectB) returns 2

12. Test `TestKnowledgeRepo_UpdateKnowledge`:
    - Save entry, update content and relevance score
    - GetByID returns updated values

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (all new tests pass)
- [ ] At least 9 test functions covering all interface methods
- [ ] FTS5 search test verifies BM25 returns relevant result for keyword query
- [ ] Upsert test verifies no duplicate creation

## Anti-patterns to Avoid

- NEVER use real filesystem paths in tests — always `:memory:` SQLite
- NEVER test internal SQL queries — test interface behavior only
- NEVER use `time.Sleep` — all operations should be synchronous
