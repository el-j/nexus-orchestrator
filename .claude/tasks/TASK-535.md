---
id: TASK-535
planId: PLAN-069
title: 'Write brain HTTP handler tests'
role: qa
status: todo
createdAt: 2026-04-13T16:00:00Z
---

# TASK-535 — Brain HTTP handler tests

## Context

No test file covers the brain HTTP handlers. After TASK-529 adds 4 new routes, there will be
9 brain endpoints total with zero handler-level test coverage:

- `POST /api/brain/ingest`
- `GET /api/brain/status`
- `POST /api/brain/context`
- `POST /api/brain/focused-context`
- `GET /api/brain/search`
- `POST /api/brain/init` (new)
- `GET /api/brain/knowledge` (new)
- `DELETE /api/brain/knowledge/{id}` (new)
- `GET /api/brain/file-map` (new)

**Depends on**: TASK-529 (routes exist).

## Work Required

Create `internal/adapters/inbound/httpapi/brain_handlers_test.go`.

Use the existing test patterns from `server_test.go` (mock `ports.Orchestrator`, `httptest.NewRecorder`).
Create a `mockBrainService` that implements `ports.BrainService` with controllable return values.

Cover per-handler:

- Happy path: correct status code + JSON shape
- Missing required params: `400`
- `brain == nil` guard: `503`
- Service error: `500`

At minimum write one test per handler (9 handlers × happy path = 9 test functions).

## File Targets

- `internal/adapters/inbound/httpapi/brain_handlers_test.go` (new)

## Acceptance Criteria

- `CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go test -race -count=1 ./internal/adapters/inbound/httpapi/...` all green
- At least 9 test functions covering each handler happy path
- `nil` brain guard and bad-input `400` covered for at least `handleIngestKnowledge`
