---
id: TASK-538
planId: PLAN-069
title: 'Add search_knowledge MCP + re-ingest dedup E2E cases to brain_test.go'
role: qa
status: todo
createdAt: 2026-04-13T16:00:00Z
---

# TASK-538 — Expand brain E2E tests

## Context

`internal/e2e/brain_test.go` (`//go:build integration`) covers:

- HTTP ingest → status → context → focused-context
- MCP ingest_knowledge + get_project_context

Missing E2E coverage:

1. **`search_knowledge` MCP tool** — never called in E2E; verifies FTS5 is compiled in and
   BM25 ranking works end-to-end against a real SQLite DB
2. **Re-ingest dedup** — ingesting the same file twice should NOT double the entry count; verifies
   upsert logic at the full stack level (HTTP → service → repo)

## Work Required

Extend `internal/e2e/brain_test.go` (do NOT create a new file):

1. After the existing `TestBlackboxBrainE2E`, add a step (or new test function) that:
   - Calls `GET /api/brain/search?projectPath=...&q=layer&limit=5` via HTTP
   - Verifies `200` and at least 1 result in `results` array

2. In `TestBlackboxBrainE2E` (or a new `TestBrainReIngest`):
   - Ingests the same file a second time
   - Calls `GET /api/brain/status` again
   - Verifies `entryCount` is still `2` (not `4`) — dedup worked

3. Add `search_knowledge` MCP call in the MCP section:
   ```json
   {
     "name": "search_knowledge",
     "arguments": { "projectPath": "...", "query": "architecture", "limit": 3 }
   }
   ```
   Verify no error and at least 1 result.

## File Targets

- `internal/e2e/brain_test.go`

## Acceptance Criteria

- `CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go test -race -count=1 -tags=integration ./internal/e2e/...` all green
- Re-ingest dedup assertion present and passing
- `search_knowledge` MCP assertion present and passing
