# PLAN-041 — Live AI Activity Monitor + Session Hygiene

**Status**: active  
**Created**: 2026-03-13

## Context

Three inter-related user requests emerged from running the orchestrator in production:

### Q: "Is Copilot/Claude wired the same as Antigravity?"
**No.** There are two fundamentally different categories:

| Provider | Detection | Can generate tasks? | Why |
|----------|-----------|---------------------|-----|
| Antigravity, LM Studio, Ollama | sys_scanner (port) | ✅ Always | Local unauthenticated OpenAI-compat server |
| Claude Desktop, ChatGPT app | sys_scanner (process) | ❌ Not directly | Desktop GUI apps — no exposed API |
| GitHub Copilot (VS Code) | sys_scanner (process) | ❌ Without token | Remote API: `NEXUS_GITHUBCOPILOT_TOKEN` required |
| Anthropic Claude API | n/a (no local process) | ❌ Without key | Remote API: `NEXUS_ANTHROPIC_API_KEY` required |

Claude Desktop and Copilot **do** show up in the Discovery view as "running", but nexusOrchestrator
cannot call their generation APIs without credentials. This is by design — they're authentication-gated.

### Q: "AI Sessions tab keeps growing — where are sessions stored?"
Sessions are stored in **SQLite (`nexus.db`)** on the local machine (same file as tasks).
The cleanup goroutine runs every 2 minutes and marks sessions inactive >5 min as `disconnected`.
**Bug**: rows are NEVER deleted — they accumulate forever. Fix: auto-purge disconnected sessions
older than 2 hours (configurable), plus a manual `DELETE /api/ai-sessions?status=disconnected` endpoint.

### Q: "Show all ongoing AI things on my machine even when nexusOrchestrator is not in use"
The DiscoveryView ALREADY shows everything sys_scanner finds (processes, ports, CLIs).
What's missing:
1. **Ollama active model** — `/api/ps` endpoint shows which models are loaded and if a generation is in progress
2. **Unified view** — Discovery + AI Sessions are separate tabs; users want one "what's live" panel
3. **Richer process state** — process detection can return PID / start time for better "live" UX

## Solutions

| Phase | Task | Scope |
|-------|------|-------|
| 1 | TASK-264 | Add `PurgeDisconnected(ctx, olderThan)` to `AISessionRepository` port + SQLite impl |
| 1 | TASK-265 | Call purge in `runSessionCleanup()` — delete disconnected sessions older than 2h |
| 1 | TASK-266 | `DELETE /api/ai-sessions?status=disconnected` HTTP endpoint + `PurgeDisconnectedSessions` Orchestrator port method |
| 1 | TASK-267 | Wails binding + AISessionsView "Clear Disconnected" button |
| 2 | TASK-268 | Extend Ollama port probe to also call `/api/ps` → populate `ActiveModels []string` + `Generating bool` on `DiscoveredProvider` |
| 2 | TASK-269 | Extend `DiscoveredProvider` domain type with `ActiveModels`, `Generating`, `PID` fields |
| 3 | TASK-270 | `LiveActivityView.vue` — single "what's happening now" view: sys_scanner results + Nexus AI sessions in one feed, refreshing every 8s via SSE |
| 3 | TASK-271 | AppSidebar nav: add **Live Activity** as primary nav item; show live count badge; relabel Discovery as subsection |

## Architecture Notes
- `PurgeDisconnected` operates only on `disconnected` status rows → safe, never touches active/idle
- Domain type extension (`ActiveModels`, `Generating`, `PID`) on `DiscoveredProvider` is additive / backwards-compatible
- Ollama `/api/ps` probe is opportunistic — silently skipped if Ollama is not reachable
- `LiveActivityView` is a new Vue component; it does NOT replace Discovery or AI Sessions (those stay for detail work)
- The Live Activity view merges two live data sources: `useDiscovery()` + `useAISessions()`

## Success Criteria
- Disconnected AI sessions are auto-deleted after 2h
- "Clear Disconnected" button works in UI
- Ollama active model shows in Discovery panel when a model is loaded
- `LiveActivityView.vue` renders correctly with both discovered providers and AI sessions
- `go vet ./...` + `go build` pass
- AppSidebar shows Live Activity with badge
