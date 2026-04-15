---
id: TASK-532
planId: PLAN-069
title: 'Fix howto_brief phantom parameter names'
role: mcp
status: todo
createdAt: 2026-04-13T16:00:00Z
---

# TASK-532 — Fix howto_brief phantom parameter names

## Context

`internal/adapters/inbound/mcp/tools.go` — `toolHowtoBrief()` (line ~754) is the compact
integration guide for small-context AI models. The guide contains three wrong parameter names that
don't match actual MCP tool schemas:

| Guide text                                                | Actual tool schema                          |
| --------------------------------------------------------- | ------------------------------------------- |
| `get_project_context {"project_path": "..."}`             | `{"projectPath": "..."}`                    |
| `get_focused_context {"task_id": "TASK-NNN"}`             | `{"projectPath": "...", "question": "..."}` |
| `claim_task {"task_id": "TASK-NNN", "session_id": "..."}` | `{"id": "TASK-NNN", "sessionId": "..."}`    |

An AI using `howto_brief` as its primary orientation guide will form tool calls with incorrect
field names, causing JSON unmarshalling failures at the server.

The guide also does not mention the brain tools (`get_brain_status`, `ingest_knowledge`,
`search_knowledge`) in the KEY TOOLS section.

## Work Required

In `internal/adapters/inbound/mcp/tools.go`, `toolHowtoBrief()`:

1. Fix `get_project_context` example: `project_path` → `projectPath`
2. Fix `get_focused_context` example: replace `task_id` with `projectPath` + `question` fields
3. Fix `claim_task` example: `task_id` → `id`, `session_id` → `sessionId`
4. Add brain tools to KEY TOOLS section: `get_brain_status`, `ingest_knowledge`, `search_knowledge`

## File Targets

- `internal/adapters/inbound/mcp/tools.go`

## Acceptance Criteria

- All parameter names in `howto_brief` match the real tool schemas in `toolList()`
- `go vet ./internal/adapters/inbound/mcp/...` clean
