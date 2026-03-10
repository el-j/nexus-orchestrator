---
id: TASK-067
title: "QA: tests for path traversal, SQLite integrity, error handling, type safety"
role: qa
planId: PLAN-007
status: todo
dependencies: [TASK-060, TASK-061, TASK-062, TASK-063, TASK-064, TASK-065, TASK-066]
createdAt: 2026-03-10T10:00:00.000Z
---

## Context
The audit identified many untested paths. This task adds comprehensive tests for all PLAN-007 changes to ensure bulletproof coverage.

## Files to Read
- `internal/adapters/outbound/fs_writer/writer_test.go`
- `internal/adapters/outbound/repo_sqlite/repo_test.go`
- `internal/core/services/orchestrator_test.go`
- `internal/adapters/inbound/httpapi/server_test.go`
- `internal/adapters/inbound/mcp/server_test.go`

## Implementation Steps
1. **fs_writer tests**: Add table-driven tests for path traversal: `../../etc/passwd`, `../sibling`, `/etc/passwd`, valid relative path. Verify error on traversal, success on valid.
2. **SQLite tests**: Test `UpdateStatus` on non-existent ID returns `domain.ErrNotFound`. Test that WAL mode is active after `New()`. Test concurrent writes don't produce SQLITE_BUSY.
3. **Orchestrator tests**: Test `CancelTask("unknown")` returns error wrapping `domain.ErrNotFound`. Test `Stop()` then `SubmitTask()` returns error. Test `Stop()` idempotency (call twice without panic).
4. **HTTP API tests**: Test cancel of PROCESSING task returns proper error. Test oversized request body returns 4xx. Test security headers present on responses.
5. **Type safety tests**: Verify `domain.RoleUser` is typed `MessageRole`. Verify `TaskStatus.String()` works.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] All new tests pass with `-race` flag
- [ ] Coverage increases for fs_writer, repo_sqlite, orchestrator, httpapi packages

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/`
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
