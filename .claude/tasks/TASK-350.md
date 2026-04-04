---
id: TASK-350
title: MCP tool — get_focused_context (task-specific bundle)
role: mcp
planId: PLAN-051
status: todo
dependencies: [TASK-349]
createdAt: 2026-03-28T00:00:00Z
---

## Context

After calling `get_project_context`, the local model knows there are queued tasks. `get_focused_context` lets it retrieve everything needed to execute a specific task: the task description, implementation steps, acceptance criteria, and the list of files to read — all without scanning the project tree itself. This is the "deep dive" companion to `get_project_context`.

## Files to Read

- `internal/adapters/inbound/mcp/tools.go` — existing patterns
- `internal/core/ports/ports.go` — `GetProjectContext` added in TASK-349
- `internal/core/domain/task.go` — `Task` struct fields
- `.claude/tasks/TASK-345.md` — example of a rich task file with Implementation Steps section

## Implementation Steps

### 1. Add `FocusedContext` struct to `internal/core/ports/ports.go`

```go
// FocusedContext is a task-centric context bundle for small-context local models.
type FocusedContext struct {
    // Task is the full task details.
    Task domain.Task `json:"task"`
    // FilesToRead is the list of files the task specifies as context (from task body).
    FilesToRead []string `json:"filesToRead,omitempty"`
    // ImplementationSteps is the parsed "## Implementation Steps" section (max 2000 chars).
    ImplementationSteps string `json:"implementationSteps,omitempty"`
    // AcceptanceCriteria is the parsed "## Acceptance Criteria" section (max 500 chars).
    AcceptanceCriteria string `json:"acceptanceCriteria,omitempty"`
    // DependencyTitles maps dependency task IDs to their titles.
    DependencyTitles map[string]string `json:"dependencyTitles,omitempty"`
    // Guidance is a one-line action recommendation.
    Guidance string `json:"guidance"`
}

// GetFocusedContext returns a task-specific context bundle.
GetFocusedContext(taskID string) (FocusedContext, error)
```

### 2. Implement `GetFocusedContext` in OrchestratorService

```go
func (s *OrchestratorService) GetFocusedContext(taskID string) (ports.FocusedContext, error) {
    task, err := s.taskRepo.GetByID(taskID)
    if err != nil {
        return ports.FocusedContext{}, fmt.Errorf("task not found: %s", taskID)
    }

    fc := ports.FocusedContext{
        Task:    task,
        Guidance: fmt.Sprintf("Claim task %s with claim_task, implement it, then call update_task_status with COMPLETED.", task.ID),
    }

    // Parse task body sections (task.Description or task.Logs may contain markdown)
    // Extract ## Files to Read, ## Implementation Steps, ## Acceptance Criteria
    parseTaskBody(&fc, task)

    // Resolve dependency titles
    if len(task.Dependencies) > 0 {
        fc.DependencyTitles = make(map[string]string)
        for _, depID := range task.Dependencies {
            if dep, err := s.taskRepo.GetByID(depID); err == nil {
                fc.DependencyTitles[depID] = dep.Title
            }
        }
    }

    return fc, nil
}

// parseTaskBody extracts markdown sections from the task description.
func parseTaskBody(fc *ports.FocusedContext, task domain.Task) {
    body := task.Description
    if body == "" {
        return
    }
    // Extract sections by header prefix
    fc.FilesToRead = extractListSection(body, "## Files to Read")
    fc.ImplementationSteps = extractSection(body, "## Implementation Steps", 2000)
    fc.AcceptanceCriteria = extractSection(body, "## Acceptance Criteria", 500)
}
```

Add helpers `extractSection(body, header string, maxLen int) string` and `extractListSection(body, header string) []string` that scan lines for the header and collect text until the next `##` header.

### 3. Add tool handler

In `tools.go`:

```go
func (s *Server) toolGetFocusedContext(args json.RawMessage) (callToolResult, error) {
    var p struct {
        TaskID string `json:"task_id"`
    }
    if err := json.Unmarshal(args, &p); err != nil || p.TaskID == "" {
        return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "task_id required"}
    }
    fc, err := s.orchestrator.GetFocusedContext(p.TaskID)
    if err != nil {
        return callToolResult{}, err
    }
    data, _ := json.MarshalIndent(fc, "", "  ")
    return textResult(string(data)), nil
}
```

Add `case "get_focused_context":` to the switch and entry to toolList.

### 4. Add stubs to all mocks

Same pattern as TASK-349 — add `GetFocusedContext` stubs to every mock implementing `ports.Orchestrator`.

## Acceptance Criteria

- `get_focused_context` for a valid task ID returns title, description, files to read, and guidance
- For an invalid task ID, returns an error message (not a crash)
- `go vet ./...` clean; existing MCP tests still pass
