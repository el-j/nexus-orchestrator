---
id: TASK-613
planId: PLAN-075
title: 'Fix ActivityService broadcaster race, runTaskWatchdog active-run & log persistence bugs, UpdateTaskStatus brain outcome recording, and holdUntilReset timer cleanup'
role: coder
wave: 2
status: done
createdAt: 2026-10-19
---

# TASK-613 — Fix ActivityService broadcaster race, runTaskWatchdog active-run & log persistence bugs, UpdateTaskStatus brain outcome recording, and holdUntilReset timer cleanup

1. **`internal/core/services/activity_service.go`**:
   - Guard `s.broadcaster` reads and writes (`SetBroadcaster` and `poll`) with a mutex to eliminate the startup race between `rt.Activity.Start()` and `httpapi.StartServerWith`.
2. **`internal/core/services/orchestrator_workers.go` (`runTaskWatchdog`)**:
   - Skip tasks that are actively executing in `o.runs` (in-process worker runs have their own context timeouts), and call `o.repo.UpdateStatus(o.dbCtx(), t.ID, domain.StatusFailed, t.Logs)` so the watchdog timeout log message is actually persisted in SQLite.
3. **`internal/core/services/task_service.go` (`UpdateTaskStatus`)**:
   - Call `o.emit(taskID, status)` instead of `o.dispatch` directly when an external AI session completes or fails a claimed task, so `o.recordOutcome` stores the outcome in the project brain when `policy.Remember` is enabled.
4. **`internal/core/services/waiting.go` (`holdUntilReset`)**:
   - Track the `time.AfterFunc` wake timer on `OrchestratorService` and stop it in `Stop()` before closing the repository.

Done when: the failing test came first, `make test` and `make nice` pass, docs and changelog updated.
