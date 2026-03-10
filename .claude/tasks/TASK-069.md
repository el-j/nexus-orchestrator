---
id: TASK-069
title: HTTP API comprehensive tests (provider CRUD, cancel, errors)
role: qa
planId: PLAN-008
status: todo
dependencies: []
createdAt: 2026-03-10T14:00:00.000Z
---

## Context
The existing integration test in `services_test` covers the happy path (submit → poll → complete) and basic endpoints (health, dashboard, redirect, SSE). We need comprehensive HTTP API tests covering: provider management lifecycle (POST → GET → GET models → DELETE), task cancellation workflow, and error scenarios (bad JSON, missing fields, not-found, oversized body). These tests use the **real** testStack with real SQLite, not mocks.

## Files to Read
- `internal/core/services/integration_test.go` (testStack pattern + existing tests)
- `internal/adapters/inbound/httpapi/server.go` (all handlers)
- `internal/adapters/inbound/httpapi/server_test.go` (existing mock-based tests)
- `internal/core/domain/task.go`
- `internal/core/domain/provider.go`
- `internal/core/ports/ports.go`

## Implementation Steps

1. Add new tests to `internal/core/services/integration_test.go` (same file, same `testStack`).
2. Write `TestCancelTaskHTTP`:
   - Submit a task via POST /api/tasks.
   - Immediately DELETE /api/tasks/{id}.
   - GET /api/tasks/{id} and verify status is `CANCELLED`.
3. Write `TestProviderLifecycleHTTP`:
   - GET /api/providers — verify MockLLM is present and active.
   - POST /api/providers with a valid ProviderConfig — verify 201.
   - GET /api/providers — verify the new provider appears.
   - GET /api/providers/{name}/models — verify models returned.
   - DELETE /api/providers/{name} — verify 204.
   - GET /api/providers — verify provider is gone.
4. Write `TestHTTPErrorScenarios` (table-driven):
   - POST /api/tasks with invalid JSON → 400.
   - POST /api/tasks with empty body → 400.
   - GET /api/tasks/nonexistent-id → 404.
   - DELETE /api/tasks/nonexistent-id → 404.
   - DELETE /api/providers/nonexistent → 404.
   - GET /api/providers/nonexistent/models → 404.
5. Write `TestMultipleTasksCompleteHTTP`:
   - Submit 3 tasks rapidly via POST /api/tasks.
   - Poll all 3 until COMPLETED (timeout 20s).
   - Verify all 3 output files written.
   - GET /api/tasks — verify all visible.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `TestCancelTaskHTTP` verifies CANCELLED status via GET after DELETE
- [ ] `TestProviderLifecycleHTTP` covers POST/GET/DELETE provider cycle
- [ ] `TestHTTPErrorScenarios` validates at least 6 error cases
- [ ] `TestMultipleTasksCompleteHTTP` submits 3 tasks and verifies all complete

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging
