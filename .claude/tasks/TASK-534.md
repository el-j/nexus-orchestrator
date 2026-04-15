---
id: TASK-534
planId: PLAN-069
title: 'Add 4 missing brain routes to buildHowToDoc'
role: docs
status: todo
createdAt: 2026-04-13T16:00:00Z
---

# TASK-534 — Document missing brain routes in buildHowToDoc

## Context

`internal/adapters/inbound/httpapi/howto.go` — `buildHowToDoc()` lists 5 brain endpoints in the
`Endpoints` slice:

```go
{"POST", "/api/brain/ingest",          "..."},
{"GET",  "/api/brain/status",          "..."},
{"POST", "/api/brain/context",         "..."},
{"POST", "/api/brain/focused-context", "..."},
{"GET",  "/api/brain/search",          "..."},
```

After TASK-529 adds 4 new routes, this list will be out of date. Clients reading `GET /api/howto`
or `GET /.well-known/nexus.json` will not know about `init`, `knowledge`, `file-map`.

There is also a typo: `"Get the knowledge repostiory status"` → `"repository"`.

## Work Required

In `internal/adapters/inbound/httpapi/howto.go`:

1. Add the 4 new brain routes to the `Endpoints` slice (after existing brain entries):
   - `{"POST", "/api/brain/init", "Auto-ingest CLAUDE.md and initialize a project's knowledge base"}`
   - `{"GET", "/api/brain/knowledge", "List all knowledge entries for a project (filter by kind)"}`
   - `{"DELETE", "/api/brain/knowledge/{id}", "Delete a knowledge entry by ID"}`
   - `{"GET", "/api/brain/file-map", "Get file path map for a project"}`

2. Fix typo: `"repostiory"` → `"repository"` in the status endpoint description.

## File Targets

- `internal/adapters/inbound/httpapi/howto.go`

## Acceptance Criteria

- `GET /api/howto` returns all 9 brain endpoint entries
- Typo fixed
- `go vet ./internal/adapters/inbound/httpapi/...` clean
