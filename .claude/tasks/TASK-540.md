---
id: TASK-540
planId: PLAN-070
title: 'Rebuild mcpTools + endpoints arrays in ApiReferenceView.vue and McpIntegrationView.vue'
role: docs
status: todo
createdAt: 2026-04-14T02:00:00Z
---

# TASK-540 — Update Vue docs views: MCP tools list + brain HTTP endpoints

## Context

The `docs/src/views/ApiReferenceView.vue` and `docs/src/views/McpIntegrationView.vue` Vue
components hard-code a `mcpTools` data array with only 6 of the actual 36 MCP tools. The
`ApiReferenceView.vue` `endpoints` array is also missing all 9 brain HTTP endpoints.

**`ApiReferenceView.vue`** — `mcpTools` has 6 entries (submit_task, get_task, get_queue,
cancel_task, get_providers, health). Missing 30 tools. The `endpoints` array has 8 entries
(tasks/providers/health) with no brain section.

**`McpIntegrationView.vue`** — same 6-tool truncated list. No brain tool usage examples.
No VS Code setup section (the markdown `mcp-integration.md` has it but the Vue view omits it).

There is also a confirmed parameter name error in both files: `get_task` / `cancel_task` are
documented as taking `taskId` but the actual schema in `tools.go` uses `"id"`.

## Work Required

1. Read `internal/adapters/inbound/mcp/tools.go` `toolList()` to extract all 36 tool names,
   descriptions, and key parameters.
2. Read `internal/adapters/inbound/httpapi/howto.go` Endpoints slice for all 13 brain + other
   new HTTP routes.
3. Read both Vue files fully to understand their current data structure.

**`ApiReferenceView.vue`:**

- Replace the truncated `mcpTools` array with all 36 tools, grouped into logical categories:
  Tasks, AI Sessions, Providers, Brain/Knowledge, System/Discovery.
- Add a "Brain / Project Knowledge" section to `endpoints` with all 9 brain routes.
- Fix `get_task`/`cancel_task` parameter: `taskId` → `id`.
- Add Brain section header to the TOC.

**`McpIntegrationView.vue`:**

- Replace the 6-tool list with the full 36-tool catalogue (same groupings).
- Add a "Brain Tools" usage examples subsection with `get_brain_status`, `ingest_knowledge`,
  `search_knowledge` examples showing correct JSON parameter names.
- Add a "VS Code Setup" section matching `mcp-integration.md` lines 47–68.

## File Targets

- `docs/src/views/ApiReferenceView.vue`
- `docs/src/views/McpIntegrationView.vue`

## Acceptance Criteria

- All 36 MCP tools listed in both views
- All 9 brain HTTP endpoints listed in `ApiReferenceView.vue`
- `get_task`/`cancel_task` parameter name corrected to `id`
- Brain tool examples present in `McpIntegrationView.vue`
- VS Code setup section present in `McpIntegrationView.vue`
- `cd docs && npm run build` — clean (no type errors)
