---
id: TASK-519
title: Integration and E2E tests for brain feature
role: qa
planId: PLAN-066
status: done
dependencies: [TASK-518]
createdAt: 2026-04-13T00:00:00Z
---

## Context

End-to-end tests verify the complete brain workflow: MCP tools + HTTP endpoints + daemon wiring. Tests exercise the full stack from tool call → service → repository → SQLite and back. Follow existing E2E/blackbox test patterns.

## Files to Read

- `internal/adapters/inbound/mcp/tools_test.go` — MCP tool test harness (if exists), how tools are tested
- `internal/adapters/inbound/httpapi/handlers_test.go` or similar — HTTP handler test patterns
- `internal/e2e/` or any blackbox test directory — look for existing integration test structure
- `internal/adapters/inbound/mcp/tools.go` — to understand how to set up test MCP server

## Implementation Steps

1. **MCP tool tests** — add to `tools_test.go` (or create `brain_tools_test.go` in mcp package):
   - Setup: create in-memory SQLite repo, create BrainService, create MCP server with `SetBrainService`
   - Test `brain_init` tool: call with valid projectPath + temp CLAUDE.md → returns BrainStatus JSON with EntryCount > 0
   - Test `get_project_context` tool: call after brain_init → returns ContextResponse JSON with Sections non-empty
   - Test `get_project_context` tool: call with no knowledge → returns ContextResponse with empty sections (not error)
   - Test `ingest_knowledge` tool: call with kind=convention, topic, content → returns ProjectKnowledge JSON
   - Test `search_knowledge` tool: ingest first, then search → returns matching sections
   - Test `brain_init` with nil brain service → returns error containing "brain service not initialized"

2. **HTTP handler tests** — add to `handlers_brain_test.go`:
   - Setup: create test HTTP server with brainSvc wired (use httptest.NewServer)
   - Test `POST /api/brain/init` → 200 with BrainStatus
   - Test `POST /api/brain/context` → 200 with ContextResponse
   - Test `POST /api/brain/knowledge` with missing fields → 400
   - Test `POST /api/brain/knowledge` with all fields → 201 with ProjectKnowledge
   - Test `POST /api/brain/search` → 200 with sections array
   - Test `GET /api/brain/status?project=/tmp/test` → 200 with BrainStatus
   - Test all endpoints with nil brainSvc → 501 Not Implemented
   - Test `GET /api/brain/files?project=/tmp/test` → 200 with array (empty ok)

3. **Brain workflow integration test** (in existing E2E test file or new `brain_e2e_test.go`):
   - Full round-trip: `brain_init` → `get_project_context` → `ingest_knowledge` → `search_knowledge`
   - Verify: knowledge ingested by brain_init is returned by get_project_context
   - Verify: manually ingested knowledge is returned by search_knowledge
   - Verify: token budget respected (response.TokensUsed <= response.TokenBudget)

4. **FTS5 trigger correctness test** (in knowledge_repo_test.go, can be added to TASK-512 or here):
   - Insert entry, update its content, search for new content → found
   - Delete entry, search for its content → not found
   - Verifies FTS5 triggers stay synchronized

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (all new tests pass)
- [ ] At least 6 MCP tool tests covering all 6 brain tools
- [ ] At least 8 HTTP handler tests covering success + error cases
- [ ] At least 1 full workflow integration test
- [ ] Nil-brain-service tests verify 501 / error responses

## Anti-patterns to Avoid

- NEVER start real daemon process in tests — use in-memory setup
- NEVER test only happy path — each handler/tool needs at least 1 error case
- NEVER use real network in unit/integration tests — httptest.NewServer only
