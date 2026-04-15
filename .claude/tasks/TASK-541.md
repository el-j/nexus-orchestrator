---
id: TASK-541
planId: PLAN-070
title: 'Update api-reference.md, mcp-integration.md, getting-started.md with brain content'
role: docs
status: todo
createdAt: 2026-04-14T02:00:00Z
---

# TASK-541 — Update markdown docs with brain features

## Context

Three markdown docs files are missing brain content:

- `docs/api-reference.md` — documents only 4 of 9 brain endpoints; MCP tools table has 11 of 36
- `docs/mcp-integration.md` — no brain tool usage examples
- `docs/getting-started.md` — zero mention of brain / project knowledge features

## Work Required

### `docs/api-reference.md`

1. Complete the Brain HTTP API section to include all 9 endpoints:
   - Existing: `POST /api/brain/ingest`, `GET /api/brain/status`, `POST /api/brain/context`,
     `GET /api/brain/search`
   - Missing: `POST /api/brain/focused-context`, `POST /api/brain/init`,
     `GET /api/brain/knowledge`, `DELETE /api/brain/knowledge/{id}`, `GET /api/brain/file-map`
2. Update the MCP tools table to reflect all 36 tools (grouped by category: Tasks, Sessions,
   Providers, Brain, System). Fix `get_task`/`cancel_task` param from `taskId` → `id`.

### `docs/mcp-integration.md`

Add a "Brain / Project Knowledge Tools" section in the Usage Examples area with:

- `get_brain_status` — check project knowledge base state
- `ingest_knowledge` — ingest a markdown file into the project brain
- `get_project_context` — get token-budgeted context for a project (correct params: `projectPath`, `maxTokens`)
- `get_focused_context` — focused context for a question (correct params: `projectPath`, `question`, `maxTokens`)
- `search_knowledge` — FTS5 semantic search (correct params: `projectPath`, `query`, `limit`)

### `docs/getting-started.md`

Add a "Project Brain / Knowledge Intelligence" section (after the basic task workflow section)
covering:

1. What it is (CLAUDE.md ingestion → SQLite FTS5 → context retrieval for AI agents)
2. Quick start: `POST /api/brain/ingest` curl example
3. How AI agents use it: `get_project_context` before claiming tasks
4. Link to full API reference brain section

## File Targets

- `docs/api-reference.md`
- `docs/mcp-integration.md`
- `docs/getting-started.md`

## Acceptance Criteria

- All 9 brain HTTP endpoints documented in `api-reference.md`
- Brain tool examples with correct parameter names in `mcp-integration.md`
- Brain quick-start section in `getting-started.md`
- `cd docs && npm run build` — clean
