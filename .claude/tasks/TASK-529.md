---
id: TASK-529
planId: PLAN-069
title: 'Add 4 missing brain HTTP routes and handlers'
role: backend
status: todo
createdAt: 2026-04-13T16:00:00Z
---

# TASK-529 — Add missing brain HTTP routes + handlers

## Context

`internal/adapters/inbound/httpapi/server.go` registers 5 brain routes (ingest, status, context,
focused-context, search). The `ports.BrainService` interface defines 4 additional methods that have
no corresponding HTTP routes:

- `InitProject(ctx, projectPath, claudeMDPath) BrainStatus`
- `ListKnowledge(ctx, projectPath, kind) []ProjectKnowledge`
- `DeleteKnowledge(ctx, id) error`
- `GetFileMap(ctx, projectPath, focusArea) []string`

Without routes, the CLI and Wails clients cannot call these operations via the daemon.

## Work Required

1. **`internal/adapters/inbound/httpapi/brain_handlers.go`** — add 4 handlers:
   - `handleInitProject` → `POST /api/brain/init` body `{projectPath, claudeMDPath?}`
   - `handleListKnowledge` → `GET /api/brain/knowledge?projectPath=&kind=`
   - `handleDeleteKnowledge` → `DELETE /api/brain/knowledge/{id}`
   - `handleGetFileMap` → `GET /api/brain/file-map?projectPath=&focusArea=`

2. **`internal/adapters/inbound/httpapi/server.go`** — register the 4 new routes in `setupRoutes()`.

## File Targets

- `internal/adapters/inbound/httpapi/brain_handlers.go`
- `internal/adapters/inbound/httpapi/server.go`

## Acceptance Criteria

- `go vet ./internal/adapters/inbound/httpapi/...` clean
- `POST /api/brain/init` returns `200 BrainStatus` JSON
- `GET /api/brain/knowledge` returns `200 []ProjectKnowledge` JSON
- `DELETE /api/brain/knowledge/{id}` returns `204` on success, `404` on missing
- `GET /api/brain/file-map` returns `200 {filePaths: []}` JSON
