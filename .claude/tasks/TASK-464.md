---
id: TASK-464
plan: PLAN-061
status: done
wave: 2
priority: 2
---

# TASK-464: Fix SetAgentScanner/SetDiscoveredAgentRepo — missing mutex

**Problem:** `internal/core/services/orchestrator.go` methods `SetAgentScanner()` and `SetDiscoveredAgentRepo()` write to struct fields without acquiring `o.mu`. Every other setter in `OrchestratorService` (e.g. `SetSessionRepo`, `SetRuntimeConfigRepo`) correctly acquires the mutex before writing. The two broken setters are called after construction during startup, and can race with the background worker goroutine that reads the same fields.

**Fix:** Add `o.mu.Lock()` / `defer o.mu.Unlock()` to both `SetAgentScanner()` and `SetDiscoveredAgentRepo()`, matching the pattern of the other setters.

**Files:**

- `internal/core/services/orchestrator.go`
- `internal/core/services/orchestrator_test.go` (verify race-free under `-race`)

## Checklist

- [x] Locate `SetAgentScanner()` in `orchestrator.go` and add `o.mu.Lock(); defer o.mu.Unlock()` before the field assignment
- [x] Locate `SetDiscoveredAgentRepo()` in `orchestrator.go` and add `o.mu.Lock(); defer o.mu.Unlock()` before the field assignment
- [x] Confirm no other setters are missing the lock (do a grep for all `func (o *OrchestratorService) Set` methods and verify each acquires `o.mu`)
- [x] Run `CGO_ENABLED=1 go test -race ./internal/core/services/...` and confirm zero data race reports
- [x] If an existing test exercises concurrent `Set*` + worker activity, confirm it still passes; otherwise add a short concurrent test that calls `SetAgentScanner` while the worker is running

## Acceptance Criteria

- Both `SetAgentScanner` and `SetDiscoveredAgentRepo` acquire `o.mu` before writing
- All `Set*` methods on `OrchestratorService` consistently use the mutex
- `CGO_ENABLED=1 go test -race ./internal/core/services/...` reports zero races
