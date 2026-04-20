---
id: TASK-570
title: Go repo tests — model_capability_repo and plan_file_repo full CRUD coverage
role: qa
planId: PLAN-071
status: todo
dependencies: [TASK-557]
createdAt: 2026-04-15T00:00:00Z
---

## Context

`model_capability_repo.go` and `plan_file_repo.go` both have zero test files. The model capability repo stores LLM context window profiles used for routing decisions. The plan file repo is the persistence layer for discovered `.claude/plans/` files. Both have critical production paths with no safety net.

## Files to Read

- `internal/adapters/outbound/repo_sqlite/model_capability_repo.go`
- `internal/adapters/outbound/repo_sqlite/plan_file_repo.go`
- `internal/core/domain/` (model capability and plan file domain types)
- Any existing `*_test.go` in `repo_sqlite/` for test setup patterns

## Implementation Steps

1. Create `internal/adapters/outbound/repo_sqlite/model_capability_repo_test.go`:
   - `TestSaveModelCapability_RoundTrip` — save then GetByModelID returns same record
   - `TestGetByModelID_NotFound` — returns `domain.ErrNotFound` (not nil, nil)
   - `TestGetAll_Empty` — returns empty slice, not nil
   - `TestGetAll_MultipleModels` — all saved models returned
   - `TestDeleteModelCapability` — deleted model not returned by GetAll
   - `TestSaveModelCapability_Update` — second save with same modelID overwrites (upsert)
2. Create `internal/adapters/outbound/repo_sqlite/plan_file_repo_test.go`:
   - `TestUpsertPlanFile_RoundTrip` — upsert then ListPlanFiles returns it
   - `TestUpsertPlanFile_Update` — second upsert with same ID updates fields
   - `TestListPlanFiles_FilterByProjectPath` — only matching project returned
   - `TestListPlanFiles_NoFilter` — returns all plans across projects
   - `TestDeleteStalePlanFiles` — files with updatedAt before cutoff are deleted, newer retained
3. Use `:memory:` SQLite and run full migrations before each test

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./internal/adapters/outbound/repo_sqlite/...` exits 0
- [ ] `model_capability_repo_test.go` covers Save, GetByModelID, GetAll, Delete, Upsert
- [ ] `plan_file_repo_test.go` covers Upsert, List with/without filter, DeleteStale
- [ ] `domain.ErrNotFound` is returned for missing-record cases (not nil error)
- [ ] No `time.Sleep` in new tests

## Anti-patterns to Avoid

- NEVER return `nil, nil` for not-found — use `domain.ErrNotFound`
- NEVER share DB state between tests
