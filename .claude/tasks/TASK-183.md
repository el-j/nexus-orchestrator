---
id: TASK-183
title: QA — unit and integration tests for backlog lifecycle
role: qa
planId: PLAN-024
status: done
dependencies: [TASK-175, TASK-176, TASK-177]
createdAt: 2026-03-11T22:00:00.000Z
---

## Context
All backlog lifecycle changes need comprehensive tests: domain validation, service logic, SQLite queries, HTTP endpoints, and MCP tools. The explicit provider routing path needs dedicated test coverage.

## Files to Read
- `internal/core/services/orchestrator_test.go` — existing test patterns
- `internal/adapters/outbound/repo_sqlite/repo_test.go` — SQLite test patterns
- `internal/adapters/inbound/httpapi/server_test.go` — HTTP test patterns
- `internal/adapters/inbound/mcp/server_test.go` — MCP test patterns

## Implementation Steps

1. **Domain unit tests** — `internal/core/domain/task_test.go`:
   - Test `IsExecutable()` returns false for DRAFT, BACKLOG, PROCESSING, COMPLETED
   - Test `IsExecutable()` returns true for QUEUED
   - Test `CommandType.IsValid()` for all values

2. **Service unit tests** — extend `orchestrator_test.go`:
   - Test `CreateDraft` → task persisted with StatusDraft, NOT enqueued
   - Test `GetBacklog` → returns only DRAFT+BACKLOG for project, sorted by priority
   - Test `PromoteTask` → DRAFT→QUEUED, enqueued, SSE fired
   - Test `PromoteTask` on QUEUED task → error
   - Test `UpdateTask` → fields merged correctly
   - Test `processNext` with `ProviderName` set → uses exact provider, not discovery
   - Test `processNext` with `ProviderName` set to non-existent → StatusNoProvider

3. **SQLite tests** — extend `repo_test.go`:
   - Test `GetByProjectPathAndStatus` with multiple status filters
   - Test `Update` persists ProviderName, Priority, Tags
   - Test `Update` on non-existent ID → ErrNotFound
   - Test Tags JSON roundtrip ([]string → "["a","b"]" → []string)

4. **HTTP API tests** — extend `server_test.go`:
   - Test `POST /api/tasks/draft` → 201 + StatusDraft
   - Test `GET /api/tasks/backlog/{projectPath}` → filtered results
   - Test `POST /api/tasks/{id}/promote` → 204
   - Test `POST /api/tasks/{id}/promote` on non-existent → 404
   - Test `PUT /api/tasks/{id}` → updated fields returned

5. **MCP tests** — extend `server_test.go`:
   - Test `create_draft` tool
   - Test `get_backlog` tool
   - Test `promote_task` tool
   - Test `update_task` tool

## Acceptance Criteria
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` passes
- [ ] At least 20 new test cases
- [ ] No data races under `-race`
- [ ] Mock SystemScanner / mock TaskRepo used where appropriate
- [ ] Full backlog lifecycle covered: create → update → promote → execute → complete

## Anti-patterns to Avoid
- NEVER skip `-race` flag
- NEVER test backlog logic by reaching into private fields — use public API only
- NEVER use real LLM calls in unit tests
