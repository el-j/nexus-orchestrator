---
id: PLAN-038
goal: "Enable automatic tracking of tasks executed by external AI agents (Copilot, Claude, etc.) via task claiming, progress reporting, and session-task binding"
status: active
createdAt: 2026-03-13T16:00:00.000Z
---

## Problem

The nexusOrchestrator can register AI agent sessions (Copilot, Claude Desktop, etc.) and maintain heartbeats, but there is **no mechanism to track which tasks are being executed by which agent**. The existing `AISession.RoutedTaskIDs` field is never populated, the `Task` domain has no reference to the owning session, and external agents have no MCP tools to claim tasks or report progress/completion. The VS Code extension only announces its presence — it never reports what Copilot is actually doing.

## Root Cause Analysis

| Gap | Current State | Impact |
|-----|---------------|--------|
| No `AISessionID` on Task | Task has no session reference | Cannot answer "who is working on this task?" |
| No `ClaimTask` service method | Agents can only submit/query tasks | Cannot self-assign queued work |
| No `UpdateTaskStatus` from external | Only internal worker transitions status | External agents can't report completion |
| `RoutedTaskIDs` never populated | Field exists in DB, always empty | Session→task relationship is dead data |
| No MCP tools for claim/progress | Only `register_session` + `get_ai_sessions` | Agents have no protocol for task ownership |
| Extension is passive | Only heartbeats, no activity monitoring | Orchestrator blind to Copilot's actual work |

## Fix Strategy — 8 Tasks in 5 Waves

### Wave 1 — Domain & Ports (TASK-251)
Add `AISessionID` field to `Task` domain, add `ClaimTask` + `UpdateTaskStatus` methods to `Orchestrator` port interface.

### Wave 2 — Persistence (TASK-252)
Add `ai_session_id` column to tasks table, implement `AppendRoutedTaskID` on AISession repo, add `GetTasksBySessionID` query.

### Wave 3 — Service Logic (TASK-253)
Implement `ClaimTask` and `UpdateTaskStatus` in `OrchestratorService` with proper state machine guards: only QUEUED→PROCESSING via claim, and PROCESSING→COMPLETED/FAILED via external status update. Populate `RoutedTaskIDs` on claim.

### Wave 4 — Inbound Adapters (TASK-254, TASK-255)
- TASK-254: MCP — Add `claim_task` and `update_task_status` JSON-RPC tools
- TASK-255: HTTP — Add `POST /api/tasks/{id}/claim` and `PUT /api/tasks/{id}/status` endpoints + `GET /api/ai-sessions/{id}/tasks`

### Wave 5 — Extension + Testing + Verification (TASK-256, TASK-257, TASK-258)
- TASK-256: VS Code extension — Enhance sessionMonitor to auto-claim project tasks
- TASK-257: Unit + integration tests for the full claim/status pipeline
- TASK-258: E2E verification of auto-tracking flow

## Tasks
- TASK-251: Architecture — Domain model + port contract for task-session binding
- TASK-252: Backend — SQLite migration + repo methods for task-session persistence
- TASK-253: Backend — OrchestratorService ClaimTask + UpdateTaskStatus implementation
- TASK-254: MCP — Add claim_task and update_task_status tools
- TASK-255: API — HTTP endpoints for task claim, status update, session-task query
- TASK-256: Extension — Enhance sessionMonitor with auto-claim and progress reporting
- TASK-257: QA — Unit + integration tests for task-session tracking pipeline
- TASK-258: Verify — E2E validation of automatic task tracking
