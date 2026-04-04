# TASK-373: Fix CancelTask — allow NO_PROVIDER → CANCELLED transition

**Plan:** PLAN-055 | **Wave:** 1 | **Status:** done | **Role:** backend

## Problem

`CancelTask` in `task_service.go` only accepts tasks in `QUEUED` status. When the worker dispatches a task and finds no active LLM provider, it sets `StatusNoProvider`. The task is now permanently stuck — it cannot be cancelled, promoted, or retried. Confirmed via live testing: task `d7f4180f` in NO_PROVIDER cannot be cancelled.

## Files to Edit

- `internal/core/services/task_service.go` — `CancelTask()` function (lines 70-85)
- `internal/core/services/orchestrator_test.go` or `orchestrator_hardening_test.go` — add test

## Implementation

### Fix CancelTask to accept QUEUED or NO_PROVIDER:

```go
// task_service.go — CancelTask
func (o *OrchestratorService) CancelTask(id string) error {
    // First try to cancel from QUEUED state (fast path)
    ok, err := o.repo.UpdateStatusIfCurrent(id, domain.StatusQueued, domain.StatusCancelled)
    if err != nil {
        return fmt.Errorf("orchestrator: cancel task: %w", err)
    }
    if ok {
        o.emit(id, domain.StatusCancelled)
        return nil
    }
    // Also allow cancelling from NO_PROVIDER state (was stuck waiting for provider)
    ok, err = o.repo.UpdateStatusIfCurrent(id, domain.StatusNoProvider, domain.StatusCancelled)
    if err != nil {
        return fmt.Errorf("orchestrator: cancel task: %w", err)
    }
    if ok {
        o.emit(id, domain.StatusCancelled)
        return nil
    }
    task, err := o.repo.GetByID(id)
    if err != nil {
        return fmt.Errorf("orchestrator: cancel task: %w", err)
    }
    return fmt.Errorf("orchestrator: cancel task: cannot cancel task with status %s", task.Status)
}
```

### Add test:

```go
func TestCancelTask_NoProvider(t *testing.T) {
    // Create a task, set its status to NO_PROVIDER, verify cancel succeeds
    svc := newTestOrchestrator(t)
    id, _ := svc.CreateDraft(domain.Task{Instruction: "x", ProjectPath: "/p"})
    // Simulate worker setting NO_PROVIDER
    repo := ... // access repo
    repo.UpdateStatus(id, domain.StatusNoProvider)

    err := svc.CancelTask(id)
    require.NoError(t, err)

    task, _ := svc.GetTask(id)
    assert.Equal(t, domain.StatusCancelled, task.Status)
}
```

## Verification

- `CGO_ENABLED=1 go test -race ./internal/core/services/... -run TestCancel`
- Confirm: `CancelTask` on a NO_PROVIDER task → returns nil, status becomes CANCELLED
- Confirm: `CancelTask` on a PROCESSING task → still returns error (PROCESSING should NOT be cancellable mid-run)

## Status

todo
