---
id: TASK-075
title: "Service: Command-aware validation in orchestrator"
role: backend
planId: PLAN-009
status: todo
dependencies: [TASK-074]
createdAt: 2026-03-10T15:00:00.000Z
---

## Context
The orchestrator service must validate CommandType on task submission. If a task has `CommandExecute` but there are no completed "plan" tasks for the same project path, the orchestrator should reject the task with `ErrNoPlan`. This implements the "smart enough to request planning first" behavior.

## Files to Read
- `internal/core/domain/task.go` (after TASK-074 changes)
- `internal/core/services/orchestrator.go`
- `internal/core/ports/ports.go`
- `internal/adapters/outbound/repo_sqlite/repo.go`

## Implementation Steps
1. In `internal/core/ports/ports.go`, add `GetByProjectPath(projectPath string) ([]domain.Task, error)` to the `TaskRepository` interface — returns all tasks for a project.
2. In `internal/adapters/outbound/repo_sqlite/repo.go`, implement `GetByProjectPath`:
   - `SELECT ... FROM tasks WHERE project_path = ? ORDER BY created_at DESC`
3. In `internal/core/services/orchestrator.go`, in `SubmitTask`:
   - After existing validations, check `task.Command.IsValid()` — if not, return error.
   - If `task.Command` is empty, set it to `domain.CommandAuto`.
   - If `task.Command == domain.CommandExecute`, query `repo.GetByProjectPath(task.ProjectPath)` and check if any completed task has `Command == domain.CommandPlan`. If none found, return `fmt.Errorf("orchestrator: submit task: %w", domain.ErrNoPlan)`.
4. Ensure backward compatibility: tasks with `CommandAuto` or `CommandPlan` always proceed normally.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] Submit with `CommandExecute` and no prior plan tasks returns `ErrNoPlan`
- [ ] Submit with `CommandPlan` always succeeds (no plan check needed)
- [ ] Submit with `CommandAuto` or empty always succeeds
- [ ] Submit with invalid command returns validation error

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/`
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
