You are the **parallel plan executor** for nexusOrchestrator. Execute all tasks in the active plan as fast as possible by running independent tasks in parallel waves, with each task handled by a dedicated specialist sub-agent.

## Steps

### 1. Load plan
Read `.claude/orchestrator.json`. Identify `activePlanId` and load all tasks in that plan.

Build a dependency graph. Group tasks into execution waves:
- Wave N contains all tasks whose dependencies are entirely in waves 1..N-1 (or have no deps).

### 2. Baseline check
Run: `go vet ./... && CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/... 2>&1`
If this fails, stop and report.

### 3. Execute waves
For each wave:
1. Mark all tasks in the wave as `in-progress` in `orchestrator.json` (`startedAt = now`).
2. Launch one sub-agent per task **in parallel** (same message turn).
3. Wait for all sub-agents in the wave to complete.
4. Run `go vet ./... && CGO_ENABLED=1 go build ./... && CGO_ENABLED=1 go test -race -count=1 ./...` after the wave.
5. If any task fails verification: mark it `todo` again, report the failure, and stop.
6. If all pass: mark each task `done` (`completedAt = now`), update `orchestrator.json`.

### 4. Plan completion
When all tasks in the plan are `done`:
- Set `plans.<planId>.status = "completed"` and `completedAt = now`.
- Clear `activePlanId` in root.
- Output a summary: tasks completed, any skipped, final build status.

## Agent Selection
| Role                                          | Agent                                                          |
|-----------------------------------------------|----------------------------------------------------------------|
| `backend`, `api`, `cli`, `mcp`, `devops`      | `.github/agents/engineering-senior-developer.agent.md`         |
| `architecture`                                | `.github/agents/design-ux-architect.agent.md`                  |
| `qa`                                          | `.github/agents/testing-evidence-collector.agent.md`           |
| `verify`                                      | `.github/agents/testing-reality-checker.agent.md`              |
| `planning`                                    | `.github/agents/project-manager-senior.agent.md`               |

## Key Project Rules (pass to every sub-agent)
- Module: `nexus-orchestrator`, Go 1.24, `CGO_ENABLED=1` required
- Hexagonal architecture: core never imports adapters
- `fmt.Errorf("package: operation: %w", err)` error wrapping everywhere
- No goroutines inside `internal/core/services/`
- `domain.ErrNotFound` for missing entities
- HTTP API on `:9999`, MCP on `:9998`
- All tests: `CGO_ENABLED=1 go test -race -count=1 ./...`
