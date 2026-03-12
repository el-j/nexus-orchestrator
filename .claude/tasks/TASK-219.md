---
id: TASK-219
title: Add provider_config_repo + Wails binding tests
role: qa
planId: PLAN-030
status: todo
dependencies: []
createdAt: 2025-07-25T00:00:00.000Z
---

## Context
Two significant test gaps: (1) `provider_config_repo.go` has zero test coverage despite handling CRUD for provider configurations including UUID generation, upsert logic, and delete operations. (2) The Wails binding layer (`app.go`) has zero tests despite being the JavaScript-to-Go gateway for the desktop GUI.

## Files to Read
- `internal/adapters/outbound/repo_sqlite/provider_config_repo.go` — full implementation
- `internal/adapters/outbound/repo_sqlite/repo_test.go` — test patterns used by task repo
- `internal/adapters/outbound/repo_sqlite/ai_session_repo_test.go` — test patterns used by AI session repo
- `app.go` — Wails App struct, all bound methods

## Implementation Steps
1. Create `internal/adapters/outbound/repo_sqlite/provider_config_repo_test.go`:
   - Test `SaveProviderConfig` — new config with auto-generated UUID, update existing config
   - Test `ListProviderConfigs` — empty list, multiple configs, ordering
   - Test `GetProviderConfig` — found, not found (returns `domain.ErrNotFound`)
   - Test `DeleteProviderConfig` — existing config, non-existent config
   - Test transaction integrity — concurrent saves don't corrupt
   - Use same `newTestRepo()` helper pattern as existing repo tests
2. Create `app_test.go` for Wails binding tests:
   - Use a mock orchestrator (interface-based) to verify delegation
   - Test `SubmitTask` — verifies orchestrator.SubmitTask called with correct args
   - Test `GetTask` — verifies correct ID forwarded
   - Test `GetQueue` — verifies slice returned
   - Test `CancelTask` — verifies cancellation delegated
   - Test error propagation — ensure errors from orchestrator surface correctly
3. Verify all methods in App struct are tested at least once

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `provider_config_repo_test.go` covers all 4 CRUD operations
- [ ] `app_test.go` covers delegation for at least 6 key methods
- [ ] All tests pass with `-race` flag

## Anti-patterns to Avoid
- NEVER test against a real database — use in-memory SQLite (`:memory:`)
- NEVER skip error path testing
- NEVER import core services in adapter tests — mock via interfaces
