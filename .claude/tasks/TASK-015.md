---
id: TASK-015
title: HTTP API — task history endpoint + session management endpoints
role: api
planId: PLAN-002
status: todo
dependencies: []
createdAt: 2026-03-09T12:00:00.000Z
---

## Context

The HTTP API currently only supports CRUD for active/queued tasks (`POST /api/tasks`, `GET /api/tasks/{id}`, `DELETE /api/tasks/{id}`). There is no way to query completed or failed tasks (history), and no API surface for session management. These are needed by the GUI (TASK-019, TASK-021), the writeback system (TASK-027), and the `sync-from-nexus` command (TASK-031).

## Files to Read

- `internal/adapters/inbound/httpapi/server.go` — existing routes
- `internal/core/ports/ports.go` — TaskRepository + SessionRepository interfaces
- `internal/adapters/outbound/repo_sqlite/repo.go` — to understand what queries already exist
- `internal/core/domain/task.go` — Task struct fields

## Implementation Steps

1. **Extend `TaskRepository` port** in `internal/core/ports/ports.go`:
   ```go
   // GetAll returns tasks filtered by optional status. Empty string = all tasks.
   GetAll(status string) ([]domain.Task, error)
   // GetBySourceProject returns tasks submitted with the given sourceProjectPath.
   GetBySourceProject(sourceProjectPath string) ([]domain.Task, error)
   ```

2. **Implement `GetAll` and `GetBySourceProject` in `repo_sqlite/repo.go`**:
   - `GetAll("completed")` → `SELECT ... WHERE status = 'completed' ORDER BY updated_at DESC`
   - `GetAll("")` → `SELECT ... ORDER BY created_at DESC`
   - `GetBySourceProject(path)` → `SELECT ... WHERE source_project_path = ? ORDER BY updated_at DESC`
   - Reuse the existing task row scanning logic to avoid duplication.

3. **Add `GET /api/tasks` route** (list all tasks, optional `?status=completed|failed|queued|processing`):
   ```
   GET /api/tasks?status=completed
   → 200 []domain.Task JSON array
   GET /api/tasks  (no filter)
   → 200 all tasks, newest first
   ```
   Register on the chi router: `r.Get("/api/tasks", s.handleListTasks)`.
   
   Note: The existing `POST /api/tasks` uses a non-parameterized path — adding `GET /api/tasks` is additive and does not conflict.

4. **Add session routes**:
   ```
   GET  /api/sessions/{projectPath}   → 200 []domain.Message JSON   (404 if no session)
   DELETE /api/sessions/{projectPath} → 204 No Content             (idempotent)
   ```
   `projectPath` is URL-encoded in the path: decode with `url.QueryUnescape(chi.URLParam(r, "projectPath"))` then `filepath.Clean`.
   
   - `GET`: call `sessionRepo.GetOrCreate(projectPath)`, return `session.Messages` (empty array if no history).
   - `DELETE`: call `sessionRepo.Delete(projectPath)` — add this method to the `SessionRepository` port and implement in `repo_sqlite/session_repo.go`.

5. **Add `Delete(projectPath string) error` to `SessionRepository` port** in `ports.go`:
   ```go
   Delete(projectPath string) error
   ```
   Implement as `DELETE FROM sessions WHERE project_path = ?`.

6. **Update `Server` struct** to accept `sessionRepo ports.SessionRepository` as a dependency (it currently only holds `orch ports.Orchestrator`). Expose via `NewServer(orch, sessionRepo)`.
   Update callers in `main.go` and `cmd/nexus-daemon/main.go` to pass the session repo.

7. **Error responses** must use proper HTTP status codes:
   - `404` for unknown projectPath in session GET (when session has 0 messages and was never created — return empty array, not 404)
   - `500` for DB errors with a JSON body `{"error": "internal server error"}`

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go build .` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] New test `TestHandleListTasks`: GET /api/tasks returns all tasks, GET /api/tasks?status=completed returns only completed
- [ ] New test `TestHandleGetSession`: GET /api/sessions/... returns messages for seeded session
- [ ] New test `TestHandleDeleteSession`: DELETE /api/sessions/... returns 204, subsequent GET returns empty array
- [ ] `SessionRepository.Delete()` port method exists and is implemented in SQLite repo
- [ ] `GetAll()` and `GetBySourceProject()` port methods exist and are implemented

## Anti-patterns to Avoid

- NEVER import adapters from core services
- NEVER return raw DB errors in HTTP responses — wrap in `{"error": "..."}` JSON
- NEVER use `chi.URLParam` without decoding URL encoding for file paths
- NEVER add auth/authorization logic — this is a local daemon; security is out of scope for this task
