---
id: PLAN-075
title: 'v0.11.0 Perfection: Closing Every Audit Gap across Engine, Surfaces, UI, Extension, Docker & Docs'
status: active
createdAt: 2026-10-19
tasks:
  [
    TASK-609,
    TASK-610,
    TASK-611,
    TASK-612,
    TASK-613,
    TASK-614,
    TASK-615,
    TASK-616,
    TASK-617,
    TASK-618,
    TASK-619,
    TASK-620,
    TASK-621,
    TASK-622,
    TASK-623,
    TASK-624,
  ]
---

# PLAN-075 — v0.11.0 Full-Round Perfection

Source: [docs/audit-v0.11.0.md](../../docs/audit-v0.11.0.md). Goal: resolve every critical bug,
concurrency hazard, `PLAN-074` engine gap, Wails/HTTP parity hole, UI regression, extension gap, and
documentation discrepancy identified in the v0.11.0 full-round audit. Every task follows the project
rules: failing test first, max 500 lines per file, 100% TS/Vue coverage, $\ge 96\%$ Go merged coverage,
0 lint/type/doc issues.

## Wave 1 — Critical Build, Release, Docker & Desktop Runtime (`TASK-609`..`TASK-611`)

| Task         | Scope                                                                             | Summary                                                                                                                                                                                                                                                                                                        |
| :----------- | :-------------------------------------------------------------------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **TASK-609** | `Dockerfile`, `ci.yml`, `Makefile`, `docscheck`                                   | Build & embed Vue frontend in `Dockerfile` Stage 0; compile `nexus-daemon` with `-DSQLITE_ENABLE_FTS5` in `Dockerfile` and `ci.yml`; add `nexus-submit` and `CGO_ENABLED=0` for `nexus-cli` in `Makefile`; guard with `docscheck` tests                                                                        |
| **TASK-610** | `release-please-config.json`, `scripts/version-sync.sh`, `publish.yml`, manifests | Remove `"release-as": "0.10.0"` bootstrap pin; sync all 5 manifests (`frontend`, `vscode-extension`, `github-action`, `docs`, `wails.json`) to `0.10.0`; run `version-sync.sh` before `wails build` and `vsce` build in `publish.yml`; fix orphan tag commit in `publish.yml`                                  |
| **TASK-611** | `frontend/src/types/wails.ts`, `app.go`, `main.go`, `httpapi_client`              | Prepend `resolveServerUrl()` in `wails.ts` `request()` so `getUsage()` works in Wails desktop mode; add `IngestContent`, `GetRecentActivities`, `GetActivityTimeline`, `GetUsage`, `GetRouting` to `httpapi_client` and wire into `app.go` / `main.go` so attached-daemon desktop mode has 100% feature parity |

## Wave 2 — Core Engine Concurrency, Persistence & Watchdog (`TASK-612`..`TASK-613`)

| Task         | Scope                                                                                                    | Summary                                                                                                                                                                                                                                                                                            |
| :----------- | :------------------------------------------------------------------------------------------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **TASK-612** | `repo_sqlite/migrations.go`, `ai_session_repo.go`, `sys_scanner/agent_scanner.go`, `session_service.go`  | Persist `pid` in `ai_sessions` SQLite table and preserve `PID`/`ProcessName` in `agent_scanner.go` deduplication so `TerminateAISession` signals real processes; fix `GetDiscoveredAgents` lock and cache return when `agentRepo == nil`                                                           |
| **TASK-613** | `activity_service.go`, `orchestrator_workers.go`, `repo_sqlite/repo.go`, `task_service.go`, `waiting.go` | Fix mutex synchronization on `ActivityService.broadcaster`; skip active in-process runs (`o.runs`) in `runTaskWatchdog` and persist watchdog failure logs; route `UpdateTaskStatus` through `o.emit` so `policy.Remember` records agent-completed tasks; track `holdUntilReset` timers on `Stop()` |

## Wave 3 — `PLAN-074` Engine & Inbound Surface Parity (`TASK-614`..`TASK-618`)

