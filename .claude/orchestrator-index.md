# Orchestrator Index — nexusOrchestrator

Full history of completed plans for the nexusOrchestrator project.

---

## Project

**nexusOrchestrator** is a Go 1.24 application (module: `nexus-orchestrator`) with hexagonal architecture.

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

**No active plan.** All 16 plans (PLAN-001–016) completed. Next plan: PLAN-017.

---

## Completed Plans

| ID | Goal | Tasks | Status | Completed |
|----|------|-------|--------|-----------|
| PLAN-001 | MCP server + per-project session isolation | TASK-001–012 | completed | 2026-03-09 |
| PLAN-003 | Dogfood PLAN-002 via nexusOrchestrator itself | TASK-032–038 | done | 2026-03-09 |
| PLAN-004 | Context-window guard (token pre-flight, StatusTooLarge) | TASK-039–043 | completed | 2026-03-10 |
| PLAN-005 | Smart multi-provider routing (FindForModel, llm_openaicompat, llm_anthropic) | TASK-044–052 | completed | 2026-03-10 |
| PLAN-006 | UI provider + model control (HTTP CRUD, Wails binding) | TASK-053–059 | completed | 2026-03-10 |
| PLAN-007 | Audit hardening (security, SQLite, concurrency, goroutine lifecycle) | TASK-060–067 | completed | 2026-03-10 |
| PLAN-008 | Comprehensive E2E + unit tests (MCP, HTTP, SSE, smoke) | TASK-068–073 | completed | 2026-03-10 |
| PLAN-009 | GitHub Pages docs site + command-aware routing (CommandType) | TASK-074–084 | completed | 2026-03-10 |
| PLAN-010 | Cross-platform release pipeline + downloads landing page | TASK-085–090 | completed | 2026-03-10 |
| PLAN-011 | Industry-grade hardening (version injection, install.sh) | TASK-091–097 | completed | 2026-03-10 |
| PLAN-012 | Semantic versioning + MIT license + zig 0.14.0 | TASK-098–103 | completed | 2026-03-10 |
| PLAN-013 | CI updated to latest action versions (gittools@v4.3.3) | TASK-104–107 | completed | 2026-03-10 |
| PLAN-014 | Unified publish.yml pipeline; fix GITHUB_TOKEN cross-trigger bug | TASK-108–111 | completed | 2026-03-10 |
| PLAN-015 | Production Node20/TypeScript GitHub Action + 24 unit tests | TASK-112–117 | completed | 2026-03-10 |
| PLAN-016 | Release pipeline finalization: delete version.yml+release.yml, CHANGELOG.md | TASK-118–121 | completed | 2026-03-11 |

---

## Counters

- **Next Task ID**: 123
- **Next Plan ID**: 17

---

## .claude System Conventions

- `orchestrator.json` — slim active registry; only non-done tasks + current plan + counters
- `.claude/tasks/TASK-NNN.md` — one file per task with YAML front-matter + implementation spec
- `.claude/plans/PLAN-NNN.md` — archived completed plans (written by `/archive-plan`)
- `.claude/commands/*.md` — slash commands invoked by Claude agents
- Completed tasks are removed from `orchestrator.json` when a plan is archived; the `.md` files are kept permanently as a record
