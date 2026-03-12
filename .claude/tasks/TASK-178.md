---
id: TASK-178
title: Wails bindings — backlog lifecycle methods
role: backend
planId: PLAN-024
status: todo
dependencies: [TASK-175]
createdAt: 2026-03-11T22:00:00.000Z
---

## Context
The desktop GUI communicates with the orchestrator via Wails bindings in `app.go`. The new backlog lifecycle methods need corresponding exported methods so Vue can call them.

## Files to Read
- `app.go` — existing Wails bindings
- `internal/core/ports/ports.go` — updated Orchestrator interface
- `internal/core/domain/task.go` — Task struct with new fields

## Implementation Steps

1. Add 4 new exported methods to the `App` struct in `app.go`:

   ```go
   func (a *App) CreateDraft(task domain.Task) (string, error) {
       return a.orchestrator.CreateDraft(task)
   }

   func (a *App) GetBacklog(projectPath string) ([]domain.Task, error) {
       return a.orchestrator.GetBacklog(projectPath)
   }

   func (a *App) PromoteTask(id string) error {
       return a.orchestrator.PromoteTask(id)
   }

   func (a *App) UpdateTask(id string, updates domain.Task) (domain.Task, error) {
       return a.orchestrator.UpdateTask(id, updates)
   }
   ```

2. Ensure all methods are exported (capitalized) so Wails auto-generates JS bindings.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] 4 new methods on App struct
- [ ] Methods proxy to orchestrator port without adding business logic

## Anti-patterns to Avoid
- NEVER add business logic in app.go — it's a thin proxy
- NEVER import adapters from app.go (only ports and domain)
