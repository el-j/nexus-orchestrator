# PLAN-002: Orchestrator Hardening + Wails 2 GUI + Writeback System

**Status:** active  
**Created:** 2026-03-09  
**Goal:** Harden the orchestrator against data-races and crashes; build a Wails 2 cross-platform GUI; implement a two-way writeback system so external projects that submit tasks to nexusOrchestrator have their local `.claude/orchestrator.json` updated on completion.

---

## Architecture Overview

```
inbound adapters → core services → ports ← outbound adapters
     CLI / HTTP API / MCP / Wails              SQLite / LLM / fs_writeback
```

New in PLAN-002:
- `domain.Task` gains optional `SourceProjectPath`, `SourceTaskID`, `SourcePlanID` fields
- New port: `WritebackClient`
- New outbound adapter: `internal/adapters/outbound/fs_writeback/`
- New frontend: `frontend/src/` — React 18 + TypeScript + Vite + Tailwind CSS
- New Wails app.go methods: `GetAllTasks`, `GetSession`, `ClearSession`, `GetStats`
- New `.claude/commands/`: `push-to-nexus.md`, `sync-from-nexus.md`

---

## Failsafe Audit Summary (22 Issues)

| ID | Severity | Location | Issue |
|----|----------|----------|-------|
| A1 | CRITICAL | orchestrator.go:25-38 | PROCESSING tasks orphaned on daemon restart |
| A2 | CRITICAL | orchestrator.go:25-38 | QUEUED tasks in DB not recovered on startup |
| B1 | HIGH | orchestrator.go:81-91 | CancelTask: memory removal before DB update |
| B2 | HIGH | orchestrator.go:108 | UpdateStatus(PROCESSING) silently ignored with `_ =` |
| D1 | HIGH | orchestrator.go:111 | All `repo.UpdateStatus()` calls swallow errors via `_ =` |
| D2 | HIGH | orchestrator.go:106-112 | LLM unavailable → infinite re-queue loop, no backoff |
| D3 | HIGH | orchestrator.go:122-129 | AppendMessage failures ignored; session history silently corrupts |
| E1 | HIGH | orchestrator.go:93-95 | Stop() panics on double-close (no sync.Once) |
| E2 | HIGH | worker goroutine | Not context-aware; blocks on LLM I/O during shutdown |
| B3 | MEDIUM | orchestrator.go:47 | Unbounded queue (DoS via HTTP/MCP) |
| B5/F1 | MEDIUM | orchestrator.go:46 | ProjectPath not normalized (filepath.Clean) |
| C1 | MEDIUM | orchestrator.go:47-56 | Race window: repo.Save called outside mu.Lock |
| C2 | MEDIUM | various | Error returns ignored throughout service |
| C3 | MEDIUM | orchestrator.go:57 | Lock released too early in SubmitTask |

**Addressed by:** TASK-013 (A1, A2, E1, D1, D2) + TASK-014 (B3, B5/F1, C1, D2 backoff)

---

## Tech Stack Decisions

| Layer | Choice | Reason |
|-------|--------|--------|
| Frontend framework | Vue 3 (Composition API + `<script setup>`) | Excellent Wails 2 support; reactive by default |
| UI component library | PrimeVue 4 with Aura theme (dark) | Production-ready, Tailwind-compatible, rich data table |
| Build tool | Vite 5 + `@tailwindcss/vite` | Fast HMR, Wails dev server compatible |
| CSS | Tailwind CSS 4 (CSS-first config, `@import`) | No config file; `@theme` directive for custom tokens |
| Icons | `lucide-vue-next` | Consistent, tree-shakeable, Vue 3 native |
| Server state | `@tanstack/vue-query` v5 | Caching, polling, invalidation |
| Client state | Pinia v2 | Vue 3 idiomatic store |
| Routing | Vue Router 4 | SPA navigation for Wails WebView |
| Charts | `chart.js` + `vue-chartjs` | Queue/activity timeline |

---

## Execution Plan — 6 Waves

### Wave 0 — Independent (run in parallel)
| Task | Role | Description |
|------|------|-------------|
| TASK-013 | backend | Startup recovery + Stop() idempotency |
| TASK-014 | backend | Retry limits + backoff + path normalization + queue cap |
| TASK-026 | architecture | Domain: source writeback fields on Task |

### Wave 1 — API surface (after Wave 0)
| Task | Role | Description |
|------|------|-------------|
| TASK-015 | api | HTTP API: history + session management endpoints |
| TASK-029 | api | HTTP API + MCP: expose source fields in submit |

### Wave 2 — CLI + Wails binding + writeback adapter (after Wave 1)
| Task | Role | Description |
|------|------|-------------|
| TASK-016 | cli | CLI: queue submit + sessions subcommand |
| TASK-018 | backend | Wails app.go: GetAllTasks, GetSession, ClearSession, GetStats |
| TASK-027 | backend | Port + adapter: fs_writeback outbound |

