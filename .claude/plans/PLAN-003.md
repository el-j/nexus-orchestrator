---
id: PLAN-003
title: Dogfood — implement PLAN-002 using nexusOrchestrator itself
status: active
createdAt: 2026-03-09T15:00:00.000Z
---

# PLAN-003: Dogfood — Orchestrate PLAN-002 via nexusOrchestrator

## Goal

Use nexusOrchestrator to implement its own PLAN-002 backlog. This directly validates the orchestrator pipeline end-to-end: daemon starts, LLM processes Go implementation tasks, outputs are written to source files, and progress is visible in an embedded web dashboard.

## Prerequisite Audit

| Item | Status | Notes |
|------|--------|-------|
| `CGO_ENABLED=1 go build ./...` | ✅ passes | No blocking issues |
| daemon starts on :63987/:63988 | ✅ works | session isolation, MCP |
| LM Studio / Ollama | ⚠️ required | provider must be online |
| JSON struct tags on domain types | ❌ missing | breaks web UI + nexus-submit |
| Web dashboard | ❌ missing | needed for live progress |
| nexus-submit tool | ❌ missing | needed to feed TASK files to daemon |
| SSE events | ❌ missing | needed for live dashboard |

## Tech Decisions

- **Dashboard**: pure Go `text/template` embedded in HTTP binary — no npm, no bundler, no external assets
- **Polling fallback**: dashboard JS polls `/api/tasks` every 2 s until SSE is live (TASK-035)
- **nexus-submit**: standalone Go binary `cmd/nexus-submit/` reading TASK-NNN.md files
- **Domain tags**: add `json:"camelCase"` tags to domain types, zero behaviour change, backward compat

## Execution Waves

| Wave | Tasks | Description |
|------|-------|-------------|
| 1 | TASK-032, TASK-033 (parallel) | JSON tags + nexus-submit tool |
| 2 | TASK-034, TASK-035 (parallel) | Web dashboard + SSE events |
| 3 | TASK-036, TASK-037 (parallel) | Dogfood scripts + integration tests |
| 4 | TASK-038 | Verification + final README |

## Task Index

| ID | Title | Deps |
|----|-------|------|
| TASK-032 | JSON struct tags on domain types | — |
| TASK-033 | cmd/nexus-submit: feed TASK-NNN.md files to daemon | TASK-032 |
| TASK-034 | Embedded web dashboard at GET /ui | TASK-032 |
| TASK-035 | SSE events endpoint + OrchestratorService event fan-out | — |
| TASK-036 | Dogfood scripts + .claude/commands/dogfood-plan002.md | TASK-033 |
| TASK-037 | Integration tests: daemon + mock LLM + real SQLite | TASK-033, TASK-034 |
| TASK-038 | PLAN-003 verification + README update | TASK-034, TASK-036 |

## Dogfood Loop

Once PLAN-003 tasks are implemented:

```
nexus-daemon &                # start orchestrator
open http://localhost:63987/ui  # open dashboard

# Submit PLAN-002 tasks to process themselves
nexus-submit --task-file .claude/tasks/TASK-013.md \
             --target internal/core/services/orchestrator.go \
             --context internal/core/services/orchestrator.go,internal/core/ports/ports.go

nexus-submit --task-file .claude/tasks/TASK-015.md \
             --target internal/adapters/inbound/httpapi/server.go \
             --context internal/adapters/inbound/httpapi/server.go
```

The LLM generates the Go implementation; the dashboard shows progress in real time.

## Acceptance Criteria (PLAN-003 complete)

- [ ] `CGO_ENABLED=1 go build ./... ` exits 0, including `cmd/nexus-submit`
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `GET http://localhost:63987/ui` returns 200 with valid HTML
- [ ] `nexus-submit` binary built; submitting a TASK-NNN.md returns a task ID
- [ ] Dashboard shows live queue updates (proves HTTP polling path works)
- [ ] `scripts/dogfood-plan002.sh` exists and is executable
