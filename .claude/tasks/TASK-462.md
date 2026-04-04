---
id: TASK-462
plan: PLAN-061
status: done
wave: 2
priority: 2
---

# TASK-462: Fix ActivityService.Stop() — add WaitGroup for goroutine cleanup

**Problem:** `internal/core/services/activity_service.go` `Stop()` closes `stopCh` and returns immediately without waiting for the background goroutines to finish. `pollLoop` and `purgeLoop` continue executing after `Stop()` returns, meaning they may access repositories, timers, and channels that the caller has already torn down. Under the race detector this is a data race; in production it manifests as use-after-free-style panics during shutdown.

**Fix:** Add a `sync.WaitGroup` to `ActivityService`. Increment it before each goroutine is spawned in `Start()`. Decrement it (`wg.Done()`) as the final statement in each goroutine. Call `wg.Wait()` in `Stop()` after closing `stopCh` so `Stop()` blocks until all goroutines have exited.

**Files:**

- `internal/core/services/activity_service.go`
- `internal/core/services/activity_service_test.go` (add shutdown test if not present)

## Checklist

- [x] Add `wg sync.WaitGroup` field to the `ActivityService` struct
- [x] In `Start()`, call `s.wg.Add(N)` (where N = number of goroutines launched) immediately before the `go` statements
- [x] Add `defer s.wg.Done()` as the first statement in `pollLoop` and `purgeLoop` (or each anonymous goroutine)
- [x] In `Stop()`, after `close(s.stopCh)`, add `s.wg.Wait()` before returning
- [x] Ensure `Stop()` is idempotent (guard against closing an already-closed channel using a `sync.Once` or a `stopped` flag if not already present)
- [x] Add or extend a test in `activity_service_test.go` that calls `Start()` followed by `Stop()` and asserts it returns without blocking indefinitely (use a timeout via `t.Deadline()` or `time.AfterFunc`)
- [x] Run `CGO_ENABLED=1 go test -race ./internal/core/services/...` and confirm zero data race reports

## Acceptance Criteria

- `Stop()` does not return until both `pollLoop` and `purgeLoop` goroutines have fully exited
- Race detector reports zero violations in `activity_service` tests
- Existing service tests continue to pass
