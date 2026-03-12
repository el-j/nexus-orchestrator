# PLAN-029: Task Queue UI Fix + AI Session Deduplication & Cleanup

**Status:** Completed  
**Completed:** 2026-03-12T12:30:00Z

## Tasks

| ID | Title | Role | Completed |
|----|-------|------|-----------|
| TASK-203 | Fix DashboardView flex wrapper: TaskQueue height chain | frontend | 2026-03-12T12:05:00Z |
| TASK-204 | Add GetAISessionByExternalID to AISessionRepository port + SQLite repo | architecture | 2026-03-12T12:10:00Z |
| TASK-205 | Orchestrator: idempotent RegisterAISession by externalId + HeartbeatAISession + stale cleanup goroutine | backend | 2026-03-12T12:15:00Z |
| TASK-206 | HTTP: POST /api/ai-sessions/{id}/heartbeat handler + HeartbeatAISession on Orchestrator port | api | 2026-03-12T12:20:00Z |
| TASK-207 | VS Code extension: heartbeat sends PATCH not re-register; update all mock orchestrators incl MCP | vscode | 2026-03-12T12:30:00Z |

## Summary

Fixed two user-reported runtime bugs and hardened the AI session lifecycle end-to-end.

### Bug 1 — Task Queue content area was completely black

**Root cause**: `DashboardView.vue` wrapped `<TaskQueue>` in `<div class="flex-1 min-h-0 overflow-auto">`. This was a *block* container, so `TaskQueue`'s own `flex-1` root had no flex parent to fill; `h-full` in the empty-state collapsed to zero height, leaving a featureless dark void.

**Fix** (`frontend/src/views/DashboardView.vue`): changed wrapper to `flex flex-col` so `TaskQueue` correctly fills the available height and the empty-state "Queue is empty" placeholder renders as intended.

### Bug 2 — 72+ AI Sessions accumulating (4 VS Code instances → unbounded row growth)

**Root causes**:
1. `SessionMonitor.heartbeat()` called `detectAndRegister()` every 60 s → always `POST /api/ai-sessions` → always a new UUID → always a new SQLite row. No deduplication by `externalId`.
2. No server-side UPSERT/idempotency on `externalId`.
3. No stale session expiry.

**Fixes**:
- `ports.AISessionRepository`: added `GetAISessionByExternalID(ctx, externalID string)` method.
- `repo_sqlite/ai_session_repo.go`: implemented `GetAISessionByExternalID` (query by `external_id` column with existing index).
- `ports.Orchestrator`: added `HeartbeatAISession(ctx, id string) error` method.
- `services/orchestrator.go` — `RegisterAISession` now idempotent: if `ExternalID != ""` and a session with that ID exists, it refreshes `LastActivity` and returns the existing record. No new row is created.
- `services/orchestrator.go` — new `HeartbeatAISession` method calls `UpdateAISessionStatus(active, now)`.
- `services/orchestrator.go` — `runSessionCleanup` goroutine runs every 2 min; marks sessions `disconnected` when `LastActivity > 5 min` old. Lifecycle tied to `stopCh`/`workerWg`.
- `httpapi/server.go`: new `POST /api/ai-sessions/{id}/heartbeat` handler wired to `HeartbeatAISession`. Returns 204 on success, 404 when session not found.
- `vscode-extension/src/nexusClient.ts`: new `heartbeatSession(id)` method calling the heartbeat endpoint.
- `vscode-extension/src/sessionMonitor.ts`: `heartbeat()` now calls `client.heartbeatSession(sessionId)`. Falls back to `detectAndRegister()` only on 404 (session was cleaned up server-side).
- All `ports.Orchestrator` mock implementations updated (`httpapi/server_test.go`, `cli/root_test.go`, `mcp/server_test.go`, `cmd/nexus-cli/main.go`) with stub `HeartbeatAISession`.

### Key files changed
- `frontend/src/views/DashboardView.vue`
- `internal/core/ports/ports.go`
- `internal/adapters/outbound/repo_sqlite/ai_session_repo.go`
- `internal/core/services/orchestrator.go`
- `internal/adapters/inbound/httpapi/server.go`
- `vscode-extension/src/nexusClient.ts`
- `vscode-extension/src/sessionMonitor.ts`
- `internal/adapters/inbound/mcp/server_test.go`
- `internal/adapters/inbound/httpapi/server_test.go`
- `internal/adapters/inbound/cli/root_test.go`
- `cmd/nexus-cli/main.go`
