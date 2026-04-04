---
id: TASK-497
plan: PLAN-064
status: done
wave: 3
priority: 1
---

# TASK-497: Add MCP claim_task -> update_task_status integration test

## Description

`internal/adapters/inbound/mcp/tools_test.go` uses a mock orchestrator (`toolHarnessOrch`) that returns zero-value `domain.Task` structs. This means no existing test can verify exclusive ownership semantics (only the claiming session may update a task) or that the QUEUED -> PROCESSING -> COMPLETED state machine transition is actually persisted. A real SQLite-backed integration test is required.

## Checklist

- [ ] Create or extend `internal/adapters/inbound/mcp/integration_test.go`
- [ ] Wire up a real SQLite repo + OrchestratorService (matching the pattern in `internal/e2e/` or existing `integration_test.go`)
- [ ] Test happy path: create task in QUEUED state; call `claim_task` via MCP tool handler; verify task status is PROCESSING in DB; call `update_task_status` COMPLETED with correct session_id; verify task status is COMPLETED in DB and `logs` field saved
- [ ] Test exclusive ownership: after first `claim_task`, second `claim_task` from different session returns error or `task not QUEUED`; task status remains PROCESSING (not double-claimed)
- [ ] Test ownership enforcement on update: call `update_task_status` with wrong session_id; verify it is rejected; task status unchanged
- [ ] Use `t.TempDir()` for ephemeral SQLite DB path; no shared DB between test runs
- [ ] Test runs under `CGO_ENABLED=1 go test -race ./internal/adapters/inbound/mcp/...`

## Files

- `internal/adapters/inbound/mcp/integration_test.go` (create or extend)
- `internal/adapters/inbound/mcp/tools.go` (reference)
- `internal/core/services/orchestrator.go` (reference)
- `internal/adapters/outbound/repo_sqlite/` (dependency)

## Acceptance Criteria

- 3 integration test cases (happy path, double-claim, wrong-session update)
- All assertions query the real SQLite DB, not a mock return value
- `CGO_ENABLED=1 go test -race ./internal/adapters/inbound/mcp/...` exits 0
