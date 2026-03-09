---
id: TASK-029
title: HTTP API + MCP — expose source writeback fields in submit endpoints
role: api
planId: PLAN-002
status: todo
dependencies: [TASK-026]
createdAt: 2026-03-09T13:00:00.000Z
---

## Context

The `POST /api/tasks` HTTP endpoint and the `submit_task` MCP tool currently accept only `projectPath` and `prompt`. The writeback system requires three additional optional fields: `sourceProjectPath`, `sourceTaskId`, `sourcePlanId`. These must be accepted and passed through to `domain.Task` so the orchestrator and `fs_writeback` adapter can use them.

## Files to Read

- `internal/adapters/inbound/httpapi/server.go` — `handleCreateTask` handler + request struct
- `internal/adapters/inbound/mcp/server.go` — `submit_task` tool handler + input parsing
- `internal/core/domain/task.go` — Task struct with source fields (added in TASK-026)
- `internal/core/ports/ports.go` — `Orchestrator.SubmitTask` signature

## Implementation Steps

1. **Update `Orchestrator` port's `SubmitTask` method signature** to accept the full task spec:
   
   Current: `SubmitTask(projectPath, prompt string) (string, error)`
   
   Option A (preferred): pass a `domain.Task` (or `SubmitTaskRequest` struct):
   ```go
   type SubmitTaskRequest struct {
       ProjectPath       string `json:"projectPath"`
       Prompt            string `json:"prompt"`
       SourceProjectPath string `json:"sourceProjectPath,omitempty"`
       SourceTaskID      string `json:"sourceTaskId,omitempty"`
       SourcePlanID      string `json:"sourcePlanId,omitempty"`
   }
   // Port:
   SubmitTask(req SubmitTaskRequest) (taskID string, err error)
   ```
   
   Option B: Add a separate `SubmitTaskWithSource(projectPath, prompt, sourceProjectPath, sourceTaskID, sourcePlanID string) (string, error)` method — avoids breaking existing callers.
   
   Choose **Option A** — cleaner, single method, callers just pass a struct with zero-value optional fields.

2. **Update `httpapi/server.go` `handleCreateTask`**:
   ```go
   type createTaskRequest struct {
       ProjectPath       string `json:"projectPath"`
       Prompt            string `json:"prompt"`
       SourceProjectPath string `json:"sourceProjectPath,omitempty"`
       SourceTaskID      string `json:"sourceTaskId,omitempty"`
       SourcePlanID      string `json:"sourcePlanId,omitempty"`
   }

   func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
       var req createTaskRequest
       if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
           http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
           return
       }
       id, err := s.orch.SubmitTask(ports.SubmitTaskRequest{
           ProjectPath:       req.ProjectPath,
           Prompt:            req.Prompt,
           SourceProjectPath: req.SourceProjectPath,
           SourceTaskID:      req.SourceTaskID,
           SourcePlanID:      req.SourcePlanID,
       })
       // rest unchanged
   }
   ```

3. **Update `mcp/server.go` `submit_task` tool**:
   - Add three optional input properties to the `tools/list` schema:
     ```json
     "sourceProjectPath": { "type": "string", "description": "Absolute path of the project that submitted this task via push-to-nexus (optional)" },
     "sourceTaskId":      { "type": "string", "description": "Task ID in the source project's .claude/orchestrator.json (optional)" },
     "sourcePlanId":      { "type": "string", "description": "Plan ID in the source project (optional)" }
     ```
   - Parse them from `params["input"]` map: `getString(input, "sourceProjectPath")` etc.
   - Pass to `orch.SubmitTask(ports.SubmitTaskRequest{...})`.

4. **Update `OrchestratorService.SubmitTask`** to accept the new request struct:
   ```go
   func (o *OrchestratorService) SubmitTask(req ports.SubmitTaskRequest) (string, error) {
       task := domain.Task{
           ID:                uuid.New().String(),
           ProjectPath:       filepath.Clean(req.ProjectPath),
           Prompt:            req.Prompt,
           Status:            domain.StatusQueued,
           CreatedAt:         time.Now(),
           UpdatedAt:         time.Now(),
           SourceProjectPath: req.SourceProjectPath,
           SourceTaskID:      req.SourceTaskID,
           SourcePlanID:      req.SourcePlanID,
       }
       // ... queue + repo.Save as before
   }
   ```

5. **Update `app.go` `SubmitTask` Wails binding** to accept optional source fields (these are optional JS parameters — default to empty strings):
   ```go
   func (a *App) SubmitTask(projectPath, prompt string) string {
       // existing — no source fields needed for GUI direct submissions
   }
   ```
   The GUI never uses writeback fields (those are only set by CLI/MCP). No change needed to the Wails binding.

6. **Update all callers** of `SubmitTask` (Wails `app.go`, old callers in tests) to use `ports.SubmitTaskRequest{...}`.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `ports.SubmitTaskRequest` struct defined in ports.go with all 5 fields
- [ ] `POST /api/tasks` with `sourceProjectPath`/`sourceTaskId` sets fields on created task
- [ ] `submit_task` MCP tool schema lists `sourceProjectPath`, `sourceTaskId`, `sourcePlanId`
- [ ] New test `TestSubmitTaskWithSourceFields`: HTTP POST with source fields → task has fields set in DB
- [ ] Old tests (without source fields) still pass — all source fields default to empty string

## Anti-patterns to Avoid

- NEVER make source fields required in the HTTP/MCP API — they must remain optional
- NEVER validate sourceProjectPath against the filesystem in the HTTP handler — accept any string
- NEVER expose source fields in the Wails GUI `SubmitTask` binding — GUI direct submissions never use writeback
- NEVER break the existing 2-argument `SubmitTask(projectPath, prompt)` call pattern without updating ALL callers
