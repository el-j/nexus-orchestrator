You are the **nexusOrchestrator task executor**. Execute the task specified by `$ARGUMENTS` (a TASK-NNN id) using the appropriate specialist sub-agent.

## MCP-First Rule

- Always try to use the project's own **nexus orchestration MCP toolchain** first when it is available.
- Preferred transport order:
  1. `cmd/nexus-mcp-stdio` against `NEXUS_MCP_URL` or `http://127.0.0.1:63988/mcp`
  2. Direct MCP JSON-RPC to the daemon's `/mcp` endpoint
  3. Pure local execution only if the nexus MCP server is unavailable
- When MCP is reachable, initialize the session, call `howto` or `howto_brief`, register the worker via `register_session`, and mirror task execution with `claim_task`, `heartbeat_task`, and `update_task_status` when the queue already contains the corresponding nexus task.
- If the local `.claude` task is not yet represented in nexus, continue local execution and update `.claude/orchestrator.json`; do not invent an HTTP fallback unless MCP lacks the required capability.

## Steps

### 1. Load the task

Read `.claude/tasks/TASK-<N>.md` and `.claude/orchestrator.json`.

Verify:

- Task `status` is `todo` (not `done` or `in-progress`).
- All tasks listed in `dependencies` have `status: done` in `orchestrator.json`. If not, report blockers and stop.

### 2. Baseline check

Run: `go vet ./... && CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/... ./cmd/nexus-submit/... 2>&1`

If this fails, report the errors and stop — do not attempt to execute the task on a broken baseline.

### 3. Mark in-progress

Update `.claude/orchestrator.json`: set `tasks.<TASK-N>.status = "in-progress"` and `startedAt = <now>`.

### 4. Select agent and execute

Use the role from the task file to select the agent:

| Role                                     | Agent file                                             |
| ---------------------------------------- | ------------------------------------------------------ |
| `backend`, `api`, `cli`, `mcp`, `devops` | `.github/agents/engineering-senior-developer.agent.md` |
| `architecture`                           | `.github/agents/design-ux-architect.agent.md`          |
| `qa`                                     | `.github/agents/testing-evidence-collector.agent.md`   |
| `verify`                                 | `.github/agents/testing-reality-checker.agent.md`      |
| `planning`                               | `.github/agents/project-manager-senior.agent.md`       |

Launch a sub-agent with:

- The full content of the task file as the prompt
- Key project rules:
  - Module: `nexus-orchestrator`, Go 1.24, `CGO_ENABLED=1` required for sqlite3
  - Architecture: Hexagonal — core never imports adapters
  - Error wrapping: `fmt.Errorf("package: operation: %w", err)`
  - Concurrency: `sync.Mutex` for shared state; no goroutines in `internal/core/services/`
  - `domain.ErrNotFound` sentinel for missing entities
  - HTTP API: chi router on `:63987`; MCP server: JSON-RPC 2.0 on `:63988`
  - Tests: `CGO_ENABLED=1 go test -race -count=1 ./...`

If nexus MCP is reachable for this workspace, the sub-agent should treat nexus as the system of coordination:

- call `howto` or `howto_brief` first
- call `register_session` once before doing material work
- use `heartbeat_ai_session` and `heartbeat_task` during long-running execution when relevant
- finish by reporting completion through `update_task_status` if the task was claimed from nexus

### 5. Verify completion

After the sub-agent finishes, run:

```
go vet ./... 2>&1
CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/... ./cmd/nexus-submit/... 2>&1
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