### Wave 3 — Frontend scaffold + writeback plumbing (after Wave 2)
| Task | Role | Description |
|------|------|-------------|
| TASK-017 | devops | Scaffold React+TS+Tailwind Wails frontend |
| TASK-028 | backend | OrchestratorService: invoke writeback on completion |
| TASK-030 | planning | New command: push-to-nexus.md |

### Wave 4 — GUI views + sync command (after Wave 3)
| Task | Role | Description |
|------|------|-------------|
| TASK-019 | devops | GUI: Dashboard + Provider status |
| TASK-020 | devops | GUI: Task Queue page with submit drawer |
| TASK-021 | devops | GUI: Task History + Detail page |
| TASK-022 | devops | GUI: Settings page |
| TASK-031 | planning | New command: sync-from-nexus.md |

### Wave 5 — Polish + validation (after Wave 4)
| Task | Role | Description |
|------|------|-------------|
| TASK-023 | devops | System tray integration |
| TASK-024 | qa | QA: hardening tests + GUI smoke tests |
| TASK-025 | verify | Cross-platform build verification + README update |

---

## Writeback System Design

### Motivation
A project using nexusOrchestrator's `.claude/commands/` workflow submits tasks via HTTP or MCP. After the orchestrator finishes a task, the source project's `.claude/orchestrator.json` should be updated automatically: task marked `"done"`, output appended, `completedAt` set.

### Data Flow
```
[Project A] /push-to-nexus TASK-045
    → POST /api/tasks  { prompt, projectPath, sourceProjectPath="/proj/A", sourceTaskId="TASK-045", ... }
    → nexusOrchestrator processes task
    → on completion: fs_writeback reads /proj/A/.claude/orchestrator.json
                     updates tasks.TASK-045.status = "done"
                     appends output to TASK-045.md
                     saves file back

[Project A] /sync-from-nexus
    → GET /api/tasks?sourceProjectPath=/proj/A&status=completed
    → for each result: update local orchestrator.json + task file
```

### WritebackClient port (pseudocode)
```go
type WritebackClient interface {
    WriteBack(ctx context.Context, wb WritebackPayload) error
}

type WritebackPayload struct {
    SourceProjectPath string
    SourceTaskID      string
    SourcePlanID      string   // optional
    NexusTaskID       string
    Status            string   // "completed" | "failed"
    Output            string   // LLM output written to files
    CompletedAt       time.Time
}
```

### fs_writeback adapter behaviour
1. Read `{SourceProjectPath}/.claude/orchestrator.json`
2. Update `tasks.{SourceTaskID}.status` → `"done"` (or `"failed"`)
3. Update `tasks.{SourceTaskID}.completedAt` → ISO 8601
4. Append `Output` snippet to `{SourceProjectPath}/.claude/tasks/{SourceTaskID}.md` under `## Nexus Output`
5. Atomically write back (write to `.tmp`, then `os.Rename`)

---

## Cross-Platform Build Matrix

| Platform | Wails target | Notes |
|----------|-------------|-------|
| macOS (arm64) | `wails build -platform darwin/arm64` | Default dev machine |
| macOS (amd64) | `wails build -platform darwin/amd64` | Intel Macs |
| Linux (amd64) | `wails build -platform linux/amd64` | CGO needs gcc |
| Windows (amd64) | `wails build -platform windows/amd64` | CGO needs mingw64 |

Headless daemon (`cmd/nexus-daemon`) builds without Wails on all platforms via `CGO_ENABLED=1 go build`.

---

## Task Index

| Task | Title | Wave | Status |
|------|-------|------|--------|
| TASK-013 | Startup recovery + Stop() idempotency | 0 | todo |
| TASK-014 | Retry limits + backoff + path norm + queue cap | 0 | todo |
| TASK-015 | HTTP API: history + session endpoints | 1 | todo |
| TASK-016 | CLI: queue submit + sessions | 2 | todo |
| TASK-017 | Frontend scaffold (React+TS+Tailwind) | 3 | todo |
| TASK-018 | app.go: GetAllTasks / GetSession / GetStats | 2 | todo |
| TASK-019 | GUI: Dashboard + Provider status | 4 | todo |
| TASK-020 | GUI: Task Queue + submit drawer | 4 | todo |
| TASK-021 | GUI: Task History + Detail | 4 | todo |
| TASK-022 | GUI: Settings | 4 | todo |
| TASK-023 | System tray | 5 | todo |
| TASK-024 | QA: hardening + GUI smoke tests | 5 | todo |
| TASK-025 | Cross-platform build + README | 5 | todo |
| TASK-026 | Domain: source writeback fields | 0 | todo |
| TASK-027 | Port + fs_writeback adapter | 2 | todo |
| TASK-028 | Orchestrator: invoke writeback | 3 | todo |
| TASK-029 | HTTP API + MCP: source fields | 1 | todo |
| TASK-030 | command: push-to-nexus.md | 3 | todo |
| TASK-031 | command: sync-from-nexus.md | 4 | todo |
