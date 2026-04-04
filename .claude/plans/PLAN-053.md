# PLAN-053 — GUI Data Quality & Session Visibility Fixes

## Goal

Fix three integration bugs visible in the nexusOrchestrator GUI:

1. Plans view shows "0 discovered across 0 projects" despite 52+ plans existing
2. Stale AI sessions persist as "idle" indefinitely (12 claude sessions from aether-swarm)
3. No bridge from discovered agents (Copilot, Continue) to AI Sessions view

## Root Causes

### Plans View (Bug)

- Frontend `DiscoveredPlansView.vue` sends empty `projectPath` to backend
- Backend scans cwd, stores files with real project path, but queries `WHERE project_path = ''` → 0 results
- Fix: Use `currentProject` from shared state; backend returns all if path is empty

### Stale Sessions (Bug)

- `runSessionCleanup` marks disconnected at 5 min, purges only at 2 hours
- ActivityService `checkIdleDisconnect` marks idle at 5 min, disconnected at 2 hours
- 12 aether-swarm sessions linger as "idle" for 2 hours before cleanup
- Fix: Reduce disconnect timeout, add "Purge All Disconnected" button already exists via MCP tool

### Agent-to-Session Bridge (Missing Feature)

- `ScanAgents()` finds Copilot/Continue via vscode-extension detection
- But no automatic conversion from `DiscoveredAgent → AISession`
- Only `ActivityService.bridgeSession()` creates sessions, and only from observed activity file reads
- Fix: Bridge discovered agents to sessions when activity is also detected

## Tasks

| Task ID  | Title                                         | File(s)                               | Depends | Status |
| -------- | --------------------------------------------- | ------------------------------------- | ------- | ------ |
| TASK-364 | Fix Plans view: use currentProject            | frontend DiscoveredPlansView.vue      | —       | todo   |
| TASK-365 | Fix backend: return all plans when path empty | plan_file_repo.go, session_service.go | —       | todo   |
| TASK-366 | Reduce stale session disconnect timeout       | activity_service.go                   | —       | todo   |
| TASK-367 | Add auto-scan on Plans view mount             | useDiscoveredPlans.ts                 | 364     | todo   |

## Wave Plan

### Wave 1 (parallel — independent fixes)

- TASK-364: Frontend Plans view fix
- TASK-365: Backend Plans query fix
- TASK-366: Reduce disconnect timeout

### Wave 2 (depends on Wave 1)

- TASK-367: Auto-scan on mount to ensure fresh data
