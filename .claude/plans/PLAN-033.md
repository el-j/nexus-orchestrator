---
id: PLAN-033
goal: "Fix data source mismatch: BacklogView and HistoryView both call getQueue() which only returns QUEUED/PROCESSING — they never see DRAFT/BACKLOG or COMPLETED/FAILED/CANCELLED tasks"
status: completed
createdAt: 2026-03-13T01:10:00.000Z
completedAt: 2026-03-13T01:50:00.000Z
---

## Problem

Three views share `useTasks()` which calls `getQueue()` → `GetPending()` → SQL `WHERE status IN ('QUEUED','PROCESSING')`. This means:

1. **BacklogView** filters for DRAFT/BACKLOG from a dataset that only contains QUEUED/PROCESSING → always empty
2. **HistoryView** filters for COMPLETED/FAILED/CANCELLED from that same dataset → always empty
3. **DashboardView** works correctly (it wants QUEUED/PROCESSING)

## Root Cause

- `getQueue()` calls `Orchestrator.GetQueue()` → `repo.GetPending()` → only QUEUED/PROCESSING
- `getBacklog(projectPath)` exists in Wails bindings but BacklogView never calls it
- No `GetAllTasks()` method exists anywhere in the stack

## Fix Strategy

### Wave 1 — Backend: Add GetAllTasks (TASK-233)
- Add `GetAllTasks()` to `ports.Orchestrator` interface
- Implement in `OrchestratorService`
- Add `GetAllTasks()` Wails binding in `app.go`
- Add `GET /api/tasks/all` HTTP endpoint
- Add `get_all_tasks` MCP tool
- Make `GetBacklog("")` (empty projectPath) return all backlog items across projects

### Wave 2 — Frontend: Fix BacklogView + HistoryView data sources (TASK-234)
- BacklogView: Create `useBacklog` composable that calls `getBacklog()` or `getAllTasks()` with proper filtering
- HistoryView: Create `useHistory` composable that calls `getAllTasks()` and filters for terminal statuses
- Add `getAllTasks()` to `wails.ts` type wrapper
- Both views get SSE refresh on task_changed events

### Wave 3 — Verification (TASK-235)
- `go vet ./...` + `go test -race ./...`
- `vue-tsc --noEmit`
- Manual verification: create draft → navigate to BacklogView → draft visible

## Tasks
- TASK-233: Backend — Add GetAllTasks + fix GetBacklog empty-project
- TASK-234: Frontend — Fix BacklogView + HistoryView data sources
- TASK-235: Build verification
