---
id: TASK-353
title: Context-budget token-estimate header in all MCP tool responses
role: mcp
planId: PLAN-051
status: todo
dependencies: [TASK-352]
createdAt: 2026-03-28T00:00:00Z
---

## Context

Small-context models need to know how many tokens a response consumed so they can manage their remaining budget. This task adds a lightweight token-estimate prefix to all MCP tool responses (a rough character-count ÷ 4 heuristic is sufficient — no tokenizer needed).

## Files to Read

- `internal/adapters/inbound/mcp/tools.go` — `textResult` helper, `callToolResult` type
- `internal/adapters/inbound/mcp/server.go` — `callToolResult` struct definition

## Implementation Steps

### 1. Add `tokenEstimate` helper

In `tools.go`:

```go
// tokenEstimate returns a rough token count estimate using the "chars / 4" heuristic.
func tokenEstimate(s string) int {
    return (len(s) + 3) / 4
}
```

### 2. Add `textResultWithBudget` helper

```go
// textResultWithBudget wraps a text response with a compact token-budget header.
// The header is a single comment line that small-context models can parse.
func textResultWithBudget(content string) callToolResult {
    estimate := tokenEstimate(content)
    wrapped := fmt.Sprintf("[~%d tokens]\n%s", estimate, content)
    return textResult(wrapped)
}
```

### 3. Update `toolGetProjectContext` and `toolGetFocusedContext` to use `textResultWithBudget`

These are the two tools specifically designed for small-context models. Wrap their responses:

```go
// In toolGetProjectContext:
return textResultWithBudget(string(data)), nil

// In toolGetFocusedContext:
return textResultWithBudget(string(data)), nil
```

Also apply to `toolHowtoBrief`.

### 4. Do NOT add the budget header to all tools

Avoid modifying tools that are used in existing tests or whose output format is machine-parsed (e.g., `get_task`, `get_queue`). Only the three context-oriented tools get the header.

## Acceptance Criteria

- `get_project_context`, `get_focused_context`, and `howto_brief` responses start with `[~N tokens]`
- Existing tests are unaffected (since they don't call these three new tools yet)
- `go vet ./...` clean
