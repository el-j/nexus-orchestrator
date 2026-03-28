---
id: TASK-351
title: MCP tool — howto_brief (ultra-compact integration guide)
role: mcp
planId: PLAN-051
status: done
dependencies: []
createdAt: 2026-03-28T00:00:00Z
---

## Context

The existing `howto` tool returns a detailed integration guide — great for large-context models but too long for qwen3.5-35b-a3b. `howto_brief` returns a 50-line, under-200-token essential guide. The goal: a model can call this as its very first MCP tool and immediately know how to work with the orchestrator.

## Files to Read

- `internal/adapters/inbound/mcp/tools.go` — `toolHowto()` implementation (line ~615)

## Implementation Steps

### 1. Add `toolHowtoBrief` to `tools.go`

```go
// toolHowtoBrief returns an ultra-compact integration guide for small-context models.
func (s *Server) toolHowtoBrief() (callToolResult, error) {
    guide := `nexusOrchestrator — Quick Start (compact edition)
==================================================
You are connected to nexusOrchestrator, an AI task orchestration server.

FIRST STEPS (run in order):
  1. get_project_context {"project_path": "/path/to/project"}
     → Returns active plan, task counts, guidance.
  2. get_focused_context {"task_id": "TASK-NNN"}
     → Returns implementation steps + files to read for one task.
  3. claim_task {"task_id": "TASK-NNN", "session_id": "your-session-id"}
     → Marks the task as yours (PROCESSING).
  4. update_task_status {"task_id": "TASK-NNN", "status": "COMPLETED", "logs": "summary"}
     → Marks done. Returns task to queue if failed.

KEY TOOLS:
  howto              — full guide (large context only)
  howto_brief        — this guide
  get_project_context — compact project snapshot
  get_focused_context — task implementation bundle
  submit_task        — queue a new task for an LLM
  get_queue          — list queued tasks
  health             — ping daemon

SMALL-CONTEXT TIP:
  Use get_project_context first, then get_focused_context for ONE task at a time.
  Do not call get_all_tasks (response too large). Use get_queue instead.
  Register your model: register_model_capabilities {"model_id": "...", "context_window": 32768}
`
    return textResult(guide), nil
}
```

### 2. Wire in `handleToolCall` switch

```go
case "howto_brief":
    result, err = s.toolHowtoBrief()
```

### 3. Add to `toolList`

```go
toolDef{
    Name:        "howto_brief",
    Description: "Get the ultra-compact integration guide (under 200 tokens). RECOMMENDED as first call for small-context models (< 32K tokens). Use howto for the full guide.",
    InputSchema: toolSchema{Type: "object", Properties: map[string]toolProp{}},
},
```

Place `howto_brief` immediately after `howto` in the tool list so it appears second.

### 4. Update `serverInfo.Instructions` in `server.go`

Change the initialize response instructions to mention `howto_brief`:

```
"Call howto_brief first if you have a small context window (< 64K tokens), or howto for the full guide."
```

## Acceptance Criteria

- `howto_brief` returns a guide under 300 lines
- It appears in `tools/list` response
- `go vet ./...` clean; existing tests pass
