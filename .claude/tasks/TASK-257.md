---
id: TASK-257
title: "QA: Unit + integration tests for task-session tracking pipeline"
role: qa
planId: PLAN-038
status: todo
dependencies: [TASK-253, TASK-254, TASK-255]
createdAt: 2026-03-13T16:00:00.000Z
---

## Context
The new claim/status pipeline needs comprehensive test coverage: service-level unit tests, repo tests, and adapter tests.

## Files to Read
- `internal/core/services/orchestrator_test.go`
- `internal/adapters/outbound/repo_sqlite/repo_test.go`
- `internal/adapters/outbound/repo_sqlite/ai_session_repo_test.go`
- `internal/adapters/inbound/mcp/server_test.go` (if exists)
- `internal/adapters/inbound/httpapi/server_test.go` (if exists)

## Implementation Steps
1. Add service tests in `orchestrator_test.go`:
   - `TestClaimTask_Success` — claim a QUEUED task, verify PROCESSING + AISessionID set
   - `TestClaimTask_NotQueued` — attempt claim on PROCESSING task, expect error
   - `TestClaimTask_InvalidSession` — non-existent session, expect error
   - `TestClaimTask_DisconnectedSession` — disconnected session, expect error
   - `TestUpdateTaskStatus_Complete` — PROCESSING→COMPLETED, verify logs saved
   - `TestUpdateTaskStatus_Fail` — PROCESSING→FAILED
   - `TestUpdateTaskStatus_WrongOwner` — mismatch sessionID, expect error
   - `TestUpdateTaskStatus_InvalidTransition` — QUEUED→COMPLETED, expect error

2. Add repo tests for `GetTasksBySessionID` and `AppendRoutedTaskID`.

3. Add HTTP handler tests if `httpapi/server_test.go` exists, or create integration test.

4. Run full test suite: `CGO_ENABLED=1 go test -race -count=1 ./...`

## Acceptance Criteria
- [ ] All 8+ test cases pass
- [ ] No data races under `-race` flag
- [ ] Repo tests verify persistence round-trip
- [ ] Full `go test -race ./...` green
