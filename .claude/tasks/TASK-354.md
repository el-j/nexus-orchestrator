---
id: TASK-354
title: Update delegate_to_nexus with small-context model workflow
role: mcp
planId: PLAN-051
status: todo
dependencies: [TASK-351, TASK-349]
createdAt: 2026-03-28T00:00:00Z
---

## Context

The existing `delegate_to_nexus` tool returns instructions for AI agents to delegate work to the orchestrator. With PLAN-051, there is now a much better workflow for small-context local models. This task updates the delegation instruction to include the compact workflow path.

## Files to Read

- `internal/adapters/inbound/mcp/tools.go` — `toolDelegateToNexus` (line ~594)
- `internal/core/services/orchestrator.go` — `delegationInstruction()` or wherever the instruction text lives

## Implementation Steps

### 1. Locate `delegationInstruction` or the inline text in `toolDelegateToNexus`

Find where the delegation instruction string is built. It likely lives either in the tool handler or in a service method.

### 2. Extend the instruction to include the small-context workflow section

Add a section after the existing instructions:

```
SMALL-CONTEXT MODEL WORKFLOW (< 64K token context window):
-----------------------------------------------------------
If your context window is limited, use this efficient workflow:

Step 1: Orient yourself (call once per session):
  get_project_context {"project_path": "<your-workspace>"}

Step 2: Register your capabilities (optional but helpful):
  register_model_capabilities {"model_id": "your-model-id", "context_window": 32768}

Step 3: Get task details (instead of reading source files):
  get_focused_context {"task_id": "TASK-NNN"}

Step 4: Work on the task:
  claim_task {"task_id": "TASK-NNN", "session_id": "<your-session-id>"}
  [implement the task using the file list from get_focused_context]
  update_task_status {"task_id": "TASK-NNN", "status": "COMPLETED", "logs": "summary"}

CONTEXT BUDGET TIPS:
- get_project_context and get_focused_context prepend a [~N tokens] estimate
- Prefer get_queue over get_all_tasks (much smaller response)
- Read only the files listed in get_focused_context.filesToRead
- One task at a time — do not load the full task history into context
```

### 3. Ensure the instruction is returned as part of the tool result

The response should be plain text, not JSON, so the model can read it easily.

## Acceptance Criteria

- `delegate_to_nexus` response includes the small-context workflow section
- Existing delegation instruction content is preserved
- `go vet ./...` clean
