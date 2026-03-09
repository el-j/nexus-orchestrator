# Orchestrator Index — nexusOrchestrator

Full history of completed plans for the nexusOrchestrator project.

---

## Project

**nexusOrchestrator** is a Go 1.24 application (module: `nexus-ai`) with hexagonal architecture.

| Property | Value |
|----------|-------|
| Language | Go 1.24, CGO required (`mattn/go-sqlite3`) |
| HTTP API | `github.com/go-chi/chi/v5` on `:9999` |
| MCP server | JSON-RPC 2.0 inbound adapter on `:9998` |
| GUI | Wails v2 desktop app |
| CLI | Cobra-based thin HTTP client |
| Build | `CGO_ENABLED=1 go build ./...` |
| Test | `CGO_ENABLED=1 go test -race -count=1 ./...` |
| Lint | `go vet ./...` |
| Error convention | `fmt.Errorf("package: operation: %w", err)` |

### Architecture layers

```
internal/adapters/inbound/   →  internal/core/services/  →  internal/core/ports/
                                                          ←  internal/adapters/outbound/
```

- `internal/core/domain/` — pure domain types (`Task`, `TaskStatus`, `Session`, `Message`)
- `internal/core/ports/` — Go interfaces only (`LLMClient`, `TaskRepository`, `SessionRepository`, `FileWriter`, `Orchestrator`)
- `internal/core/services/` — business logic; depends only on ports; no goroutines here
- `internal/adapters/inbound/` — CLI (Cobra), HTTP API (chi), MCP (JSON-RPC 2.0), Wails GUI, system tray
- `internal/adapters/outbound/` — LM Studio, Ollama, SQLite, filesystem

### Agent files

| Role | Agent file |
|------|-----------|
| `backend`, `api`, `cli`, `mcp`, `devops` | `.github/agents/engineering-senior-developer.agent.md` |
| `architecture` | `.github/agents/design-ux-architect.agent.md` |
| `qa` | `.github/agents/testing-evidence-collector.agent.md` |
| `verify` | `.github/agents/testing-reality-checker.agent.md` |
| `planning` | `.github/agents/project-manager-senior.agent.md` |

---

## Active Plan

**PLAN-001** — Add MCP server inbound adapter and per-project session isolation to nexusOrchestrator

### Task Summary

| ID | Title | Role | Status | Dependencies |
|----|-------|------|--------|--------------|
| TASK-001 | Domain: Session + Message types | architecture | todo | — |
| TASK-002 | Ports: SessionRepository + Chat() on LLMClient | architecture | todo | TASK-001 |
| TASK-003 | SQLite: session repository impl + migration | backend | todo | TASK-002 |
| TASK-004 | LLM adapters: implement Chat() for LMStudio + Ollama | backend | todo | TASK-002 |
| TASK-005 | OrchestratorService: session-isolated task processing | backend | todo | TASK-003, TASK-004 |
| TASK-006 | MCP server inbound adapter (JSON-RPC 2.0 + SSE) | mcp | todo | TASK-002 |
| TASK-007 | Wire MCP server into all three entry points | devops | todo | TASK-006 |
| TASK-008 | Tests: SQLite session repository | qa | todo | TASK-003 |
| TASK-009 | Tests: OrchestratorService session isolation | qa | todo | TASK-005 |
| TASK-010 | Tests: MCP server handler | qa | todo | TASK-006 |
| TASK-011 | Rewrite .github/agents for nexusOrchestrator | planning | done | — |
| TASK-012 | Update copilot-instructions.md + README | planning | todo | TASK-005, TASK-006 |

---

## Completed Plans

| ID | Goal | Tasks | Status | Completed |
|----|------|-------|--------|-----------|

*(No completed plans yet — PLAN-001 is active)*

---

## Counters

- **Next Task ID**: 13
- **Next Plan ID**: 2

---

## .claude System Conventions

- `orchestrator.json` — slim active registry; only non-done tasks + current plan + counters
- `.claude/tasks/TASK-NNN.md` — one file per task with YAML front-matter + implementation spec
- `.claude/plans/PLAN-NNN.md` — archived completed plans (written by `/archive-plan`)
- `.claude/commands/*.md` — slash commands invoked by Claude agents
- Completed tasks are removed from `orchestrator.json` when a plan is archived; the `.md` files are kept permanently as a record
