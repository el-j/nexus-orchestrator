---
id: TASK-176
title: HTTP API — backlog CRUD, promote, and update endpoints
role: api
planId: PLAN-024
status: todo
dependencies: [TASK-175]
createdAt: 2026-03-11T22:00:00.000Z
---

## Context
The GUI and CLI need HTTP endpoints to create drafts, list per-project backlogs, promote items, and update task fields. These consume the new Orchestrator port methods.

## Files to Read
- `internal/adapters/inbound/httpapi/server.go` — existing chi router, task handlers
- `internal/core/ports/ports.go` — updated Orchestrator interface
- `internal/core/domain/task.go` — new statuses and fields

## Implementation Steps

1. Add routes to the chi router:
   ```go
   r.Post("/api/tasks/draft", s.handleCreateDraft)
   r.Get("/api/tasks/backlog/{projectPath}", s.handleGetBacklog)
   r.Post("/api/tasks/{id}/promote", s.handlePromoteTask)
   r.Put("/api/tasks/{id}", s.handleUpdateTask)
   ```

2. `handleCreateDraft`:
   - Decode `domain.Task` from body (same as handleCreateTask)
   - Call `s.orch.CreateDraft(task)`
   - Return `201 Created` with `{task_id, status: "DRAFT"}`

3. `handleGetBacklog`:
   - Extract `{projectPath}` from URL (URL-decode it)
   - Call `s.orch.GetBacklog(projectPath)`
   - Return `200 OK` with `[]Task` JSON

4. `handlePromoteTask`:
   - Extract `{id}` from URL
   - Call `s.orch.PromoteTask(id)`
   - Return `204 No Content` on success
   - Return `404` for ErrNotFound, `400` for invalid state transition

5. `handleUpdateTask`:
   - Extract `{id}` from URL
   - Decode partial `domain.Task` from body
   - Call `s.orch.UpdateTask(id, updates)`
   - Return `200 OK` with updated task JSON
   - Return `404` for ErrNotFound

6. Extend SSE event types to include `task_promoted` events.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `POST /api/tasks/draft` creates a DRAFT task (201)
- [ ] `GET /api/tasks/backlog/{projectPath}` returns filtered backlog (200)
- [ ] `POST /api/tasks/{id}/promote` transitions DRAFT/BACKLOG → QUEUED (204)
- [ ] `PUT /api/tasks/{id}` updates mutable fields (200)
- [ ] Existing `POST /api/tasks` is unchanged (additive only)

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER skip `fmt.Errorf("httpapi: operation: %w", err)` error wrapping
- NEVER break existing API endpoints — only add new ones
- NEVER expose internal error details in API responses
