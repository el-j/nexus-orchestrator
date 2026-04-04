---
id: TASK-308
title: Add httpapi tests for 8 untested routes
role: testing
planId: PLAN-047
status: todo
dependencies: [TASK-303]
createdAt: 2026-03-25T00:00:00.000Z
---

## Context

`internal/adapters/inbound/httpapi/server.go` has 33.4% coverage. Eight routes added in PLAN-043–046 have zero test coverage:

| Route                                | Handler                         | Missing scenarios                                     |
| ------------------------------------ | ------------------------------- | ----------------------------------------------------- |
| POST /api/ai-sessions/{id}/terminate | handleTerminateAISession        | 204 success, force=true, 404, 400 bad JSON            |
| POST /api/ai-sessions/{id}/heartbeat | handleHeartbeatAISession        | 204 success, 404                                      |
| POST /api/tasks/{id}/heartbeat       | handleHeartbeatTask             | 204 success, 400 missing session_id, 404              |
| POST /api/tasks/{id}/claim           | handleClaimTask                 | 204 success, 409 conflict, 400 missing sessionId, 404 |
| PUT /api/tasks/{id}/status           | handleUpdateTaskStatus          | 200 success, 403 forbidden, 400 invalid status, 404   |
| POST /api/ai-sessions/{id}/delegate  | handleDelegateToNexus           | 200 success, 404                                      |
| GET /api/ai-sessions/{id}/tasks      | handleGetSessionTasks           | 200 success, empty list                               |
| DELETE /api/ai-sessions              | handlePurgeDisconnectedSessions | 200 success with count                                |

## Files to Read

- `internal/adapters/inbound/httpapi/server_test.go` — existing test patterns (mockOrchestrator, newServer helper, table-driven tests)
- `internal/adapters/inbound/httpapi/server.go` — all 8 handlers to understand request/response format

## Implementation Steps

Add test functions for each route following the existing pattern:

- Use the existing `newServer(t, mock)` helper (or whatever the test setup function is called)
- Use table-driven tests where multiple scenarios exist
- Assert HTTP status code, response body shape

Example skeleton:

```go
func TestHandleTerminateAISession(t *testing.T) {
    tests := []struct {
        name       string
        id         string
        body       string
        terminateErr error
        wantStatus int
    }{
        {"success no force", "sess-1", `{}`, nil, http.StatusNoContent},
        {"success force true", "sess-1", `{"force":true}`, nil, http.StatusNoContent},
        {"not found", "missing", `{}`, domain.ErrNotFound, http.StatusNotFound},
        {"bad json body", "sess-1", `{bad}`, nil, http.StatusBadRequest},
    }
    // ...
}
```

Add `terminateAISessionErr` and similar fields to the `mockOrchestrator` struct as needed.

## Acceptance Criteria

- [ ] `go vet ./internal/adapters/inbound/httpapi/...` clean
- [ ] `go test ./internal/adapters/inbound/httpapi/... -race -count=1 -v` all pass
- [ ] Coverage of httpapi package improves from 33% to ≥60%
- [ ] Each of the 8 routes has at least a success + one error scenario tested
