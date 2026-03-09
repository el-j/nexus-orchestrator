---
id: TASK-024
title: QA — orchestrator hardening tests + GUI build smoke test
role: qa
planId: PLAN-002
status: todo
dependencies: [TASK-013, TASK-014, TASK-019, TASK-020, TASK-021, TASK-022]
createdAt: 2026-03-09T12:00:00.000Z
---

## Context

TASK-013 and TASK-014 harden the orchestrator with startup recovery, Stop() idempotency, retry limits, and queue caps. These changes need comprehensive table-driven tests that cover edge cases not already in `orchestrator_test.go`. The frontend scaffold also needs a CI smoke test that verifies `npm run build` succeeds and produces the expected output files.

## Files to Read

- `internal/core/services/orchestrator_test.go` — existing test structure + stubs
- `internal/core/services/orchestrator.go` — after TASK-013 + TASK-014 are applied
- `internal/adapters/outbound/repo_sqlite/repo.go` — for integration tests checking DB state
- `internal/adapters/outbound/repo_sqlite/session_repo.go` — session repo for isolation tests
- `frontend/package.json` — confirm build script name for smoke test documentation

## Implementation Steps

1. **New test file `internal/core/services/orchestrator_hardening_test.go`** (or extend `orchestrator_test.go` — check file size first):

   **TestStartupRecovery** (integration — uses in-memory stubs):
   - Seed memRepo with 1 QUEUED + 1 PROCESSING task.
   - Create new OrchestratorService — verify both tasks are in queue with status QUEUED.
   - Verify no task remains in PROCESSING state post-startup.

   **TestStopIdempotent**:
   - Create OrchestratorService, call Start(), call Stop() twice.
   - Assert no panic (recover with t.Fatal if panic).

   **TestRetryLimitExceeded** (table-driven):
   - LLM mock: always returns error `"LLM unavailable"`.
   - Submit one task, let worker run for N ticks (N > maxRetries).
   - Assert task eventually reaches `StatusFailed`.
   - Assert `retryCount == maxRetries` in repo.

   **TestQueueAtCapacity**:
   - Create OrchestratorService with `maxQueueSize = 3`.
   - Submit 3 tasks — all succeed.
   - Submit 4th task — assert error contains "queue at capacity".

   **TestProjectPathNormalization**:
   - Submit task with path `"/project/foo/"` (trailing slash).
   - Assert stored `task.ProjectPath == "/project/foo"`.
   - Submit second task with `"/project/foo"` (no slash).
   - Assert both tasks resolve to the same session (same `projectPath` key).

   **TestCancelTaskConsistency** (regression for B1):
   - Submit task, cancel it.
   - Assert task status in repo == StatusFailed/Cancelled AND task not in memory queue.
   - Simulate DB error during cancel — assert task NOT removed from memory queue (rollback).

   **TestUpdateStatusErrors** (regression for D1):
   - Mock repo that returns error from `UpdateStatus`.
   - Assert task is re-enqueued (not silently dropped) on UpdateStatus failure.

2. **Frontend smoke test documentation in `TASK-024.md`** (this file):
   The CI smoke test for the frontend is NOT a Go test — it is documented in TASK-025 as a build matrix step:
   ```sh
   cd frontend && npm ci && npm run build
   # Verify dist/index.html exists
   test -f frontend/dist/index.html
   # Verify JS bundle exists
   ls frontend/dist/assets/*.js | head -1
   ```

3. **HTTP API integration tests** (add to `httpapi/` or new file `httpapi/server_hardening_test.go`):

   **TestListTasksEndpoint**: `GET /api/tasks?status=completed` returns correct subset.
   **TestListTasksAllStatuses**: `GET /api/tasks` (no filter) returns all tasks.
   **TestGetSessionEndpoint**: `GET /api/sessions/<path>` returns messages JSON.
   **TestDeleteSessionEndpoint**: `DELETE /api/sessions/<path>` returns 204, subsequent GET returns `[]`.
   **TestSubmitTaskWithSourceFields**: `POST /api/tasks` with `sourceProjectPath` + `sourceTaskId` — verify fields persisted.

4. **Ensure all stubs in `orchestrator_test.go` remain race-safe** after TASK-013/014 changes add new methods to `TaskRepository` port (`GetPending`, `GetAll`, `GetBySourceProject`). All stubs must implement the full interface.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 with all new tests passing
- [ ] `TestStartupRecovery` passes — PROCESSING tasks reset to QUEUED
- [ ] `TestStopIdempotent` passes — no panic on double Stop()
- [ ] `TestRetryLimitExceeded` passes — task reaches StatusFailed after maxRetries
- [ ] `TestQueueAtCapacity` passes — 4th submit returns error
- [ ] `TestProjectPathNormalization` passes — trailing slash normalized
- [ ] All in-memory stubs implement updated `TaskRepository` port interface

## Anti-patterns to Avoid

- NEVER use `time.Sleep` in tests — use a mock ticker or synchronous worker invocation
- NEVER share state between parallel tests without mutex — each test must have its own repo/service instance
- NEVER test implementation details — test observable behaviour (status changes, errors returned)
- NEVER write tests that depend on specific retry backoff durations — mock the clock or the LLM
