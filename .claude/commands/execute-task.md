You are the **nexusOrchestrator task executor**. Execute the task specified by `$ARGUMENTS` (a TASK-NNN id) using the appropriate specialist sub-agent.

## Steps

### 1. Load the task
Read `.claude/tasks/TASK-<N>.md` and `.claude/orchestrator.json`.

Verify:
- Task `status` is `todo` (not `done` or `in-progress`).
- All tasks listed in `dependencies` have `status: done` in `orchestrator.json`. If not, report blockers and stop.

### 2. Baseline check
Run: `go vet ./... && CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/... 2>&1`

If this fails, report the errors and stop — do not attempt to execute the task on a broken baseline.

### 3. Mark in-progress
Update `.claude/orchestrator.json`: set `tasks.<TASK-N>.status = "in-progress"` and `startedAt = <now>`.

### 4. Select agent and execute
Use the role from the task file to select the agent:

| Role                                          | Agent file                                                   |
|-----------------------------------------------|--------------------------------------------------------------|
| `backend`, `api`, `cli`, `mcp`, `devops`      | `.github/agents/engineering-senior-developer.agent.md`       |
| `architecture`                                | `.github/agents/design-ux-architect.agent.md`                |
| `qa`                                          | `.github/agents/testing-evidence-collector.agent.md`         |
| `verify`                                      | `.github/agents/testing-reality-checker.agent.md`            |
| `planning`                                    | `.github/agents/project-manager-senior.agent.md`             |

Launch a sub-agent with:
- The full content of the task file as the prompt
- Key project rules:
  - Module: `nexus-ai`, Go 1.24, `CGO_ENABLED=1` required for sqlite3
  - Architecture: Hexagonal — core never imports adapters
  - Error wrapping: `fmt.Errorf("package: operation: %w", err)`
  - Concurrency: `sync.Mutex` for shared state; no goroutines in `internal/core/services/`
  - `domain.ErrNotFound` sentinel for missing entities
  - HTTP API: chi router on `:9999`; MCP server: JSON-RPC 2.0 on `:9998`
  - Tests: `CGO_ENABLED=1 go test -race -count=1 ./...`

### 5. Verify completion
After the sub-agent finishes, run:
```
go vet ./... 2>&1
CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/... 2>&1
CGO_ENABLED=1 go test -race -count=1 ./... 2>&1
```

Check each acceptance criterion from the task file. If any fail, report what failed and ask the user how to proceed.

### 6. Mark done
If all criteria pass:
- Update `.claude/orchestrator.json`: `tasks.<TASK-N>.status = "done"`, `completedAt = <now>`, `updatedAt = <now>`.
- Output: `TASK-<N> completed ✓`

## Core Architecture Rules (enforce in every task)
- `internal/core/domain/` — pure domain types, no framework imports
- `internal/core/ports/` — Go interfaces only, no concrete implementations
- `internal/core/services/` — business logic, depends only on ports
- `internal/adapters/inbound/` — driven by CLI, HTTP, MCP, Wails
- `internal/adapters/outbound/` — LM Studio, Ollama, SQLite, filesystem
- Dependency direction: inbound → services → ports ← outbound
