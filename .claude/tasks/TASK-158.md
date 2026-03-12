---
id: TASK-158
title: QA — integration tests for AI session lifecycle
role: qa
planId: PLAN-022
status: todo
dependencies: [TASK-157]
priority: medium
estimated_effort: M
createdAt: 2026-03-12T11:00:00.000Z
---

## Goal
Write integration tests that exercise the full AI session lifecycle (register → list → heartbeat update → deregister) through the real HTTP API and the SQLite-backed service layer, verifying correctness end-to-end.

## Context
The project already has `internal/core/services/integration_test.go` as a reference for integration-style tests that spin up a real in-memory SQLite DB + `OrchestratorService` and call it directly. The HTTP-level tests use `httptest.NewServer` in `httpapi/server_test.go`.

Write tests at **both** levels:
1. Service-level (in `internal/core/services/integration_test.go` or a new `ai_session_integration_test.go` in the same package)
2. HTTP-level (in `internal/adapters/inbound/httpapi/server_test.go` or a new `ai_session_http_test.go`)

## Scope

### Files to create
- `internal/core/services/ai_session_service_test.go` — service-level integration tests
- `internal/adapters/inbound/httpapi/ai_session_http_test.go` — HTTP-level tests

## Implementation Steps

### 1. Service-level tests (ai_session_service_test.go)
Use `package services_test`. Construct a real `*services.OrchestratorService` with:
- In-memory SQLite repo (`:memory:`)
- `repo_sqlite.NewAISessionRepo(r)`
- Injected via `SetAISessionRepo` (or constructor pattern from TASK-153)

Write table-driven test `TestAISessionLifecycle`:
```
registering a session → ListAISessions returns 1 active session
registering same externalId twice → returns the same session_id (idempotent)
deregistering → session status = disconnected in list
registering after deregister → status is active again
```

Write `TestAISessionBroadcast`:
- Wire a `*Hub` broadcaster
- Register a session via `orch.RegisterAISession`
- Verify `Hub.Broadcast` was called with `type = "ai_session_changed"`

### 2. HTTP-level tests (ai_session_http_test.go)
Use `package httpapi_test`. Wire `httptest.NewServer` with a real `*OrchestratorService` backed by `:memory:` SQLite (same pattern as existing `server_test.go`).

Table-driven tests:
```
POST /api/ai-sessions — missing agentName → 400
POST /api/ai-sessions — valid body with source=vscode → 201 + id in response
GET  /api/ai-sessions — returns the registered session
DELETE /api/ai-sessions/{id} — 204 No Content
GET  /api/ai-sessions — session status is now disconnected
DELETE /api/ai-sessions/nonexistent — 404
```

### 3. Race detection
All tests must pass with `-race`. Use `t.Parallel()` on independent sub-tests. Ensure no mutex is missed.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./internal/core/services/... ./internal/adapters/inbound/httpapi/...` exits 0
- [ ] `TestAISessionLifecycle` covers the 4 scenarios listed above
- [ ] `TestAISessionBroadcast` verifies SSE event emission
- [ ] HTTP integration table covers all 6 test cases listed above
- [ ] No flaky behaviour under `-race` (mutex coverage complete)

## Anti-patterns to Avoid
- NEVER use `time.Sleep` for synchronisation — use channels or `t.Cleanup` ordering
- NEVER test with a shared global DB — each test function gets its own `:memory:` DB
- NEVER mock the SQLite repo in service-level tests — use the real implementation
- NEVER omit `-race` in the test command — data race detection is mandatory per project conventions
