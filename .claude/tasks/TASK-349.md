---
id: TASK-349
title: MCP tool — get_project_context (compact project snapshot)
role: mcp
planId: PLAN-051
status: todo
dependencies: [TASK-346]
createdAt: 2026-03-28T00:00:00Z
---

## Context

A local model starting a new chat needs to orient itself quickly without deep-scanning the project. `get_project_context` gives it everything it needs in one call: active plan summary, task queue state, key file paths, and a recommended next action — all sized for small-context models (<500 tokens output).

## Files to Read

- `internal/adapters/inbound/mcp/tools.go` — existing tool patterns (toolHowto, toolGetQueue)
- `internal/adapters/inbound/mcp/server.go` — Server struct, how orchestrator is accessed
- `internal/core/ports/ports.go` — Orchestrator interface methods available
- `.claude/orchestrator.json` — example of what project context looks like

## Implementation Steps

### 1. Add `GetProjectContext` to the Orchestrator port (`internal/core/ports/ports.go`)

```go
// GetProjectContext returns a compact summary of the project state for use by
// small-context local models. projectPath may be empty to return global state.
GetProjectContext(projectPath string) (ProjectContext, error)
```

Add the `ProjectContext` struct:

```go
// ProjectContext is a compact, token-efficient summary of a project's orchestration state.
type ProjectContext struct {
    // ProjectPath is the resolved project directory.
    ProjectPath string `json:"projectPath,omitempty"`
    // ActivePlanID is the currently active plan (empty if none).
    ActivePlanID string `json:"activePlanId,omitempty"`
    // ActivePlanGoal is a one-line description of the active plan.
    ActivePlanGoal string `json:"activePlanGoal,omitempty"`
    // QueuedTasks is the count of QUEUED tasks.
    QueuedTasks int `json:"queuedTasks"`
    // ProcessingTasks is the count of PROCESSING tasks.
    ProcessingTasks int `json:"processingTasks"`
    // BacklogTasks is the count of DRAFT/BACKLOG tasks.
    BacklogTasks int `json:"backlogTasks"`
    // RecentTasks is up to 5 most recent tasks with their status.
    RecentTasks []TaskSummary `json:"recentTasks,omitempty"`
    // KeyFiles is a list of important file paths (from active plan if available).
    KeyFiles []string `json:"keyFiles,omitempty"`
    // Guidance is a one-paragraph recommendation for the model's next action.
    Guidance string `json:"guidance"`
}

// TaskSummary is a compact representation of a task for context snapshots.
type TaskSummary struct {
    ID     string `json:"id"`
    Title  string `json:"title"`
    Status string `json:"status"`
}
```

### 2. Implement `GetProjectContext` in `OrchestratorService` (`internal/core/services/orchestrator.go` or `task_service.go`)

```go
func (s *OrchestratorService) GetProjectContext(projectPath string) (ports.ProjectContext, error) {
    var tasks []domain.Task
    var err error
    if projectPath != "" {
        tasks, err = s.taskRepo.GetByProjectPath(projectPath)
    } else {
        tasks, err = s.taskRepo.GetAll()
    }
    if err != nil {
        return ports.ProjectContext{}, err
    }

    ctx := ports.ProjectContext{ProjectPath: projectPath}

    // Count by status
    var recent []ports.TaskSummary
    for _, t := range tasks {
        switch t.Status {
        case domain.StatusQueued:
            ctx.QueuedTasks++
        case domain.StatusProcessing:
            ctx.ProcessingTasks++
        case domain.StatusDraft, domain.StatusBacklog:
            ctx.BacklogTasks++
        }
        if len(recent) < 5 {
            recent = append(recent, ports.TaskSummary{
                ID: t.ID, Title: t.Title, Status: string(t.Status),
            })
        }
    }
    ctx.RecentTasks = recent

    // Active plan from orchestrator state (use activePlanID if set)
    // For now read from the last note or discovery — simple heuristic
    ctx.Guidance = s.buildGuidance(ctx)
    return ctx, nil
}

func (s *OrchestratorService) buildGuidance(ctx ports.ProjectContext) string {
    if ctx.QueuedTasks > 0 {
        return fmt.Sprintf("There are %d queued tasks. Call get_focused_context with a task ID to get implementation instructions, then claim_task + update_task_status when done.", ctx.QueuedTasks)
    }
    if ctx.BacklogTasks > 0 {
        return fmt.Sprintf("No queued tasks. %d backlog items available — call get_backlog to view them, then promote_task to queue one.", ctx.BacklogTasks)
    }
    return "No active tasks. Use create_draft to add work items or submit_task to run a task now."
}
```

### 3. Add `GetProjectContext` to the mock and Wails bind

- In any mock that implements `ports.Orchestrator`, add a stub:
  ```go
  func (m *mockOrchestrator) GetProjectContext(projectPath string) (ports.ProjectContext, error) {
      return ports.ProjectContext{Guidance: "stub"}, nil
  }
  ```
- In `internal/adapters/inbound/wailsbind/bind.go`, add a binding method.

### 4. Add `toolGetProjectContext` to the MCP server

In `internal/adapters/inbound/mcp/tools.go`:

```go
func (s *Server) toolGetProjectContext(args json.RawMessage) (callToolResult, error) {
    var p struct {
        ProjectPath string `json:"project_path"`
    }
    _ = json.Unmarshal(args, &p)
    ctx, err := s.orchestrator.GetProjectContext(p.ProjectPath)
    if err != nil {
        return callToolResult{}, err
    }
    data, _ := json.MarshalIndent(ctx, "", "  ")
    return textResult(string(data)), nil
}
```

Add `case "get_project_context":` to the switch in `handleToolCall`.

Add to `toolList`:

```go
toolDef{
    Name: "get_project_context",
    Description: "Get a compact project state snapshot for small-context models. Returns active plan, task counts, recent tasks, and guidance. Ideal as first call when starting a new chat session.",
    InputSchema: toolSchema{
        Type: "object",
        Properties: map[string]toolProp{
            "project_path": {Type: "string", Description: "Absolute path to project directory. Leave empty for global state."},
        },
    },
},
```

## Acceptance Criteria

- `get_project_context` returns valid JSON with queued/processing/backlog counts and guidance text
- Response is under 500 tokens (test with an empty project)
- All existing MCP tests still pass
- `go vet ./...` clean
