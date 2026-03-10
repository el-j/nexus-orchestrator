---
id: TASK-028
title: OrchestratorService — invoke WritebackClient on task completion
role: backend
planId: PLAN-002
status: todo
dependencies: [TASK-027, TASK-013]
createdAt: 2026-03-09T13:00:00.000Z
---

## Context

After `processNext()` successfully generates output and writes files, if the task has `SourceProjectPath` set, the orchestrator must call `WritebackClient.WriteBack()` to update the source project's `.claude/orchestrator.json` and task file. This closes the push-to-nexus loop: external projects get automatic status updates without polling.

The WritebackClient is optional — when nil (or using NoopClient), the orchestrator behaves exactly as before TASK-026/027.

## Files to Read

- `internal/core/services/orchestrator.go` — full file (especially `processNext()`)
- `internal/core/ports/ports.go` — `WritebackClient` interface (added in TASK-027)
- `internal/core/domain/task.go` — Task struct with source fields (added in TASK-026)
- `internal/adapters/outbound/fs_writeback/writeback.go` — adapter to be injected

## Implementation Steps

1. **Add `writebackClient ports.WritebackClient` field** to `OrchestratorService`:
   ```go
   type OrchestratorService struct {
       mu              sync.Mutex
       queue           []domain.Task
       repo            ports.TaskRepository
       llm             ports.LLMClient
       fileWriter      ports.FileWriter
       sessionRepo     ports.SessionRepository
       writebackClient ports.WritebackClient   // optional; nil = no writeback
       stopCh          chan struct{}
       stopOnce        sync.Once
   }
   ```

2. **Update `NewOrchestrator` constructor** to accept `writebackClient ports.WritebackClient` (can be nil):
   ```go
   func NewOrchestrator(
       repo ports.TaskRepository,
       llm ports.LLMClient,
       fileWriter ports.FileWriter,
       sessionRepo ports.SessionRepository,
       writebackClient ports.WritebackClient,
   ) (*OrchestratorService, error) {
   ```
   If `writebackClient == nil`, the service skips writeback silently. Use `fs_writeback.NoopClient{}` in tests that don't need writeback.

3. **In `processNext()`, after successful task completion** (just before `return`):
   ```go
   // Writeback to source project if this task was submitted via push-to-nexus
   if task.SourceProjectPath != "" && o.writebackClient != nil {
       payload := ports.WritebackPayload{
           SourceProjectPath: task.SourceProjectPath,
           SourceTaskID:      task.SourceTaskID,
           SourcePlanID:      task.SourcePlanID,
           NexusTaskID:       task.ID,
           Status:            "completed",
           Output:            output,   // LLM output string
           CompletedAt:       time.Now().UTC(),
       }
       if err := o.writebackClient.WriteBack(ctx, payload); err != nil {
           // Log but do NOT fail the task — writeback is best-effort
           log.Printf("orchestrator: writeback for %s: %v", task.ID, err)
       }
   }
   ```

4. **On task failure** (after `repo.UpdateStatus(task.ID, StatusFailed)`), similarly call WriteBack with `Status: "failed"`. This ensures the source project knows about failures too.

5. **Update all entry points** (`main.go`, `cmd/nexus-daemon/main.go`) to pass the `fs_writeback.New()` adapter:
   ```go
   import "nexus-orchestrator/internal/adapters/outbound/fs_writeback"
   // ...
   wb := fs_writeback.New()
   orch, err := services.NewOrchestrator(repo, llmClient, fileWriter, sessionRepo, wb)
   ```

6. **Update all test stubs** in `orchestrator_test.go` to pass `nil` as writebackClient (or use `fs_writeback.NoopClient{}`).

7. **New test `TestWritebackInvokedOnCompletion`**:
   - Mock `WritebackClient` that records calls.
   - Submit task with `SourceProjectPath="/tmp/testproj"`, `SourceTaskID="TASK-099"`.
   - Let worker run once with a successful LLM mock.
   - Assert `WriteBack` was called once with correct payload (Status="completed", NexusTaskID matches task ID).

8. **New test `TestWritebackNotInvokedWithoutSourcePath`**:
   - Submit task without `SourceProjectPath` (empty string).
   - Assert `WriteBack` is never called.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `OrchestratorService` has `writebackClient` field
- [ ] `processNext()` calls WriteBack on both success and failure when SourceProjectPath is set
- [ ] WriteBack errors are logged but do NOT fail or re-queue the task
- [ ] `TestWritebackInvokedOnCompletion` passes
- [ ] `TestWritebackNotInvokedWithoutSourcePath` passes
- [ ] All entry points pass `fs_writeback.New()` to NewOrchestrator

## Anti-patterns to Avoid

- NEVER block the worker goroutine waiting for writeback — if WriteBack is slow, log and continue
- NEVER make writeback errors cause task failure — writeback is best-effort
- NEVER import `fs_writeback` adapter in `internal/core/services/` — inject via port
- NEVER call WriteBack before `repo.UpdateStatus(StatusCompleted)` — always persist final status first
