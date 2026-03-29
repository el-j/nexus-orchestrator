# PLAN-057: Mission Control Dashboard & UI Consolidation

**Status:** active  
**Created:** 2026-03-29  
**Priority:** 1 — UX Overhaul

## Problem Statement

The nexusOrchestrator GUI has 11 sidebar views that present ~5 distinct data entities in overlapping, entity-per-page layouts. A power user running multiple AI agents across multiple projects must visit 5+ pages to understand system state. Key problems:

### 🔴 Critical: View Redundancy (~60% aggregate overlap)

| Overlap Pair                                                | Redundancy | Detail                                                |
| ----------------------------------------------------------- | ---------- | ----------------------------------------------------- |
| AI Sessions ↔ AI Agents                                     | 100%       | AI Agents = AI Sessions + Discovered Agents bolted on |
| Discovery ↔ Providers (Discovered section)                  | 100%       | Identical `DiscoveredProvidersPanel` rendered in both |
| Providers (AI Coding Tools) ↔ AI Agents (Discovered Agents) | 90%        | Same `DiscoveredAgent[]` data, different card layouts |
| Dashboard (ProviderStatus) ↔ Providers (Active)             | 80%        | Same `ProviderInfo[]`, compact pills vs full cards    |
| Live AI ↔ Projects                                          | 70%        | Same `/api/activities/timeline`, different grouping   |

### 🔴 Critical: Missing Data in Every View

