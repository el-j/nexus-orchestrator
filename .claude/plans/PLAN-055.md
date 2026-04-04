# PLAN-055: Production Hardening & Full E2E Test Coverage

**Status:** completed  
**Created:** 2026-03-28  
**Priority:** 1 — Release Blocker

## Problem Statement

During live MCP session usage three classes of real bugs were discovered plus several UX-breaking gaps that must be resolved before shipping as a stable release:

### 🔴 Critical Bugs (discovered via live MCP usage)

1. **NO_PROVIDER deadlock** — `promote_task` immediately dispatches to the LLM worker which sets `NO_PROVIDER` if no active LLM. `CancelTask` only handles `QUEUED` → task is permanently stuck (no cancel, no retry, no escape). Confirmed via live MCP: task `d7f4180f` remains permanently in `NO_PROVIDER`.
2. **PromoteTask has no provider pre-flight check** — user gets no warning before task is queued and immediately fails. UX: queued successfully → silent NO_PROVIDER 1 second later.
3. **VS Code extension shows ALL projects' tasks globally** — `TaskQueueProvider.getChildren()` calls `getAllTasks()` with no project filter. Multi-window VS Code shows every task from every project in every window.
4. **`useActivities.ts` `projectFilter` option is accepted but silently ignored** — the API call never passes `projectPath`. LiveActivityView dropdown does nothing.
5. **`HistoryView.vue` loads all tasks then filters client-side** — performance problem and wrong semantics when task count exceeds hundreds.

### 🟡 Coverage Gaps (testing)

6. `mcp/tools_test.go` — MISSING. 31 MCP tool handlers have zero dedicated unit tests.
7. Frontend has Vitest infrastructure but ZERO test files for views or composables.
8. VS Code extension has no test for project-scoped task display.
9. No full black-box E2E test covering the complete system.

## Root Cause Analysis

| Issue                            | Root Cause                                                                                   | Fix Strategy                                        |
| -------------------------------- | -------------------------------------------------------------------------------------------- | --------------------------------------------------- |
| NO_PROVIDER deadlock             | `CancelTask` state machine only allows `QUEUED→CANCELLED`                                    | Add `NO_PROVIDER→CANCELLED` path                    |
| No pre-flight                    | `PromoteTask` calls `validateQueueAdmission` (quota/policy) but not provider discovery       | Add soft `hasProvider()` check, return warning JSON |
| Extension global tasks           | `TaskQueueProvider` uses `getAllTasks()` with no filter; `NexusClient` has params but unused | Add `workspacePath` param + VS Code settings toggle |
| Activities projectFilter ignored | `useActivities` opts accepted but URL not modified                                           | Wire `projectFilter` into URLSearchParams           |
| History client-side filter       | `HistoryView` fetches `/api/tasks` then JS-filters                                           | Pass `projectPath` query param to API               |
| Zero MCP tool tests              | Tests in `server_test.go` + `integration_test.go` only cover protocol, not tool logic        | Create `tools_test.go` with table-driven cases      |
| Zero frontend tests              | `vitest.config.ts` exists but coverage only configured for `BacklogView`+`HistoryView`       | Create composable + view tests                      |

## Tasks

| Task     | Wave | Role      | Description                                                                                       |
| -------- | ---- | --------- | ------------------------------------------------------------------------------------------------- |
| TASK-373 | 1    | backend   | Fix CancelTask: allow NO_PROVIDER → CANCELLED transition + add tests                              |
| TASK-374 | 1    | backend   | Add provider pre-flight soft-check to PromoteTask (warn, don't block)                             |
| TASK-375 | 2    | extension | Fix TaskQueueProvider: filter by workspace folder + scope toggle                                  |
| TASK-376 | 2    | extension | Add workspace isolation tests to taskQueueProvider.test.ts                                        |
| TASK-377 | 2    | extension | Fix WorkspaceOrchView: LiveTasksGroupNode filters tasks by folderPath                             |
| TASK-378 | 3    | frontend  | Fix useActivities.ts: wire projectFilter option to API query param                                |
| TASK-379 | 3    | frontend  | Fix HistoryView + LiveActivityView: pass projectPath to API not client-filter                     |
| TASK-380 | 4    | testing   | Create mcp/tools_test.go: table-driven tests for all 31 tool handlers                             |
| TASK-381 | 4    | testing   | Create httpapi/handlers_tasks_test.go: NO_PROVIDER cancel + promote pipeline                      |
| TASK-382 | 5    | testing   | Create frontend Vitest tests for composables (useActivities, useProjectState, useDiscoveredPlans) |
| TASK-383 | 5    | testing   | Create frontend Vitest tests for views (HistoryView, LiveActivityView, ProvidersView)             |
| TASK-384 | 5    | testing   | Add VS Code extension workspace isolation tests                                                   |
| TASK-385 | 6    | testing   | Create black-box E2E Go test: full pipeline from HTTP API + MCP lifecycle                         |

## Wave Plan

```
Wave 1 (TASK-373, 374)   — Backend bug fixes. Unblock all state machine issues.
Wave 2 (TASK-375, 376, 377) — VS Code extension workspace isolation. Independent of Wave 1.
Wave 3 (TASK-378, 379)   — Frontend project filtering fixes. Independent of Waves 1+2.
Wave 4 (TASK-380, 381)   — Backend + MCP test coverage. Requires Wave 1 fixes to be in place.
Wave 5 (TASK-382, 383, 384) — Frontend + extension test suites. Requires Waves 2+3.
Wave 6 (TASK-385)        — Full E2E black-box validation. Requires all prior waves.
```

Waves 1, 2, and 3 are **fully independent** and can run as a parallel swarm.

## Success Criteria

- [x] `CancelTask` succeeds for NO_PROVIDER tasks (returns 204)
- [x] `PromoteTask` returns `{ promoted: true, warning: "no active provider" }` when no LLM registered
- [x] VS Code extension shows only current workspace tasks by default; global toggle available
- [x] `useActivities({ projectFilter })` passes `?projectPath=...` to API
- [x] `HistoryView` API call includes `?projectPath=` when project is selected
- [x] `CGO_ENABLED=1 go test -race ./internal/adapters/inbound/mcp/...` PASS (including new tools_test.go)
- [x] `npx vitest run` passes all new frontend tests
- [x] `npm test` in vscode-extension passes all new workspace isolation tests
- [x] E2E blackbox test passes against running daemon
