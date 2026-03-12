---
id: TASK-173
title: Ports — extend Orchestrator and TaskRepository for backlog lifecycle
role: architecture
planId: PLAN-024
status: todo
dependencies: [TASK-172]
createdAt: 2026-03-11T22:00:00.000Z
---

## Context
The Orchestrator port needs methods to create drafts, list backlogs per project, promote items to queue, and update task fields. TaskRepository needs a filtered query for backlog items and a generic update method.

## Files to Read
- `internal/core/ports/ports.go` — all port interfaces
- `internal/core/domain/task.go` — Task struct with new fields from TASK-172

## Implementation Steps

1. Extend `TaskRepository` port with:
   ```go
   // GetByProjectPathAndStatus returns tasks for a project filtered by status(es).
   GetByProjectPathAndStatus(projectPath string, statuses ...domain.TaskStatus) ([]domain.Task, error)
   // Update persists changes to an existing task's mutable fields (status, priority, providerName, tags, instruction).
   Update(t domain.Task) error
   ```

2. Extend `Orchestrator` port with:
   ```go
   // CreateDraft creates a task with StatusDraft. It does NOT enter the queue.
   CreateDraft(task domain.Task) (string, error)
   // GetBacklog returns DRAFT and BACKLOG tasks for the given project, ordered by priority then creation time.
   GetBacklog(projectPath string) ([]domain.Task, error)
   // PromoteTask transitions a DRAFT or BACKLOG task to QUEUED and enqueues it.
   PromoteTask(id string) error
   // UpdateTask updates mutable fields on an existing task (instruction, priority, providerName, tags, status).
   UpdateTask(id string, updates domain.Task) (domain.Task, error)
   ```

3. Update all `Orchestrator` implementors (test mocks, CLI remote client) with stub implementations so the build doesn't break.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `TaskRepository` has `GetByProjectPathAndStatus` and `Update` methods
- [ ] `Orchestrator` has `CreateDraft`, `GetBacklog`, `PromoteTask`, `UpdateTask` methods
- [ ] All existing methods unchanged (additive only)

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER break existing Orchestrator method signatures