| Task         | Scope                                                                                               | Summary                                                                                                                                                                                                                                                          |
| :----------- | :-------------------------------------------------------------------------------------------------- | :--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **TASK-614** | `services/ladder.go`, `services/agent_loop.go`, `services/execution_engine.go`                      | Enforce daily token budgets (`enforceBudgets`) even when `task.ProviderName` is set and between multi-turn agentic/correction calls; call `o.stepUp(run, current)` in `agent_loop.go` when verification fails                                                    |
| **TASK-615** | `services/verification.go`                                                                          | Prevent self-healing correction turns from polluting the persistent project `sessionRepo` with broken code/logs; bound `verifyOut` in `correctionPrompt` with `truncateObservation`                                                                              |
| **TASK-616** | `services/prompts.go`, `services/brain_service.go`, `llm_anthropic`                                 | Inject FTS-ranked brain entries (`GetFocusedContext`) into task prompts; make `GetProjectBrief` prefix byte-stable by removing live task queue counters; send `cache_control: {"type": "ephemeral"}` on system blocks in `llm_anthropic`                         |
| **TASK-617** | `mcp/brain_tools.go`, `mcp/tools_tasks.go`, `cli/brain.go`, `cli/root.go`, `surface_matrix_test.go` | Support structured `{kind, topic, content}` in MCP `ingest_knowledge` and CLI `nexus brain ingest`; add `--question` to `nexus brain context` (`GetFocusedContext`); add `targetFile`, `providerHint`, `status` to MCP `update_task` and CLI `nexus task update` |
| **TASK-618** | `cmd/nexus-submit/main.go`, `cmd/nexus-submit/main_test.go`                                         | Support `NEXUS_API_TOKEN` bearer authentication and `WAITING` status (`waitUntil`) in `nexus-submit`                                                                                                                                                             |

## Wave 4 — Frontend UI & Policy Editor Completeness (`TASK-619`..`TASK-622`)

| Task         | Scope                                                                                                 | Summary                                                                                                                                                                                                                                                                                    |
| :----------- | :---------------------------------------------------------------------------------------------------- | :----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **TASK-619** | `wails.ts`, `SettingsView.vue`                                                                        | Store and send `Authorization: Bearer <token>` in browser UI when API token is enabled/rotated; make `queueCap` editable in `SettingsView.vue`                                                                                                                                             |
| **TASK-620** | `useTasks.ts`, `MissionControlView.vue`, `TaskDetailDrawer.vue`, `HistoryView.vue`, `BacklogList.vue` | Include `WAITING` tasks in active counts/filters, display `waitUntil` resume badge, enable Cancel on `WAITING` tasks in `TaskDetailDrawer.vue`, show `ranBy`/`providerHint`, surface `PromoteResult.warning`, and include `TOO_LARGE`/`NO_PROVIDER` in history                             |
| **TASK-621** | `ProvidersView.vue`, `ProviderStatus.vue`, `AgentsView.vue`, `LiveActivityView.vue`, `BrainView.vue`  | Render `<ProviderLimitBadges>` and routing preview (`GET /api/routing`) on `ProvidersView.vue` and remove duplicate config table; wire `DelegateToNexus` and `TerminateAISession` in `AgentsView.vue`; fix `LiveActivityView.vue` refresh button; sync views with sidebar `currentProject` |
| **TASK-622** | `httpapi`, `cli`, `app.go`, `wails.ts`, `SettingsView.vue` / `ProvidersView.vue`                      | Expose project policy (`GET/PUT /api/policy`: `ladder`, `budgets`, `quotaThreshold`, `contextBudget`, `historyBudget`) over REST, CLI, Wails, and Frontend UI (`PLAN-074` B3/C4)                                                                                                           |

## Wave 5 — VS Code Extension, GitHub Action & Documentation (`TASK-623`..`TASK-624`)

| Task         | Scope                                                         | Summary                                                                                                                                                                                                                                                                                                |
| :----------- | :------------------------------------------------------------ | :----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **TASK-623** | `vscode-extension/src/*`, `github-action/src/*`               | Show provider cooldowns, quota, and 24h token usage in VS Code status bar & provider commands; keep `WAITING` tasks in `active` tree with `waitUntil` tooltip and cancel support; report token `usage` from `lmBridge.ts`; handle `NO_PROVIDER`, `waitUntil`, and `api_token` in `github-action`       |
| **TASK-624** | `docs/*`, `README.md`, `CHANGELOG.md`, `internal/docscheck/*` | Update `docs/mcp-integration.md` to all 44 tools; fix route/arg typos in `docs/api-reference.md`, `docs/delegating-subscriptions.md`, `docs/getting-started.md`; add `token-economy.md`, `switching-providers.md`, `subscription-bridges.md` to `docs/src/data/guides.ts`; tighten `docscheck_test.go` |
