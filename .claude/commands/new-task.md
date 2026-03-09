You are a task-definition writer for the **nexusOrchestrator** Go project. Given the description below, produce one precise, self-contained task file and register it in `.claude/orchestrator.json`.

## Input

`$ARGUMENTS` — a plain-English description of the work to be done.

## Steps

### 1. Gather context
- Read `.claude/orchestrator.json` to get `counters.nextTaskId` and `activePlanId`.
- Identify which layer is affected:
  - If domain model changes → read `internal/core/domain/task.go` and any existing domain files.
  - If port interface changes → read `internal/core/ports/ports.go`.
  - If service logic changes → read `internal/core/services/orchestrator.go` or `discovery.go`.
  - If SQLite changes → read `internal/adapters/outbound/repo_sqlite/repo.go`.
  - If HTTP API changes → read `internal/adapters/inbound/httpapi/server.go`.
  - If MCP changes → read `internal/adapters/inbound/mcp/server.go` (if it exists).
  - If CLI changes → read `internal/adapters/inbound/cli/root.go`.
  - If LLM adapter changes → read the relevant adapter in `internal/adapters/outbound/`.
- Run `go vet ./... 2>&1` to confirm baseline is passing before creating the task.

### 2. Determine role
Choose the single most appropriate role:

| Role           | Agent                                                       | Covers                                           |
|----------------|-------------------------------------------------------------|--------------------------------------------------|
| `backend`      | `.github/agents/engineering-senior-developer.agent.md`      | Core services, domain, ports, SQLite             |
| `api`          | `.github/agents/engineering-senior-developer.agent.md`      | HTTP API endpoints, chi router                   |
| `cli`          | `.github/agents/engineering-senior-developer.agent.md`      | Cobra CLI commands, remote client                |
| `mcp`          | `.github/agents/engineering-senior-developer.agent.md`      | MCP JSON-RPC 2.0 server adapter                  |
| `devops`       | `.github/agents/engineering-senior-developer.agent.md`      | Entry point wiring, Makefile, CGO build config   |
| `architecture` | `.github/agents/design-ux-architect.agent.md`               | Port contracts, domain model, protocol design    |
| `qa`           | `.github/agents/testing-evidence-collector.agent.md`        | go test, table-driven tests, coverage            |
| `verify`       | `.github/agents/testing-reality-checker.agent.md`           | Full pipeline validation, smoke tests            |
| `planning`     | `.github/agents/project-manager-senior.agent.md`            | Task decomposition, JSON updates, status reviews |

### 3. Write the task file
Create `.claude/tasks/TASK-<nextTaskId>.md` with this exact structure:

````
---
id: TASK-<N>
title: <concise action phrase>
role: <role>
planId: <activePlanId>
status: todo
dependencies: [<comma-separated TASK IDs, or empty>]
createdAt: <ISO 8601 timestamp>
---

## Context
<1-3 sentences explaining WHY this task is needed and what problem it solves>

## Files to Read
<list of file paths the executor must read before starting>

## Implementation Steps
<numbered list of concrete steps — specific enough that no follow-up questions are needed>

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [ ] <task-specific criterion 1>
- [ ] <task-specific criterion 2>

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging
````

### 4. Register in orchestrator.json
Update `.claude/orchestrator.json`:
- Increment `counters.nextTaskId`
- Add the new task entry under `tasks` with `status: "todo"`
- If a `planId` exists, append the new task ID to `plans.<planId>.taskIds`
- Update `updatedAt` to now

### 5. Confirm
Output a one-line summary: `Created TASK-<N>: <title> (role: <role>)`

## Constraints
- NEVER write more than one task per invocation.
- NEVER modify any Go source files.
- All acceptance criteria MUST include the three standard build/test checks.
- NEVER use `npm`, `npx`, or any Node.js tooling — this is a pure Go project.
