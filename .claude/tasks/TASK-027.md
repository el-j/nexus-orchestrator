---
id: TASK-027
title: Port + outbound adapter — WritebackClient (fs_writeback)
role: backend
planId: PLAN-002
status: todo
dependencies: [TASK-026]
createdAt: 2026-03-09T13:00:00.000Z
---

## Context

When the orchestrator completes a task that has `SourceProjectPath` set, it needs to write results back to the source project's `.claude/orchestrator.json` and task file. This requires a new port `WritebackClient` and a new outbound adapter `internal/adapters/outbound/fs_writeback/` that reads/writes local filesystem files. The adapter must operate atomically (write-to-temp then rename) to avoid corrupting the destination files.

## Files to Read

- `internal/core/ports/ports.go` — all existing port interfaces (to add WritebackClient)
- `internal/core/domain/task.go` — Task struct (with fields from TASK-026)
- `internal/adapters/outbound/fs_writer/writer.go` — existing fs adapter for pattern reference

## Implementation Steps

1. **Add `WritebackClient` port** to `internal/core/ports/ports.go`:
   ```go
   // WritebackClient writes task completion results back to the source project's
   // .claude/orchestrator.json and .claude/tasks/<SourceTaskID>.md.
   type WritebackClient interface {
       WriteBack(ctx context.Context, payload WritebackPayload) error
   }

   // WritebackPayload carries all data needed to update the source project.
   type WritebackPayload struct {
       SourceProjectPath string
       SourceTaskID      string
       SourcePlanID      string    // optional; used to update plan status if all tasks done
       NexusTaskID       string
       Status            string    // "completed" or "failed"
       Output            string    // LLM output text written to task file
       CompletedAt       time.Time
   }
   ```

2. **Create `internal/adapters/outbound/fs_writeback/` directory** with `writeback.go`:
   ```go
   package fs_writeback

   import (
       "context"
       "encoding/json"
       "fmt"
       "os"
       "path/filepath"
       "time"

       "nexus-orchestrator/internal/core/ports"
   )

   type Client struct{}

   func New() *Client { return &Client{} }

   func (c *Client) WriteBack(ctx context.Context, p ports.WritebackPayload) error {
       orchestratorPath := filepath.Join(p.SourceProjectPath, ".claude", "orchestrator.json")
       taskFilePath := filepath.Join(p.SourceProjectPath, ".claude", "tasks", p.SourceTaskID+".md")

       // Step 1: Update orchestrator.json
       if err := c.updateOrchestratorJSON(orchestratorPath, p); err != nil {
           return fmt.Errorf("fs_writeback: update orchestrator.json: %w", err)
       }

       // Step 2: Append output to task file
       if p.SourceTaskID != "" {
           if err := c.appendTaskOutput(taskFilePath, p); err != nil {
               return fmt.Errorf("fs_writeback: append task output: %w", err)
           }
       }
       return nil
   }
   ```

3. **Implement `updateOrchestratorJSON`**:
   - Read the file: `os.ReadFile(orchestratorPath)` — return `fmt.Errorf(...ErrNotFound...)` if file does not exist
   - Unmarshal into `map[string]interface{}` (generic JSON, not a typed struct — avoids coupling to the CLI project's schema)
   - Navigate: `data["tasks"].(map[string]interface{})[p.SourceTaskID].(map[string]interface{})`
   - Set: `task["status"] = "done"`, `task["completedAt"] = p.CompletedAt.UTC().Format(time.RFC3339)`
   - Add: `task["nexusTaskId"] = p.NexusTaskID` (traceability)
   - Re-marshal with `json.MarshalIndent(data, "", "  ")`
   - Atomic write: `os.WriteFile(tmp, bytes, 0644)` then `os.Rename(tmp, orchestratorPath)`
   - Temp file path: `orchestratorPath + ".nexus.tmp"`

4. **Implement `appendTaskOutput`**:
   - Read existing file with `os.ReadFile` (skip if file does not exist — not an error)
   - Append section:
     ```markdown

     ## Nexus Output
     <!-- Written by nexusOrchestrator on <timestamp> — NexusTaskID: <id> -->

     <output content>
     ```
   - Atomic write as above

5. **Error handling**:
   - If `SourceProjectPath` is empty: return nil (no writeback needed — not an error condition)
   - If orchestrator.json doesn't exist: return wrapped `fs_writeback: orchestrator.json not found: %w`
   - If task key doesn't exist in JSON: log warning + return nil (task may have been deleted — don't fail the orchestrator)
   - All file operations use `filepath.Clean` on paths

6. **No-op implementation for tests** — define `NoopWritebackClient` in the same package:
   ```go
   type NoopClient struct{}
   func (n *NoopClient) WriteBack(_ context.Context, _ ports.WritebackPayload) error { return nil }
   ```

7. **Tests** in `fs_writeback/writeback_test.go`:
   - `TestWriteBackUpdatesOrchestratorJSON`: seed a temp orchestrator.json with a task, call WriteBack, verify `status: "done"` and `completedAt` set
   - `TestWriteBackAppendsToTaskFile`: seed a TASK-045.md, call WriteBack with Output, verify "## Nexus Output" section appended
   - `TestWriteBackEmptySourcePath`: SourceProjectPath="" returns nil (no-op)
   - `TestWriteBackMissingOrchestratorJSON`: missing file returns an error

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `ports.WritebackClient` interface + `WritebackPayload` struct defined in ports.go
- [ ] `internal/adapters/outbound/fs_writeback/writeback.go` exists implementing `WritebackClient`
- [ ] Atomic write (write-to-tmp + os.Rename) used in all file write operations
- [ ] `TestWriteBackUpdatesOrchestratorJSON` passes
- [ ] `TestWriteBackEmptySourcePath` returns nil (no-op correctly handled)

## Anti-patterns to Avoid

- NEVER import `fs_writeback` from `internal/core/services/` — inject via port interface
- NEVER use `ioutil.WriteFile` (deprecated) — use `os.WriteFile`
- NEVER unmarshal orchestrator.json into a nexusOrchestrator-specific struct — use `map[string]interface{}` to decouple from source project schema
- NEVER skip atomic write — non-atomic writes corrupt files on crash
- NEVER panic on missing task key in orchestrator.json — log + return nil
