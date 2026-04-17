---
id: TASK-569
title: Go repo tests — activity_repo complex dynamic SQL and runtime_config_repo
role: qa
planId: PLAN-071
status: todo
dependencies: [TASK-557]
createdAt: 2026-04-15T00:00:00Z
---

## Context

`activity_repo.go` has zero tests. Its `ListActivities` method builds dynamic SQL based on a `domain.ActivityFilter` struct with five optional fields (agentName, projectPath, type, since, limit). Any regression in the filter builder is silent. `runtime_config_repo.go` also has zero tests and is daemon-startup critical — corrupt config will crash the daemon with no safety net.

## Files to Read

- `internal/adapters/outbound/repo_sqlite/activity_repo.go`
- `internal/adapters/outbound/repo_sqlite/runtime_config_repo.go`
- `internal/adapters/outbound/repo_sqlite/repo.go` (shared test helpers)
- Any existing `*_test.go` in `repo_sqlite/` to understand test patterns

## Implementation Steps

1. Create `internal/adapters/outbound/repo_sqlite/activity_repo_test.go`:
   - `TestSaveActivity_RoundTrip` — save then retrieve by ID
   - `TestListActivities_FilterByAgentName` — only matching agent returned
   - `TestListActivities_FilterByProjectPath` — only matching project returned
   - `TestListActivities_FilterBySince` — only activities after cutoff returned
   - `TestListActivities_FilterByType` — only matching activity type returned
   - `TestListActivities_Limit` — respects limit field
   - `TestListActivities_CombinedFilters` — all filters combined
   - `TestPurgeOlderThan` — entries before cutoff deleted, newer entries retained
2. Create `internal/adapters/outbound/repo_sqlite/runtime_config_repo_test.go`:
   - `TestGetRuntimeConfig_Default` — returns sensible defaults on empty DB
   - `TestSaveRuntimeConfig_RoundTrip` — save then get returns same values
   - `TestSaveRuntimeConfig_Overwrite` — second save replaces first
   - `TestGetRuntimeConfig_CorruptRow` — corrupted row returns error, not panic
3. Use an in-memory SQLite DB for all tests (`:memory:` DSN); run full migrations before each test

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./internal/adapters/outbound/repo_sqlite/...` exits 0 with all new tests passing
- [ ] `activity_repo_test.go` covers all 5 filter fields independently and combined
- [ ] `runtime_config_repo_test.go` covers default, roundtrip, overwrite, and corrupt-row cases
- [ ] No `time.Sleep` in new tests — use deterministic timestamps

## Anti-patterns to Avoid

- NEVER use `time.Sleep` for test synchronisation — use explicit timestamp manipulation instead
- NEVER share DB state between tests — each test gets its own `:memory:` instance
