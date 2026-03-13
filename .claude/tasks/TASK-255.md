---
id: TASK-255
title: "API: HTTP endpoints for task claim, status update, session-task query"
role: api
planId: PLAN-038
status: todo
dependencies: [TASK-253]
createdAt: 2026-03-13T16:00:00.000Z
---

## Context
The HTTP API needs endpoints for the task claiming and status reporting operations, plus a query endpoint to list tasks owned by a session. The CLI and VS Code extension use HTTP, not MCP.

## Files to Read
- `internal/adapters/inbound/httpapi/server.go` (existing endpoint patterns)
- `internal/core/ports/ports.go` (Orchestrator interface)

## Implementation Steps
1. Add `POST /api/tasks/{id}/claim` endpoint:
   - Request body: `{ "sessionId": "..." }`
   - Response: 200 with claimed task JSON
   - 404 if task not found, 409 if not claimable (not QUEUED)

2. Add `PUT /api/tasks/{id}/status` endpoint:
   - Request body: `{ "sessionId": "...", "status": "COMPLETED"|"FAILED", "logs": "..." }`
   - Response: 200 with updated task JSON
   - 403 if sessionId doesn't match task owner, 409 if invalid transition

3. Add `GET /api/ai-sessions/{id}/tasks` endpoint:
   - Response: 200 with array of tasks claimed by this session
   - 404 if session not found

4. Register routes in chi router within the existing middleware chain.

## Acceptance Criteria
- [ ] `POST /api/tasks/{id}/claim` claims a QUEUED task
- [ ] `PUT /api/tasks/{id}/status` updates status with ownership check
- [ ] `GET /api/ai-sessions/{id}/tasks` returns session's claimed tasks
- [ ] Proper HTTP status codes (200, 403, 404, 409)
- [ ] JSON request/response format consistent with existing API
- [ ] `go vet ./...` passes
