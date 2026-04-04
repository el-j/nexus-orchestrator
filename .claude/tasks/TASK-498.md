---
id: TASK-498
plan: PLAN-064
status: done
wave: 1
priority: 1
---

# TASK-498: Add stale-task watchdog test

## Description

The PROCESSING task watchdog in `OrchestratorService` calls `GetStaleProcessing` and re-queues or fails tasks that have been stuck too long. The `memRepo` stub used in existing orchestrator tests always returns nil for `GetStaleProcessing`, meaning this entire recovery path has never been exercised. A single missed watchdog cycle could leave tasks wedged in PROCESSING forever in production.

## Checklist

- [ ] Create `internal/core/services/orchestrator_hardening_test.go` (or add to existing hardening file if present)
- [ ] Implement a test `memRepo` method for `GetStaleProcessing` that returns a pre-seeded PROCESSING task with `UpdatedAt` set to `time.Now().Add(-staleness_threshold - 1s)`
- [ ] If staleness threshold is a hardcoded constant, expose it as a package-level `var StaleThreshold` or accept it as an option so tests can inject a short timeout (e.g. 100ms)
- [ ] Test: create one PROCESSING task older than threshold; start orchestrator with short tick interval; wait 2 tick cycles; assert task status is no longer PROCESSING (either QUEUED retry or FAILED depending on implementation)
- [ ] Test: PROCESSING task younger than threshold is NOT re-queued during same window
- [ ] Confirm no data race under `-race` flag
- [ ] If threshold exposure requires a small production code change, make it an unexported `var` with `//nolint:gochecknoglobals` comment to keep impact minimal

## Files

- `internal/core/services/orchestrator_hardening_test.go` (create)
- `internal/core/services/orchestrator.go` (may require minor threshold exposure)
- `internal/core/ports/ports.go` (reference for `GetStaleProcessing` signature)

## Acceptance Criteria

- 2 test cases: stale task re-queued/failed; fresh task unaffected
- No mock `GetStaleProcessing` returns nil; tests use non-trivial fixture data
- `CGO_ENABLED=1 go test -race ./internal/core/services/...` exits 0