- **Task.Instruction** (the most important field) hidden behind detail drawers, never shown inline
- **Provider context window sizes** not displayed (LLMClient.ContextLimit() exists but isn't surfaced)
- **Token usage, costs, rate limits, latency** — zero visibility
- **Provider health age** — no "last checked Xm ago" indicator
- **Task duration** — no time-to-completion anywhere

### 🟠 High: Session Detection Gap

Two VS Code Copilot sessions running but invisible. Root cause: Copilot runs in-process in VS Code — no filesystem artifact, port, or process flag for probing. Only appears if the MCP `register_session` tool is actively called. No passive detection exists.

### 🟡 Medium: Provider Card Duplication

ProvidersView "AI Coding Tools" section shows 10+ duplicate "Claude CLI" cards. Root cause: `probeClaudeSubAgents()` creates one `DiscoveredAgent` per `.jsonl` session file in `~/.claude/projects/`, each with a unique ID so dedup doesn't collapse them.

### 🟡 Medium: No Unified Dashboard

No single view showing all active work, all agents, all provider states, and recent completions in one dense layout. Users must visit Task Queue + Live AI + Providers + AI Sessions to get full picture.

## Design: Mission Control Dashboard

```
┌──────────────────────────────────────────────────────────────────┐
│  MISSION CONTROL                              ⟳ 3s ago │ ⚙     │
├──────────────────────────────────────────────────────────────────┤
│  ┌─── STATUS BAR ─────────────────────────────────────────────┐ │
│  │ 🟢 2 Agents  │ 🟡 1 Queued  │ 🔴 3 Providers Down        │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                  │
│  ┌─── ACTIVE WORK (unified lifecycle) ────────────────────────┐ │
│  │ ▸ PROCESSING T-407  "Add retry logic…"   claude/sonnet 12s │ │
│  │ ▸ QUEUED     T-408  "Update API docs…"   any · swarm       │ │
│  │ ▸ DRAFT      T-409  "Cost tracking…"     p2  [Promote]     │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                  │
│  ┌─── AGENTS ──────────────┐  ┌─── PROVIDERS ────────────────┐ │
│  │ 🟢 claude-sonnet-4-6   │  │ 🔴 LM Studio   :1234        │ │
│  │   swarm · 14.2k tok    │  │    ctx: 32k · 300s timeout   │ │
│  │ 🟢 copilot-vscode      │  │ 🔴 Ollama      :11434       │ │
│  │   nexusOrch · 8.1k tok │  │    ctx: 128k · 120s timeout  │ │
│  │ 🟡 claude-code         │  │ 🟢 Antigravity  :4315       │ │
│  │   portolio · idle 4m   │  │    ctx: 200k · active        │ │
│  └─────────────────────────┘  └───────────────────────────────┘ │
│                                                                  │
│  ┌─── RECENT (last 10) ──────────────────────────────────────┐ │
│  │ ✅ T-386 "Archive PLAN-056"          23m ago  0.8s  1.2k  │ │
│  │ ❌ T-385 "Build wasm target"         1h ago   FAILED      │ │
│  └────────────────────────────────────────────────────────────┘ │
│  ┌─── SUBMIT ────────────────────────────────────────── [▾] ──┐ │
│  │ [Instruction...] [Project ▾] [Target] [Provider ▾]  [Send] │ │
│  └────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

## Design: Sidebar Consolidation (11 → 6 views)

```
BEFORE (11)              AFTER (6)
───────────              ─────────
Task Queue  ─────┐
Backlog     ─────┤──→  🏠 Mission Control (unified task + status)
History     ─────┘

Live AI     ─────┐
AI Sessions ─────┤──→  🤖 Agents (merged live view + sessions)
AI Agents   ─────┘

Providers   ─────┐
Discovery   ─────┤──→  🔌 Providers (merged infra + config)
Settings    ─────┘

Projects    ──────→  📁 Projects (kept, enriched)
Plans       ──────→  📋 Plans (kept, made actionable)
(new)       ──────→  ⚙ Settings (extracted config-only)
```

## Tasks

| Task     | Wave | Role     | Description                                                                                                                      |
| -------- | ---- | -------- | -------------------------------------------------------------------------------------------------------------------------------- |
| TASK-407 | 1    | backend  | Enrich `ProviderInfo` with `ContextLimit`, `TimeoutSec`, `LastChecked`, `ConsecutiveFails` fields                                |
| TASK-408 | 1    | backend  | Dedup `probeClaudeSubAgents`: fold all session files into one `DiscoveredAgent` per project dir with session count               |
| TASK-409 | 1    | backend  | Add `DurationMs` computed field to Task (from CreatedAt → CompletedAt) in HTTP/Wails responses                                   |
| TASK-410 | 2    | frontend | Build `MissionControlView.vue` — status bar + unified task list (DRAFT/QUEUED/PROCESSING/COMPLETED) with instruction text inline |
| TASK-411 | 2    | frontend | MissionControlView agents panel — live sessions with model, token count, project, idle time                                      |
| TASK-412 | 2    | frontend | MissionControlView providers panel — compact cards with context limit, timeout, last-checked age, health dot                     |
| TASK-413 | 2    | frontend | MissionControlView recent completions — last 10 terminal tasks with instruction, duration, token count                           |
| TASK-414 | 2    | frontend | MissionControlView submit form — collapsible inline task submission                                                              |
| TASK-415 | 3    | frontend | Delete `DiscoveryView.vue`, merge into `ProvidersView.vue` (already has `DiscoveredProvidersPanel`)                              |
| TASK-416 | 3    | frontend | Merge AI Sessions + AI Agents → unified `AgentsView.vue` with registered sessions + discovered agents in one page                |
| TASK-417 | 3    | frontend | Delete `AISessionsView.vue` and `AIAgentsView.vue`, wire `AgentsView.vue` into sidebar                                           |
| TASK-418 | 3    | frontend | Merge Settings configuration into `ProvidersView.vue` as collapsible "Configuration" section                                     |
| TASK-419 | 3    | frontend | Update `AppSidebar.vue` — restructure from 11 → 6 items with new icons                                                           |
| TASK-420 | 4    | frontend | Show `task.instruction` truncated inline in all task lists (Dashboard/History/Backlog components)                                |
| TASK-421 | 4    | frontend | Add "last refreshed Xm ago" indicator to every view header                                                                       |
| TASK-422 | 4    | frontend | Global SSE connection in App.vue — single EventSource shared across all views via composable                                     |
| TASK-423 | 5    | frontend | Live AI → merge into AgentsView as "Activity" tab (timeline grouped by session)                                                  |
| TASK-424 | 5    | frontend | History → merge into Mission Control as expandable "All History" section                                                         |
| TASK-425 | 5    | frontend | Backlog → merge into Mission Control as filtered view of DRAFT/BACKLOG tasks                                                     |
| TASK-426 | 6    | backend  | Track provider response latency (p50) in discovery service health checks                                                         |
| TASK-427 | 6    | backend  | Detect and surface 429 rate-limit state per provider (from last error)                                                           |
| TASK-428 | 6    | frontend | ProvidersView — show latency, rate-limit badge, circuit-breaker state on provider cards                                          |
| TASK-429 | 7    | testing  | Vitest: MissionControlView renders all 5 panels with mock data                                                                   |
| TASK-430 | 7    | testing  | Vitest: AgentsView shows merged sessions + discovered agents                                                                     |
| TASK-431 | 7    | testing  | Vitest: ProvidersView shows enriched provider info (context, latency, health age)                                                |
| TASK-432 | 7    | testing  | Go tests: ProviderInfo enrichment, DiscoveredAgent dedup, DurationMs computation                                                 |

## Wave Plan

```
Wave 1 (TASK-407..409)    — Backend enrichment: ProviderInfo, agent dedup, task duration
Wave 2 (TASK-410..414)    — Mission Control view: all 5 panels
Wave 3 (TASK-415..419)    — View consolidation: delete redundancy, merge, sidebar restructure
Wave 4 (TASK-420..422)    — Cross-cutting UX: instruction text, freshness indicators, global SSE
Wave 5 (TASK-423..425)    — Final merges: Live AI → Agents tab, History/Backlog → Mission Control
Wave 6 (TASK-426..428)    — Provider observability: latency, rate limits, circuit-breaker
Wave 7 (TASK-429..432)    — Test coverage for all new/changed views + backend
```

Wave 1 is prerequisite for all UI work.
Waves 2–4 can proceed in parallel after Wave 1.
Wave 5 depends on Waves 2+3.
Wave 6 is independent (backend only, can run with any wave).
Wave 7 runs last.

## Success Criteria

- [ ] Single "Mission Control" view shows: task lifecycle (DRAFT→COMPLETED), agents, providers, recent completions
- [ ] Task.Instruction text visible inline in all task lists without opening drawer
- [ ] Provider cards show context window size, timeout, last-checked age
- [ ] DiscoveryView deleted — content lives in ProvidersView
- [ ] AI Sessions + AI Agents merged into single AgentsView
- [ ] Sidebar has ≤6 items (down from 11)
- [ ] No duplicate "Claude CLI" cards in any view
- [ ] Global SSE connection — navigating between views doesn't drop real-time updates
- [ ] Every panel shows "last refreshed Xm ago"
- [ ] Vitest coverage for all new views
- [ ] Go tests for ProviderInfo enrichment + agent dedup + DurationMs

## Key Files

| Layer     | File                                                      | Change                             |
| --------- | --------------------------------------------------------- | ---------------------------------- |
| Ports     | `internal/core/ports/ports.go`                            | `ProviderInfo` struct enrichment   |
| Discovery | `internal/core/services/discovery.go`                     | Populate new ProviderInfo fields   |
| Scanner   | `internal/adapters/outbound/sys_scanner/agent_scanner.go` | Dedup claude session agents        |
| Domain    | `internal/core/domain/task.go`                            | `DurationMs` computed field        |
| Frontend  | `frontend/src/views/MissionControlView.vue` (NEW)         | Unified dashboard                  |
| Frontend  | `frontend/src/views/AgentsView.vue` (NEW)                 | Merged sessions + agents           |
| Frontend  | `frontend/src/views/ProvidersView.vue`                    | Absorb Discovery + Settings config |
| Frontend  | `frontend/src/views/DiscoveryView.vue`                    | DELETE                             |
| Frontend  | `frontend/src/views/AISessionsView.vue`                   | DELETE                             |
| Frontend  | `frontend/src/views/AIAgentsView.vue`                     | DELETE                             |
| Frontend  | `frontend/src/components/AppSidebar.vue`                  | 11 → 6 items                       |
| Frontend  | `frontend/src/composables/useGlobalSSE.ts` (NEW)          | Shared EventSource                 |
| Tests     | `frontend/src/test/`                                      | 3 new view spec files              |
| Tests     | `internal/core/services/` + adapters                      | Backend enrichment tests           |
