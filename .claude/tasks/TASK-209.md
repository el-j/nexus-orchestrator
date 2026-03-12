---
id: TASK-209
title: Fix error handling consistency in core services
role: backend
planId: PLAN-030
status: todo
dependencies: []
createdAt: 2025-07-25T00:00:00.000Z
---

## Context
Multiple error handling inconsistencies exist across core services and outbound adapters: bare `return err` without wrapping, bare sentinel errors returned without context prefix, discarded `RowsAffected()` errors in SQLite repo, and missing "package:" prefix in `fmt.Errorf` calls. The project convention is `fmt.Errorf("package: operation: %w", err)` everywhere.

## Files to Read
- `internal/core/services/orchestrator.go` — line ~1010 (`PromoteProvider` bare `domain.ErrNotFound`)
- `internal/core/services/discovery.go` — lines ~92, ~120-125, ~140 (inconsistent wrapping)
- `internal/adapters/outbound/repo_sqlite/repo.go` — lines ~186, ~202, ~285 (`RowsAffected()` errors discarded with `_`)
- `internal/adapters/outbound/repo_sqlite/repo.go` — lines ~92, ~99 (migrate() raw errors unwrapped)

## Implementation Steps
1. In `orchestrator.go` `PromoteProvider()`: wrap `domain.ErrNotFound` → `fmt.Errorf("orchestrator: promote provider: %w", domain.ErrNotFound)`
2. In `discovery.go`: add "discovery:" prefix to all `fmt.Errorf` calls at lines ~92, ~120, ~140
3. In `repo.go` `UpdateStatus()`, `UpdateLogs()`, `Update()`: check `res.RowsAffected()` error instead of discarding with `_`
4. In `repo.go` `migrate()`: wrap raw errors with `fmt.Errorf("sqlite: migrate: %w", err)` at lines ~92, ~99
5. In `session_repo.go`: document why `tx.Rollback()` error suppression with `//nolint:errcheck` is acceptable, or handle properly
6. Grep entire codebase for bare `return err` without wrapping — fix any remaining instances in core/services and adapters/outbound

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `grep -rn 'return err$' internal/core/services/` shows zero results (all wrapped)
- [ ] All `RowsAffected()` calls properly check returned error
- [ ] All `fmt.Errorf` calls include package prefix per convention

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/`
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
