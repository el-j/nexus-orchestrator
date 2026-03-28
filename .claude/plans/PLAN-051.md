# PLAN-051: Local Model Intelligence Layer

**Status:** active
**Created:** 2026-03-28
**Goal:** Make nexusOrchestrator a smart context broker for small-context local models (e.g. qwen3.5-35b-a3b via .continue). Instead of each new chat requiring a full project scan, the model calls one MCP tool and gets a qualified, compact orientation — current plan, active tasks, key file paths, and a tool guide sized to its context budget.

---

## Problem Statement

Local models (≤32K context) accessed via `.continue` struggle with complex projects because:

1. Every new chat must re-scan the entire codebase to orient itself
2. No model-capability awareness — responses are sized for large-context models
3. No pre-computed project state — the model has to query many files and tools before it can work

## Solution

nexusOrchestrator already knows: the project plan, active tasks, discovered agents/sessions, and provider context limits. We expose this as a **compact context API** tuned for small-context models, plus:

- Model capability profiles (context window, strengths) stored and queryable
- Enhanced .continue reader that captures the model name from session files
- `get_project_context` — one call → full orientation (plan + tasks + key files)
- `get_focused_context` — one call → task-specific context (what to read + what to do)
- `howto_brief` — ultra-compact (50-line) integration guide for small-context models
- `register_model_capabilities` / `get_model_capabilities` — self-registration for models
- Context-budget header on all tool responses (token estimate)
- Updated `delegate_to_nexus` workflow guiding small-context models

---

## Waves

### Wave 1 — Domain Foundation (parallel)

- **TASK-346**: `ModelCapabilityProfile` domain type + built-in known profiles registry
- **TASK-347**: SQLite `model_capabilities` table + `ModelCapabilityRepository` port + adapter

### Wave 2 — Continue Reader Enhancement

- **TASK-348**: Extract `model` field from `.continue` session files + populate `AIActivity.Model`

### Wave 3 — New MCP Tools (parallel)

- **TASK-349**: `get_project_context` MCP tool — compact project snapshot
- **TASK-350**: `get_focused_context` MCP tool — task-specific file + context bundle
- **TASK-351**: `howto_brief` MCP tool — ultra-compact 50-line guide

### Wave 4 — Capability API (parallel)

- **TASK-352**: `register_model_capabilities` + `get_model_capabilities` MCP tools + Orchestrator port methods
- **TASK-353**: Context-budget token-estimate header in all MCP tool responses

### Wave 5 — Integration & Docs

- **TASK-354**: Update `delegate_to_nexus` to recommend new small-context workflow
- **TASK-355**: Update `copilot-instructions.md` with .continue MCP config + workflow guide

---

## Task List

| ID       | Title                                                  | Role     | Wave |
| -------- | ------------------------------------------------------ | -------- | ---- |
| TASK-346 | ModelCapabilityProfile domain type + built-in profiles | backend  | 1    |
| TASK-347 | SQLite model_capabilities repo                         | backend  | 1    |
| TASK-348 | Continue reader: extract model from session files      | backend  | 2    |
| TASK-349 | MCP tool: get_project_context                          | mcp      | 3    |
| TASK-350 | MCP tool: get_focused_context                          | mcp      | 3    |
| TASK-351 | MCP tool: howto_brief                                  | mcp      | 3    |
| TASK-352 | MCP tools: register/get model capabilities             | mcp      | 4    |
| TASK-353 | Context-budget token-estimate header in MCP responses  | mcp      | 4    |
| TASK-354 | Update delegate_to_nexus for small-context workflow    | mcp      | 5    |
| TASK-355 | Update copilot-instructions.md with .continue guide    | planning | 5    |

---

## Acceptance Criteria

1. A `.continue`-connected local model can call `howto_brief` and get oriented in <200 tokens
2. `get_project_context` returns the active plan, task count, key files in <500 tokens
3. `get_focused_context` for a task ID returns everything needed to execute that task in <800 tokens
4. `register_model_capabilities` stores a model profile; `get_model_capabilities` retrieves it
5. `.continue` session activities report the model name in `AIActivity.Model`
6. `delegate_to_nexus` includes a small-context workflow section
7. All Go tests pass with `-race`; `go vet` clean
