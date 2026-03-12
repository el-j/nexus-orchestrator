---
id: TASK-175
title: OrchestratorService — backlog lifecycle and explicit provider routing
role: backend
planId: PLAN-024
status: todo
dependencies: [TASK-173, TASK-174]
createdAt: 2026-03-11T22:00:00.000Z
---

## Context
The OrchestratorService needs real implementations for the new port methods: `CreateDraft`, `GetBacklog`, `PromoteTask`, `UpdateTask`. The `processNext` worker must also be updated to use `ProviderName` for explicit provider routing (short-circuit discovery when set).

## Files to Read
- `internal/core/services/orchestrator.go` — SubmitTask, processNext, worker loop
- `internal/core/services/discovery.go` — FindForModel, GetClientByName
- `internal/core/ports/ports.go` — updated Orchestrator interface
- `internal/core/domain/task.go` — StatusDraft, StatusBacklog, ProviderName

## Implementation Steps

1. Implement `CreateDraft(task domain.Task) (string, error)`:
   - Validate: Instruction not empty, ProjectPath not empty
   - Set `task.Status = domain.StatusDraft`, generate UUID, set timestamps
   - If `task.Priority == 0`, default to 2 (medium)
   - Save via `o.repo.Save(task)` — does NOT enqueue or wake worker
   - Broadcast SSE event `task_created` with status DRAFT
   - Return task ID

2. Implement `GetBacklog(projectPath string) ([]domain.Task, error)`:
   - Call `o.repo.GetByProjectPathAndStatus(projectPath, domain.StatusDraft, domain.StatusBacklog)`
   - Return sorted by priority ASC, then CreatedAt ASC

3. Implement `PromoteTask(id string) error`:
   - Fetch task by ID
   - Verify status is `StatusDraft` or `StatusBacklog`, else return error
   - Set `Status = StatusQueued`, `UpdatedAt = now`
   - Save via `o.repo.Update(task)`
   - Enqueue in `o.queue` and wake worker (same pattern as SubmitTask)
   - Broadcast SSE event `task_promoted`

4. Implement `UpdateTask(id string, updates domain.Task) (domain.Task, error)`:
   - Fetch existing task by ID
   - Apply non-zero/non-empty fields from `updates` to existing (merge pattern)
   - Cannot update status to PROCESSING/COMPLETED/FAILED (those are worker-managed)
   - Save via `o.repo.Update(task)`
   - Broadcast SSE event

5. **Explicit provider routing** — update `processNext()`:
   ```go
   var llm ports.LLMClient
   if task.ProviderName != "" {
       // Short-circuit: exact provider match
       client, ok := o.discovery.GetClientByName(task.ProviderName)
       if !ok || !client.Ping() {
           o.repo.UpdateStatus(task.ID, domain.StatusNoProvider)
           // ...
           return
       }
       llm = client
   } else {
       // Existing: FindForModel(modelID, providerHint)
       llm, err = o.discovery.FindForModel(task.ModelID, task.ProviderHint)
       // ...
   }
   ```

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `CreateDraft` creates a task with StatusDraft that never enters the queue
- [ ] `PromoteTask` transitions DRAFT/BACKLOG → QUEUED and enqueues
- [ ] `UpdateTask` merges fields without overwriting worker-managed states
- [ ] `processNext` uses `ProviderName` for direct routing when set
- [ ] Worker loop still ignores DRAFT/BACKLOG tasks (only processes QUEUED)

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/`
- NEVER skip `fmt.Errorf("orchestrator: operation: %w", err)` error wrapping
- NEVER allow PromoteTask to work on tasks already in QUEUED/PROCESSING/terminal states
