---
id: TASK-549
planId: PLAN-070
title: 'MCP brain_tools unit tests + fix E2E FTS search failure'
role: qa
status: todo
createdAt: 2026-04-14T02:00:00Z
---

# TASK-549 — MCP brain tools coverage + fix E2E brain test

## Context

### Gap 1: `brain_tools.go` 0% coverage

`internal/adapters/inbound/mcp/brain_tools.go` — all 5 tool handlers
(`toolGetProjectContext`, `toolGetFocusedContext`, `toolSearchKnowledge`, `toolGetBrainStatus`,
`toolIngestKnowledge`) have zero test coverage. Serialisation bugs (wrong field names, wrong
JSON structure) are invisible until an AI agent calls the tool.

### Gap 2: E2E FTS search step returns 0 results

`internal/e2e/brain_test.go` step 8 (HTTP search for "architecture") returns 0 results after
ingest. This indicates a bug in the `SearchFTS` → HTTP handler pipeline when running against
the real blackbox stack. The test was written assuming results would appear but the assertion
`len(results) >= 1` fails silently because of the `//go:build integration` tag.

Likely root cause: `SearchFTS` is called before a full-text index update is committed, OR the
search query hits an FTS5 tokenisation edge case. Investigate with the real SQLite in the E2E
stack.

## Work Required

### MCP brain tools tests

Extend `internal/adapters/inbound/mcp/tools_test.go` OR create
`internal/adapters/inbound/mcp/brain_tools_test.go`:

Create a `mockBrainForMCP` implementing `ports.BrainService` with configurable return values.
Add tests:

1. `TestMCP_GetBrainStatus_OK` — dispatch `get_brain_status`, verify JSON result
2. `TestMCP_GetBrainStatus_NilBrain` — dispatch with nil brain, verify error in result
3. `TestMCP_IngestKnowledge_OK` — dispatch `ingest_knowledge`, verify `ingestedSections` in result
4. `TestMCP_GetProjectContext_OK` — dispatch `get_project_context`, verify sections in result
5. `TestMCP_GetFocusedContext_OK` — dispatch `get_focused_context` with question
6. `TestMCP_SearchKnowledge_OK` — dispatch `search_knowledge`, verify results array

### Fix E2E FTS search

1. Run the integration test to confirm the exact failure:
   `CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go test -v -tags=integration -run TestBlackboxBrainE2E ./internal/e2e/...`
2. Diagnose: check if `SearchFTS` returns results in a unit test immediately after
   `SaveKnowledge`. If the issue is FTS5 index synchronization, try wrapping in a transaction
   or issuing `INSERT INTO knowledge_fts(knowledge_fts) VALUES('rebuild')` after inserts.
3. Fix the root cause in `knowledge_repo.go` and confirm the E2E test passes.

## File Targets

- `internal/adapters/inbound/mcp/brain_tools_test.go` (new) OR extend `tools_test.go`
- `internal/adapters/outbound/repo_sqlite/knowledge_repo.go` (if FTS sync fix needed)

## Acceptance Criteria

- `CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go test -race -count=1 ./internal/adapters/inbound/mcp/...` green
- `CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go test -tags=integration -race -count=1 ./internal/e2e/...` green (all brain steps pass including search)
- `brain_tools.go` coverage ≥ 80%
