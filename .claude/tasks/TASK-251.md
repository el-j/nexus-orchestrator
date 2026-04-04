---
id: TASK-251
title: "Architecture: Domain model + port contract for task-session binding"
role: architecture
planId: PLAN-038
status: todo
dependencies: []
createdAt: 2026-03-13T16:00:00.000Z
---

## Context
Tasks currently have no reference to the AI session that claimed or is executing them. The `Orchestrator` port has no methods for external agents to claim tasks or report status. This task adds the domain-level field and the port contract methods that all subsequent tasks depend on.

## Files to Read
- `internal/core/domain/task.go`
- `internal/core/domain/ai_session.go`
- `internal/core/ports/ports.go`

## Implementation Steps
1. Add `AISessionID string` field to `domain.Task` struct with JSON tag `"aiSessionId,omitempty"`.
2. Add `ClaimTask(ctx context.Context, taskID string, sessionID string) (domain.Task, error)` to the `Orchestrator` interface in `ports.go`.
3. Add `UpdateTaskStatus(ctx context.Context, taskID string, sessionID string, status domain.TaskStatus, logs string) (domain.Task, error)` to the `Orchestrator` interface.
4. Add `GetTasksBySessionID(sessionID string) ([]domain.Task, error)` to the `TaskRepository` interface in `ports.go`.
5. Add `AppendRoutedTaskID(ctx context.Context, sessionID string, taskID string) error` to the `AISessionRepository` interface in `ports.go`.

## Acceptance Criteria
- [ ] `domain.Task` has `AISessionID` field
- [ ] `Orchestrator` interface has `ClaimTask` and `UpdateTaskStatus` methods
- [ ] `TaskRepository` interface has `GetTasksBySessionID` method
- [ ] `AISessionRepository` interface has `AppendRoutedTaskID` method
- [ ] `go vet ./...` passes (compile check on interface satisfaction will fail until Wave 2-3 implement them)
