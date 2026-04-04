---
id: TASK-253
title: "Backend: OrchestratorService ClaimTask + UpdateTaskStatus implementation"
role: backend
planId: PLAN-038
status: todo
dependencies: [TASK-251, TASK-252]
createdAt: 2026-03-13T16:00:00.000Z
---

## Context
External AI agents need service-level methods to claim queued tasks and report their completion status. This task implements the core business logic with proper state machine guards and populates `RoutedTaskIDs` on the owning session.

## Files to Read
- `internal/core/services/orchestrator.go` (RegisterAISession, processTask patterns)
- `internal/core/ports/ports.go` (updated interface from TASK-251)

## Implementation Steps
1. Implement `ClaimTask(ctx, taskID, sessionID) (Task, error)`:
   - Verify session exists and is active via `aiSessionRepo.GetAISessionByID()`
   - Use `repo.UpdateStatusIfCurrent(taskID, StatusQueued, StatusProcessing)` to atomically claim
   - Set `task.AISessionID = sessionID` and persist via `repo.Update()`
   - Call `aiSessionRepo.AppendRoutedTaskID(ctx, sessionID, taskID)`
   - Broadcast `EventTaskProcessing`
   - Return updated task

2. Implement `UpdateTaskStatus(ctx, taskID, sessionID, status, logs) (Task, error)`:
   - Fetch task, verify `task.AISessionID == sessionID` (ownership check)
   - Only allow transitions: PROCESSING→COMPLETED, PROCESSING→FAILED
   - Update task status and logs via `repo.UpdateStatus()` and `repo.UpdateLogs()`
   - If COMPLETED, call `fileWriter.WriteCodeToFile()` if applicable
   - Broadcast appropriate event
   - Return updated task

3. Add `sync.Mutex` protection for the claim path to prevent double-claiming.

## Acceptance Criteria
- [ ] `ClaimTask` transitions QUEUED→PROCESSING atomically
- [ ] `ClaimTask` rejects non-QUEUED tasks with appropriate error
- [ ] `ClaimTask` rejects invalid/disconnected sessions
- [ ] `UpdateTaskStatus` enforces ownership (sessionID must match)
- [ ] `UpdateTaskStatus` only allows PROCESSING→COMPLETED or PROCESSING→FAILED
- [ ] `RoutedTaskIDs` is populated on the session after claim
- [ ] Events are broadcast on claim and status update
- [ ] No data races under `-race`
