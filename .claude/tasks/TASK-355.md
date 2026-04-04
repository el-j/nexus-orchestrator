---
id: TASK-355
title: Update copilot-instructions.md with .continue MCP config + small-context workflow guide
role: planning
planId: PLAN-051
status: todo
dependencies: [TASK-354]
createdAt: 2026-03-28T00:00:00Z
---

## Context

The `copilot-instructions.md` file is the project's documentation for how AI agents should integrate with nexusOrchestrator. With PLAN-051, there is now specific guidance for `.continue` users with local small-context models (like qwen3.5-35b-a3b). This task adds a dedicated section.

## Files to Read

- `copilot-instructions.md` (project root) — existing structure and sections
- `.claude/plans/PLAN-051.md` — full plan description
- `.claude/tasks/TASK-351.md` — `howto_brief` tool details

## Implementation Steps

### 1. Add a new section to `copilot-instructions.md`:

Find the appropriate location (after the existing "Integration Guide" or "AI Agents" section) and add:

````markdown
## .continue Integration (Local Models)

If you use [Continue](https://continue.dev) with a local model (e.g. `qwen3.5-35b-a3b`, `llama3`, `codestral`), configure the nexusOrchestrator MCP server in your `.continue/config.json`:

```json
{
  "mcpServers": [
    {
      "name": "nexusOrchestrator",
      "transport": {
        "type": "http",
        "url": "http://127.0.0.1:63988/mcp"
      }
    }
  ]
}
```
````

### Small-Context Model Workflow

Local models often have limited context windows (8K–32K tokens). Use this workflow to stay within budget:

1. **Orient** — Call `howto_brief` for the 200-token quick-start guide.
2. **Register** — Call `register_model_capabilities` with your model's context window.
3. **Get context** — Call `get_project_context {"project_path": "/your/project"}` for a compact project snapshot.
4. **Get task** — Call `get_focused_context {"task_id": "TASK-NNN"}` for task-specific implementation instructions.
5. **Claim & complete** — `claim_task` → implement → `update_task_status`.

**Token-saving tips:**

- Responses from `get_project_context` and `get_focused_context` include a `[~N tokens]` budget hint.
- Use `get_queue` instead of `get_all_tasks` (much smaller response).
- Only read files listed in `get_focused_context.filesToRead`.
- One task at a time — avoid loading full task history.

### Built-in Model Profiles

nexusOrchestrator ships with known context limits for these models:

- `qwen3.5-35b-a3b` — 32K tokens
- `qwen3-coder-next` — 128K tokens
- `llama3.1`, `llama3.2` — 128K tokens
- `codestral` — 32K tokens
- `mistral` — 32K tokens
- `deepseek-coder` — 16K tokens
- `phi-4` — 16K tokens
- `gemma` — 8K tokens

Call `get_model_capabilities {"model_id": "your-model"}` to check if your model has a built-in profile.

```

### 2. Keep all existing content intact

Do not modify any existing sections. Only append the new section.

## Acceptance Criteria

- `copilot-instructions.md` contains a new `.continue Integration` section
- The section includes the `config.json` snippet, the 5-step workflow, and the built-in model profiles list
- File is valid markdown (no broken code fences)
```
