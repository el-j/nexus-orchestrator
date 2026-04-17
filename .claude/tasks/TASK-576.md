---
id: TASK-576
title: Replace time.Sleep synchronisation in Go tests with channel/sync primitives
role: qa
planId: PLAN-071
status: todo
dependencies: [TASK-569, TASK-570, TASK-571]
createdAt: 2026-04-15T00:00:00Z
---

## Context

Twenty-five or more `time.Sleep` calls across 7 test files make the test suite flaky under CI load. The worst is `orchestrator_hardening_test.go` with 700ms sleeps. These should use channels, `sync.WaitGroup`, or testable callback hooks to signal completion instead of waiting arbitrary wall-clock time.

## Files to Read

- `internal/core/services/orchestrator_hardening_test.go`
- `internal/core/services/orchestrator_test.go`
- `internal/core/services/integration_test.go`
- `internal/adapters/inbound/mcp/integration_test.go`
- `internal/adapters/inbound/httpapi/plan_scan_worker_test.go`
- `internal/adapters/outbound/repo_sqlite/knowledge_repo_test.go`
- `internal/core/services/orchestrator.go` (to understand the hooks available)

## Implementation Steps

For each test file, replace `time.Sleep` patterns:

1. `plan_scan_worker_test.go` — `StartPlanScanWorker` should accept an optional `onScan func()` callback (or use an atomic counter that `TriggerScan` increments); in tests, block on a channel `done` that receives after the desired scan count instead of sleeping
2. `orchestrator_hardening_test.go` and `orchestrator_test.go` — for state-transition tests, use `require.Eventually(t, condition, timeout, tick)` from `testify` instead of `time.Sleep + assertion`. This retries the assertion up to the timeout rather than blocking unconditionally.
3. `integration_test.go` and `mcp/integration_test.go` — replace SSE/connection sleeps with a helper that reads from the SSE stream until it receives the expected event or times out (use a channel + goroutine reader)
4. `knowledge_repo_test.go` — replace timestamp-ordering sleep with explicit time manipulation: insert records with `time.Now().Add(-1 * time.Second)` and `time.Now()` to control order without sleeping

For all changes:

- Keep `require.Eventually` timeouts generous (e.g., 2s) but intervals fast (10ms) so tests are fast in CI but not brittle under load
- Ensure all new test goroutines are cleaned up — use `t.Cleanup` or `defer`

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 with no data races
- [ ] Zero `time.Sleep` calls remain in the 6 test files listed above (except `knowledge_repo_test.go` which uses timestamp manipulation instead)
- [ ] All 7 test files still pass after the refactor
- [ ] New tests added in TASK-569/570/571 also use `require.Eventually` or channels — not `time.Sleep`

## Anti-patterns to Avoid

- NEVER replace `time.Sleep(100ms)` with `time.Sleep(50ms)` — that just makes it flaky faster
- NEVER use `time.Sleep` in newly written tests — set the pattern for zero-sleep tests
- NEVER leave goroutine leaks in tests — always use `t.Cleanup` for teardown
