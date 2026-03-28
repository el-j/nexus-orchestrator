# TASK-374: Add provider pre-flight soft-check to PromoteTask

**Plan:** PLAN-055 | **Wave:** 1 | **Status:** done | **Role:** backend

## Problem

`PromoteTask` transitions DRAFT → QUEUED without checking if any provider can handle the task. The worker then executes immediately and sets NO_PROVIDER. User experience: appears to succeed but silently fails 1 second later with no explanation in the MCP response.

## Strategy

**Soft-check** (not hard block): pre-flight checks provider availability, but still promotes. Returns a `warning` field in the response so callers know to expect potential NO_PROVIDER. This preserves the workflow where a user queues tasks ahead of spinning up a provider.

## Files to Edit

- `internal/core/services/task_service.go` — `PromoteTask()` return type → `PromoteResult`
- `internal/core/ports/ports.go` — Add `PromoteResult` struct to `Orchestrator` interface
- `internal/adapters/inbound/mcp/tools.go` — `toolPromoteTask()` to include warning in response
- `internal/adapters/inbound/httpapi/handlers_tasks.go` — `handlePromoteTask()` response body
- `internal/core/services/orchestrator_test.go` — add test for promote-with-no-provider warning

## Implementation

### 1. Add PromoteResult to ports.go:

```go
// PromoteResult is returned by PromoteTask to indicate success and any soft warnings.
type PromoteResult struct {
    Promoted bool   `json:"promoted"`
    Warning  string `json:"warning,omitempty"` // non-empty if no provider available
}
```

### 2. Update Orchestrator interface:

```go
type Orchestrator interface {
    // ...
    PromoteTask(id string) (PromoteResult, error) // was: error
    // ...
}
```

### 3. Update PromoteTask in task_service.go:

```go
func (o *OrchestratorService) PromoteTask(id string) (ports.PromoteResult, error) {
    task, err := o.repo.GetByID(id)
    if err != nil {
        return ports.PromoteResult{}, fmt.Errorf("orchestrator: promote task: %w", err)
    }
    if task.Status != domain.StatusDraft && task.Status != domain.StatusBacklog {
        return ports.PromoteResult{}, fmt.Errorf("orchestrator: promote task: cannot promote task with status %s", task.Status)
    }
    if err := o.validateQueueAdmission(task); err != nil {
        return ports.PromoteResult{}, fmt.Errorf("orchestrator: promote task: %w", err)
    }
    // Soft pre-flight: check provider availability
    var warning string
    if _, provErr := o.selectProviderForTaskDryRun(task); provErr != nil {
        warning = fmt.Sprintf("no active provider available for this task (%s); it will dispatch when a provider comes online or you retry", provErr.Error())
    }
    ok, err := o.repo.UpdateStatusIfCurrent(id, task.Status, domain.StatusQueued)
    if err != nil {
        return ports.PromoteResult{}, fmt.Errorf("orchestrator: promote task: %w", err)
    }
    if !ok {
        return ports.PromoteResult{}, fmt.Errorf("orchestrator: promote task: task state changed during promotion")
    }
    o.signalWorker()
    o.emit(task.ID, domain.StatusQueued)
    return ports.PromoteResult{Promoted: true, Warning: warning}, nil
}
```

### 4. Add selectProviderForTaskDryRun (no side effects — no status update):

```go
// selectProviderForTaskDryRun checks if a provider can handle the task without mutating state.
func (o *OrchestratorService) selectProviderForTaskDryRun(task domain.Task) (ports.LLMClient, error) {
    if task.ProviderName != "" {
        if client, ok := o.discovery.GetClientByName(task.ProviderName); ok {
            return client, nil
        }
        return nil, fmt.Errorf("provider %q not found or not active", task.ProviderName)
    }
    return o.discovery.FindForModel(task.ModelID, task.ProviderHint)
}
```

### 5. Update MCP toolPromoteTask:

Returns `{ "promoted": true, "warning": "..." }` so MCP callers see the warning inline.

### 6. Update HTTP handlePromoteTask:

Returns JSON body `{ "promoted": true, "warning": "..." }` instead of empty 200.

## Verification

- `CGO_ENABLED=1 go test -race ./internal/core/services/... -run TestPromote`
- Promote a DRAFT with no active providers → returns `{promoted: true, warning: "no active provider..."}`, status becomes QUEUED
- Promote a DRAFT with LM Studio active → returns `{promoted: true}` (no warning)
- HTTP `POST /api/tasks/{id}/promote` → 200 JSON body with `warning` field when no provider

## Status

todo
