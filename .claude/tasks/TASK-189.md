---
id: TASK-189
title: "Orchestrator hardening: startup recovery, path normalization, queue cap, retry limit"
status: todo
priority: high
role: backend
dependencies: none
estimated_effort: M 1h
---

## Goal

Harden `OrchestratorService` with four production-safety features: re-queue tasks stuck in `PROCESSING` at startup (crash recovery), normalize `ProjectPath` to an absolute clean path before storage, cap the in-memory work queue size, and limit per-task retry attempts.

## Context

- Core service: `internal/core/services/orchestrator.go`
- Domain: `internal/core/domain/task.go` — `StatusProcessing`, `StatusFailed`
- Port: `internal/core/ports/ports.go` — `TaskRepository` (need `GetByStatus` or equivalent)
- `NewOrchestrator()` starts the background worker but does **not** scan for stuck tasks
- `Stop()` is already idempotent (`stopOnce.Do`) — no change needed there
- `OrchestratorService.workCh` channel has capacity 1 — queue back-pressure relies on the channel; a configurable integer cap on enqueued (QUEUED) tasks is missing
- No retry counter exists on `domain.Task` or in the service loop

## Scope

### Files to modify
- `internal/core/domain/task.go` — add `RetryCount int` field
- `internal/core/services/orchestrator.go` — add startup recovery, path normalization, queue cap check, retry logic (max 3 retries before StatusFailed)

### Files to create
- `internal/core/services/orchestrator_hardening_test.go` — unit tests for each new behavior

## Implementation

1. **Startup recovery** — In `NewOrchestrator()`, after constructing the service, call `repo.GetAll()` (or a status-filtered variant), find any tasks with `Status == StatusProcessing`, and call `repo.UpdateStatus(id, StatusQueued)` on each. Then signal `workCh`.
2. **Path normalization** — In `SubmitTask()` and `CreateDraft()` / `PromoteTask()`, apply `filepath.Clean(filepath.Abs(task.ProjectPath))` before saving.
3. **Queue cap** — In `SubmitTask()`, count tasks with `StatusQueued` from the repo; if count >= configurable cap (default 50), return `ErrQueueFull`. Expose `WithQueueCap(n int)` option.
4. **Retry limit** — In `runWorker()`, on LLM error: if `task.RetryCount < MaxRetries (3)`, increment `RetryCount`, set status back to `StatusQueued`, re-signal `workCh`. On 3rd failure, set `StatusFailed`.

## Acceptance Criteria
- [ ] `go vet ./...` passes
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./internal/core/services/...` passes
- [ ] A task whose status is `PROCESSING` at startup is re-queued on the next `NewOrchestrator()` call (test proves it)
- [ ] `SubmitTask` with a relative `ProjectPath` stores an absolute cleaned path
- [ ] `SubmitTask` returns an error (not panic) when the queue is at cap
- [ ] A failing task is retried up to 3 times before moving to `StatusFailed`
