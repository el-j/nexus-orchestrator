# PLAN-054 — Live Views Overhaul: Plans Hierarchy, AI Tools Visibility, Activity Structure

## Goal

Fix four UI quality issues visible in the running GUI:

1. Plans view shows a flat file list with no project overview or plan grouping
2. Providers view has no way to see GitHub Copilot / VS Code / Continue as AI Coding Tools
3. Live AI view activities are a flat unsorted feed with stale timestamps — no session context
4. Task History shows only 2 completed tasks with no breakdown of pipeline state

## Investigation Summary

### Plans View (TASK-368 + TASK-372)

- 324 `.claude/tasks/*.md` files dumped in a flat list — no grouping by parent plan
- `orchestrator.json` (kind=nexus) not visually distinguished from individual task files
- Summary field on each `TASK-NNN.md` starts with `# TASK-NNN: ... **Plan:** PLAN-052 | ...`
  → Can parse parent plan from summary via regex
- Fix: Group files by parent plan, show nexus orchestrator.json as "Project Brain" header card
- Backend: Enrich nexus-kind summary with plan/task counts extracted from orchestrator.json

### Providers View (TASK-369)

- GitHub Copilot and Continue are `DiscoveredAgent` not `DiscoveredProvider`
- Backend already exposes them via `GET /api/ai-sessions/discovered`
- These agents are completely invisible from Providers view
- Fix: Add "AI Coding Tools" section to Providers view calling existing endpoint

### Live AI (TASK-370)

- Activities have a `sessionId` field in their UUID (`claude-{sessionId}-{uuid}`)
- `AIActivity` struct has `SessionID string` field
- Frontend composable ignores it — activities shown as flat chronological list
- User/Responding event pairs not visually grouped
- Fix: Group activities by sessionId into conversation threads with turn pairs

### Task History (TASK-371)

- Only shows COMPLETED/FAILED/CANCELLED (by design), but 2-task display looks broken
- No summary of the broader pipeline state (how many queued, in-progress, etc.)
- Fix: Add status distribution summary bar at the top of History view

## Tasks

| Task ID  | Title                                              | Role     | Depends | Status |
| -------- | -------------------------------------------------- | -------- | ------- | ------ |
| TASK-368 | Plans view: project brain header + group by plan   | frontend | 372     | todo   |
| TASK-369 | Providers view: AI Coding Tools section            | frontend | —       | todo   |
| TASK-370 | Live AI: session grouping + conversation threads   | frontend | —       | todo   |
| TASK-371 | Task History: status pipeline summary bar          | frontend | —       | todo   |
| TASK-372 | Plan scanner: enrich nexus-kind with project stats | backend  | —       | todo   |

## Wave Plan

### Wave 1 (parallel — independent)

- TASK-369: Providers AI Coding Tools section
- TASK-370: Live AI session grouping
- TASK-371: Task History summary bar
- TASK-372: Backend plan scanner enrichment

### Wave 2 (depends on Wave 1 — 372 must be done for 368 to be richest)

- TASK-368: Plans view project brain + plan grouping
