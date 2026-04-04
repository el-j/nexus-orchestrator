---
id: TASK-233
title: "Backend: Add GetAllTasks to port/service/app/HTTP/MCP + fix GetBacklog empty-project"
role: backend
planId: PLAN-033
status: done
dependencies: []
createdAt: 2026-03-13T01:10:00.000Z
completedAt: 2026-03-13T01:50:00.000Z
---

## Context
BacklogView and HistoryView cannot display data because the only task-fetching method (`GetQueue`) returns only QUEUED/PROCESSING tasks. A `GetAllTasks()` method is needed across the full stack. Also `GetBacklog("")` should return all draft/backlog tasks across all projects.

## Files to Read
- `internal/core/ports/ports.go` (Orchestrator + TaskRepository interfaces)
- `internal/core/services/orchestrator.go` (service implementation)
- `internal/adapters/outbound/repo_sqlite/repo.go` (SQLite queries + GetByProjectPath)
- `internal/adapters/inbound/httpapi/server.go` (HTTP routes)
- `internal/adapters/inbound/mcp/server.go` (MCP tool definitions)
- `app.go` (Wails bindings)

## Implementation Steps

1. **Port — TaskRepository**: No change needed. `GetByProjectPath` exists but isn't useful for "all projects". Add a `GetAll() ([]domain.Task, error)` method to `TaskRepository` interface.

2. **SQLite — repo.go**: Implement `GetAll()`:
   ```go
   func (r *Repository) GetAll() ([]domain.Task, error) {
       rows, err := r.db.Query(`SELECT ... FROM tasks ORDER BY created_at DESC`)
       // ...
       tasks := []domain.Task{}
       // scan loop
       return tasks, rows.Err()
   }
   ```

3. **Port — Orchestrator**: Add `GetAllTasks() ([]domain.Task, error)` to the `Orchestrator` interface.

4. **Service — orchestrator.go**: Implement `GetAllTasks()`:
   ```go
   func (o *OrchestratorService) GetAllTasks() ([]domain.Task, error) {
       return o.repo.GetAll()
   }
   ```

5. **Service — orchestrator.go**: Fix `GetBacklog("")` — when projectPath is empty, return ALL draft/backlog tasks:
   ```go
   func (o *OrchestratorService) GetBacklog(projectPath string) ([]domain.Task, error) {
       if projectPath == "" {
           tasks, err := o.repo.GetAll()
           if err != nil { return nil, ... }
           var backlog []domain.Task
           for _, t := range tasks {
               if t.Status == domain.StatusDraft || t.Status == domain.StatusBacklog {
                   backlog = append(backlog, t)
               }
           }
           if backlog == nil { backlog = []domain.Task{} }
           return backlog, nil
       }
       // existing path-specific logic
   }
   ```

6. **Wails — app.go**: Add `GetAllTasks()`:
   ```go
   func (a *App) GetAllTasks() ([]domain.Task, error) {
       return a.orchestrator.GetAllTasks()
   }
   ```

7. **HTTP — server.go**: Add `GET /api/tasks/all` route:
   ```go
   r.Get("/api/tasks/all", s.handleGetAllTasks)
   ```
   Handler calls `s.orch.GetAllTasks()`.

8. **MCP — server.go**: Add `get_all_tasks` tool with no params, returns all tasks.

9. **Update memRepo test stub** (if one exists) to implement `GetAll()`.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `GetAllTasks()` returns tasks of ALL statuses
- [ ] `GetBacklog("")` returns all DRAFT+BACKLOG tasks across all projects
- [ ] `GET /api/tasks/all` returns JSON array of all tasks
- [ ] MCP `get_all_tasks` tool registered in tools/list

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/`
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
