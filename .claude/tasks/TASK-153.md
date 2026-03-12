---
id: TASK-153
title: HTTP API — /api/ai-sessions CRUD + Orchestrator port methods
role: api
planId: PLAN-022
status: todo
dependencies: [TASK-151, TASK-152]
priority: high
estimated_effort: L
createdAt: 2026-03-12T11:00:00.000Z
---

## Goal
Add `POST /api/ai-sessions`, `GET /api/ai-sessions`, `DELETE /api/ai-sessions/{id}` HTTP endpoints and the corresponding `RegisterAISession`, `ListAISessions`, `DeregisterAISession` methods on the `Orchestrator` port and `OrchestratorService`, so external agents (VS Code extension, MCP clients) can self-register.

## Context
The change touches three layers in dependency order:
1. **Port** (`internal/core/ports/ports.go`) — add 3 methods to the `Orchestrator` inbound interface
2. **Service** (`internal/core/services/orchestrator.go`) — implement those methods, delegate to `AISessionRepository`
3. **HTTP adapter** (`internal/adapters/inbound/httpapi/server.go`) — add 3 route handlers
4. **SSE** — on session register/deregister, emit `ai_session_changed` event via `EventBroadcaster`

## Scope

### Files to modify
- `internal/core/ports/ports.go` — add 3 methods to `Orchestrator` interface
- `internal/core/services/orchestrator.go` — implement the 3 new methods; add `aiSessionRepo ports.AISessionRepository` field
- `internal/adapters/inbound/httpapi/server.go` — register 3 new routes + 3 handler functions
- `internal/adapters/inbound/httpapi/server_test.go` — extend mock orchestrator + add HTTP tests

## Implementation Steps

### 1. ports.go — extend Orchestrator interface
Add to the `Orchestrator` interface (after `ListProviderConfigs`):
```
RegisterAISession(ctx context.Context, s domain.AISession) (domain.AISession, error)
ListAISessions(ctx context.Context) ([]domain.AISession, error)
DeregisterAISession(ctx context.Context, id string) error
```

### 2. orchestrator.go — add aiSessionRepo + implement methods

Add `aiSessionRepo ports.AISessionRepository` field to `OrchestratorService` struct.

Update `NewOrchestrator` (or add `WithAISessionRepo(r ports.AISessionRepository)` setter — follow the same pattern as `SetBroadcaster`) to accept an optional AISessionRepo. When nil, `RegisterAISession` returns an error; `ListAISessions` returns empty slice.

**`RegisterAISession`**:
- Assign a `uuid.New().String()` ID if `s.ID` is empty
- Set `s.Status = domain.SessionStatusActive`, `s.CreatedAt = time.Now()`, `s.UpdatedAt = s.CreatedAt`, `s.LastActivity = s.CreatedAt`
- Call `aiSessionRepo.SaveAISession(ctx, s)`
- Broadcast `ports.TaskEvent{Type: "ai_session_changed", TaskID: s.ID, Status: "active"}` via broadcaster (reuse `TaskEvent` — note: the Status field is `domain.TaskStatus`, so cast `domain.AISessionStatus` to `domain.TaskStatus` for the broadcast payload, or use a sentinel string; prefer a new `EventType` constant `EventAISessionChanged`)
- Return the populated `domain.AISession`

**`ListAISessions`**:
- Delegates to `aiSessionRepo.ListAISessions(ctx)`
- If `aiSessionRepo == nil`, return `(nil, nil)` — no error

**`DeregisterAISession`**:
- Calls `aiSessionRepo.UpdateAISessionStatus(ctx, id, domain.SessionStatusDisconnected, time.Now())`
- Then broadcasts `ai_session_changed` event

### 3. httpapi/server.go — add routes + handlers
In the chi router setup add after the existing provider routes:
```
r.Post("/api/ai-sessions", s.handleRegisterAISession)
r.Get("/api/ai-sessions", s.handleListAISessions)
r.Delete("/api/ai-sessions/{id}", s.handleDeregisterAISession)
```

**`handleRegisterAISession`**:
- Decode request body into a partial `domain.AISession` (AgentName required, Source required, ProjectPath optional)
- Validate: `AgentName` must not be empty → 400
- Validate: `Source` must be one of `mcp/vscode/http` → 400
- Call `s.orch.RegisterAISession(r.Context(), session)`
- Return 201 with the full session JSON

**`handleListAISessions`**:
- Call `s.orch.ListAISessions(r.Context())`
- Return 200 with JSON array (never null — write `[]` if empty)

**`handleDeregisterAISession`**:
- Read `{id}` from URL param
- Call `s.orch.DeregisterAISession(r.Context(), id)`
- Return 204 No Content on success; 404 if `domain.ErrNotFound` wrapped

### 4. server_test.go — extend mock + add tests
Add `RegisterAISession`, `ListAISessions`, `DeregisterAISession` to the mock orchestrator.
Add table-driven HTTP tests:
- `POST /api/ai-sessions` with missing AgentName → 400
- `POST /api/ai-sessions` valid → 201 + body has `id`
- `GET /api/ai-sessions` → 200 + JSON array
- `DELETE /api/ai-sessions/unknown-id` → 404

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] All three HTTP endpoints return correct status codes as specified above
- [ ] `Orchestrator` interface has 3 new methods; all existing implementations still compile
- [ ] SSE `ai_session_changed` event emitted on register and deregister
- [ ] New `EventAISessionChanged` constant added to `ports.EventType` if appropriate
- [ ] Empty session list returns `[]` not `null`

## Anti-patterns to Avoid
- NEVER import `repo_sqlite` or any adapter from `services/orchestrator.go`
- NEVER bypass the `Orchestrator` port from HTTP handlers — only call `s.orch.*` methods
- NEVER panic on nil `aiSessionRepo` — gracefully return empty results
- NEVER forget to update the mock orchestrator in server_test.go (compilation failure)
- NEVER use goroutines inside `OrchestratorService` methods — worker goroutine is already in `runWorker()`
