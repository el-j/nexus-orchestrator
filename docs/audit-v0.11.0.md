# Audit of v0.11.0 (Pre-Release Full Code Review & Gap Analysis)

Measured on `feature/plan074-finish` on 2026-10-19 after completing the initial `PLAN-073` and
`PLAN-074` waves. Every source file across the Go backend (`internal/core`, `internal/adapters`,
`internal/bootstrap`, `cmd/*`, `main.go`, `app.go`), the Vue 3 frontend (`frontend/src`), the VS
Code extension (`vscode-extension/src`), the GitHub Action (`github-action/src`), the documentation
site (`docs/`), and the build/release pipelines (`Makefile`, `Dockerfile`, `.github/workflows/*`,
`scripts/*`) was audited line by line.

---

## 1. Executive Summary & Readiness Matrix

| Area                                                      | Current State                                                                                                                                                                                                                                                                                                                                                                                  | Verdict                                                   |
| :-------------------------------------------------------- | :--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :-------------------------------------------------------- |
| **Test & Type Gates**                                     | Go merged coverage **97.6 %** (gate 96 %); Frontend, Docs, Extension, GitHub Action **100 %** (all 4 metrics); `go vet`, `vue-tsc`, `tsc`, `check-tsdoc`, `check-file-size.sh` all **0 errors**                                                                                                                                                                                                | **Green**, with specific functional and parity gaps below |
| **Docker Image (`Dockerfile`)**                           | Compiles `nexus-daemon` **without** `-DSQLITE_ENABLE_FTS5` (crashes at startup on `project_knowledge_fts`) and **without** building/embedding the Vue frontend (`/app/`)                                                                                                                                                                                                                       | **Critical blocker** (`TASK-609`)                         |
| **Release & Version Sync**                                | `release-please-config.json` still pins `"release-as": "0.10.0"`; `version-sync.sh` misses `github-action` and `docs`; `publish.yml` `build-desktop` stamps `1.0.0` and `release` tags an unpushed commit                                                                                                                                                                                      | **High priority** (`TASK-610`)                            |
| **Wails Desktop & Attached Mode**                         | `wails.ts` `request()` uses relative paths so `getUsage()` fails in Wails desktop mode; attached-daemon mode (`main.go`) lacks `Activity`, `IngestContent`, and workspace watch                                                                                                                                                                                                                | **High priority** (`TASK-611`)                            |
| **Core Engine & SQLite Persistence**                      | `ai_sessions` drops `PID` so `TerminateAISession` never signals processes; data race on `ActivityService.broadcaster`; `runTaskWatchdog` times out active in-process runs and drops logs; `UpdateTaskStatus` skips `recordOutcome`                                                                                                                                                             | **High priority** (`TASK-612`, `TASK-613`)                |
| **`PLAN-074` Engine Completeness**                        | Explicit `task.ProviderName` bypasses daily token budgets; `agent_loop.go` never calls `stepUp` on failed verification; correction turns pollute project `sessionRepo`; `prepareChatPrompt` never calls `GetFocusedContext` (FTS); `GetProjectBrief` embeds live counters breaking prompt-cache prefix stability                                                                               | **High priority** (`TASK-614`, `TASK-615`, `TASK-616`)    |
| **Inbound Surface Parity (`mcp`, `cli`, `nexus-submit`)** | `mcp`/`cli` `ingest_knowledge` only ingests files (not structured `POST /api/brain/knowledge`); `cli` lacks `focused-context`; `mcp`/`cli` `update_task` omit `targetFile`, `providerHint`, `status`; `nexus-submit` ignores `NEXUS_API_TOKEN` and `WAITING`                                                                                                                                   | **Medium priority** (`TASK-617`, `TASK-618`)              |
| **Frontend UI Completeness**                              | Browser token lockout in `SettingsView.vue`; `WAITING` tasks & `waitUntil` hidden from active filters/counts/drawer cancel; duplicate config tables & missing `<ProviderLimitBadges>` / routing preview in `ProvidersView.vue`; dead `AISessionCard.vue` (missing Delegate/Terminate in `AgentsView.vue`); broken Scan button in `LiveActivityView.vue`; `currentProject` ignored across views | **High priority** (`TASK-619`, `TASK-620`, `TASK-621`)    |
| **Policy Management (`PLAN-074` B3/C4)**                  | `.nexus/policy.json` (`ladder`, `budgets`, `quotaThreshold`, `contextBudget`, `historyBudget`) has no REST/Wails/CLI/UI editor                                                                                                                                                                                                                                                                 | **Medium priority** (`TASK-622`)                          |
| **VS Code Extension & GitHub Action**                     | Extension status bar and provider picker ignore `rateLimited`, `quota`, and 24h token usage; `WAITING` tasks misplaced in `recent` and missing `waitUntil`; `lmBridge.ts` omits token `usage`; GitHub Action mishandles `NO_PROVIDER`                                                                                                                                                          | **Medium priority** (`TASK-623`)                          |
| **Documentation & `docscheck`**                           | `docs/mcp-integration.md` lists 40 tools instead of 44; route typos in `docs/api-reference.md` and `docs/delegating-subscriptions.md`; 3 new `PLAN-074` guides missing from `docs/src/data/guides.ts`; stale claims in `README.md`                                                                                                                                                             | **Medium priority** (`TASK-624`)                          |

---

## 2. Detailed Findings by Layer

### 2.1 Docker, CI/CD, Makefile & Version Sync (`TASK-609`, `TASK-610`)

1. **`Dockerfile` Crashes at Startup & Ships Without Full UI (`Dockerfile:25-58`)**:
   - `internal/adapters/outbound/repo_sqlite/knowledge_repo.go:31-52` unconditionally creates `project_knowledge_fts USING fts5(...)` on startup. `Dockerfile:40` runs `CGO_ENABLED=1 go build` without `CGO_CFLAGS="-DSQLITE_ENABLE_FTS5"`, causing the container to exit immediately with `no such module: fts5`.
   - `Dockerfile` has no Node 22 build stage to populate `internal/adapters/inbound/webui/dist` before building `nexus-daemon`, so container builds lack the full Vue UI at `/app/`.
   - `.github/workflows/ci.yml:155` (`build` job) also compiles `nexus-daemon` without `CGO_CFLAGS="-DSQLITE_ENABLE_FTS5"`.
2. **`release-please-config.json` Blocks `v0.11.0` (`release-please-config.json:3,8`)**:
   - Still contains `"release-as": "0.10.0"` and `"bootstrap-sha": "41ac63fe85508e25d5a0a26a021dc4203acbf3f3"`. Per `docs/releasing.md:41-42`, these must be removed so `release-please` computes `0.11.0`.
3. **Manifest Version Drift & `scripts/version-sync.sh` (`scripts/version-sync.sh:19-50`)**:
   - `.release-please-manifest.json` is `"0.10.0"`, `frontend/package.json`, `github-action/package.json`, `docs/package.json`, and `wails.json` are `"1.0.0"`, and `vscode-extension/package.json:5` is `"0.2.0-62"`.
   - `scripts/version-sync.sh` updates `frontend/package.json`, `vscode-extension/package.json`, and `wails.json`, but misses `github-action/package.json` and `docs/package.json`.
4. **`.github/workflows/publish.yml` Build & Tagging Discrepancies**:
   - `build-desktop` (`publish.yml:443-480`) never runs `scripts/version-sync.sh` (so desktop binaries report version `1.0.0` from `wails.json`) and does not copy `build/frontend/.` into `internal/adapters/inbound/webui/dist/`.
   - `build-vscode` (`publish.yml:597-605`) runs `npm run build` _before_ `scripts/version-sync.sh`.
   - `release` (`publish.yml:690-711`) creates a local unpushed commit before `git tag -a "${TAG}"`, causing beta tags to point to an orphan commit SHA that does not match the `COMMIT` SHA embedded in the released binaries.
   - `Makefile` (`Makefile:82-89, 295-370`) omits `nexus-submit` from `make build` and `make build-*` targets and builds `nexus-cli` with `CGO_ENABLED=1` instead of `CGO_ENABLED=0`.

---

### 2.2 Desktop App (`main.go`, `app.go`), `wails.ts` & Attached-Daemon Parity (`TASK-611`)

1. **`getUsage()` Fails in Wails Desktop Mode (`frontend/src/types/wails.ts:88-107, 143-145`)**:
   - `request<T>(method, path, body)` calls `fetch(path, ...)` with a relative URL instead of prepending `await resolveServerUrl()`.
   - Because `getUsage()` (`wails.ts:143-145`) uses `request('GET', '/api/usage?since=...')` and has no Wails binding on `App`, calling `getUsage()` inside the Wails WebView (`wails://wails` or `http://wails.localhost`) hits the static asset server instead of `http://127.0.0.1:63987`, breaking `UsageCard.vue` in the desktop app.
2. **Attached-Daemon Mode (`main.go:54-73`, `app.go`) Drops Activity Feed and Content Ingestion**:
   - When `attachOrStart` attaches to an already-running `nexus-daemon`, `App.activitySvc` is `nil` (`app.go:20` is typed to concrete `*services.ActivityService`), so `GetRecentActivities` and `GetActivityTimeline` fail with `"app: activity service not available"`.
   - `httpapi_client.Client` lacks `GetRecentActivities` (`GET /api/activities`) and `GetActivityTimeline` (`GET /api/activities/timeline`), as well as `GetUsage` (`GET /api/usage`) and `GetRouting` (`GET /api/routing`).
   - `httpapi_client.BrainClient` (`internal/adapters/outbound/httpapi_client/brain_client.go`) lacks `IngestContent` (`POST /api/brain/ingest-content`), so `App.IngestContent` (`app.go:272-283`) fails with `"brain service cannot ingest uploaded content"` when attached to an external daemon.
3. **Missing Wails Bindings on `App` (`app.go`)**:
   - `App` is missing bindings for `GetUsage`, `GetRouting`, `PromoteProvider`, `TerminateAISession`, `GetDiscoveredAgents`, `DelegateToNexus`, and `GetDiscoveredPlanFiles`.

---

### 2.3 Core Engine, Concurrency & SQLite Persistence (`TASK-612`, `TASK-613`)

1. **`TerminateAISession` Never Signals Processes Because `PID` Is Not Persisted (`migrations.go:40-51`, `ai_session_repo.go:39-51`, `session_service.go:123-157`)**:
   - `ai_sessions` has no `pid` column in SQLite. `SaveAISession` and `scanAISession` drop `domain.AISession.PID`.
   - When `TerminateAISession` loads the session from SQLite, `sess.PID` is always `0`, so `if sess.PID > 0` is never entered and the OS process is never signalled.
   - In `internal/adapters/outbound/sys_scanner/agent_scanner.go:44-57`, merging duplicate agent IDs across probes also forgets to copy `a.PID` and `a.ProcessName`.
2. **Data Race on `ActivityService.broadcaster` (`internal/bootstrap/runtime.go:179,204`, `internal/core/services/activity_service.go:60,152`)**:
   - `rt.Activity.Start()` starts `pollLoop()` (which reads `s.broadcaster` without a lock at line 152) before `httpapi.StartServerWith` calls `activitySvc.SetBroadcaster(hub)` (which writes `s.broadcaster` without a lock at line 60).
3. **`runTaskWatchdog` Races With Active In-Process Runs and Drops Logs (`internal/core/services/orchestrator_workers.go:212-236`)**:
   - `runTaskWatchdog` marks any `PROCESSING` task older than 5 minutes as `FAILED`, even if `processNext` is actively running it in `o.runs` (for example a 10-minute `llm_cli` call or multi-turn agent loop).
   - It then calls `o.repo.Update(o.dbCtx(), t)` (`repo.go:248-256`), whose `UPDATE tasks` SQL statement does not include the `logs` column—silently discarding the watchdog's `[System] Task timed out...` log message.
4. **`UpdateTaskStatus` Skips `recordOutcome` (`internal/core/services/task_service.go:399-406`)**:
   - When an external MCP agent completes or fails a claimed task via `UpdateTaskStatus`, it calls `o.dispatch` instead of `o.emit`, bypassing `o.recordOutcome` (`brain_ingest.go:108`) so externally completed tasks are never remembered in the project brain.
5. **`GetDiscoveredAgents` Cache Bug Without `agentRepo` (`internal/core/services/session_service.go:239-268`)**:
   - Reads `o.agentScanner` and `o.agentRepo` outside `o.mu.Lock()`, and when called within the 30-second cache window with `o.agentRepo == nil`, returns `nil` instead of the last scanned slice.
6. **Untracked Timer in `holdUntilReset` (`internal/core/services/waiting.go:46-52`)**:
   - `time.AfterFunc` is not cancelled on `OrchestratorService.Stop()`, risking a call to `ResumeWaitingTasks` against a closed database during shutdown.

---

### 2.4 `PLAN-074` Routing, Budget & Token-Economy Engine Gaps (`TASK-614`, `TASK-615`, `TASK-616`)

1. **Explicit `task.ProviderName` Bypasses Daily Token Budgets (`internal/core/services/ladder.go:25-33`)**:
   - `routeChain` returns early when `task.ProviderName != ""` _before_ calling `o.enforceBudgets(chain, policy)`. Calling `o.enforceBudgets(chain, policy)` before the early return ensures daily budgets apply even when a task names its provider.
2. **Multi-Turn Runs Do Not Re-Check Budgets (`internal/core/services/agent_loop.go`, `verification.go`)**:
   - `enforceBudgets` is only run once before turn 1; a 25-turn agentic task or multi-turn self-healing loop continues calling a provider even after crossing its daily token budget mid-task.
3. **`agent_loop.go` Never Calls `stepUp` After Failed Verification (`internal/core/services/agent_loop.go:183-195`)**:
   - Non-agentic verification (`verification.go:84`) steps up the model ladder (`current = o.stepUp(run, current)`) when verification fails, but `runAgentLoop` never calls `o.stepUp` when the `"verify"` or `"finish"` tool fails verification.
4. **Self-Healing Correction Turns Pollute Project `sessionRepo` & Send Unbounded Output (`internal/core/services/verification.go:84-95, 140-147`)**:
   - `verification.go:92` calls `executeGeneration(..., useChat = true)`, which appends every failed correction prompt (containing the full broken file and full test output) into the persistent project `sessionRepo`.
   - `correctionPrompt` sends the entire unfiltered `verifyOut` instead of extracting/bounding failing output (`truncateObservation`).
5. **FTS-Ranked Brain Entries (`GetFocusedContext`) Never Used by Task Prompt Builder (`internal/core/services/prompts.go:13-57`)**:
   - `PLAN-074` D1 specified using FTS-ranked brain entries relevant to the task, but `prepareChatPrompt` and `applyProjectRules` only call `GetProjectBrief` (`GetContext`), never `brain.GetFocusedContext` with `task.Instruction`.
6. **Unstable Prompt Prefix Prevents Prompt Caching (`internal/core/services/brain_service.go:338-383`, `internal/adapters/outbound/llm_anthropic/adapter.go:162-174`)**:
   - `GetProjectBrief` embeds live task counters (`- **Tasks:** %d queued, %d processing...`) and `- **Current Task:** %s (%s)` into the briefing, mutating the prefix on every task.
   - `llm_anthropic` serializes `system` as a plain string without `cache_control: {"type": "ephemeral"}`.

---

### 2.5 Inbound Adapter Parity: MCP, CLI & `nexus-submit` (`TASK-617`, `TASK-618`)

1. **Structured `IngestKnowledge` Unreachable from MCP and CLI (`internal/adapters/inbound/mcp/brain_tools.go:119-145`, `internal/adapters/inbound/cli/brain.go:53-77`)**:
   - Both MCP `ingest_knowledge` and CLI `nexus brain ingest` only accept `filePath` (`IngestFromFile`). Neither accepts `{projectPath, kind, topic, content}` to call `BrainService.IngestKnowledge` (`POST /api/brain/knowledge`), even though `baselineRules` tells agents to report learnings via `ingest_knowledge`.
2. **CLI `nexus brain context` Cannot Query `GetFocusedContext` (`internal/adapters/inbound/cli/brain.go:79-103`)**:
   - `surface_matrix_test.go:85` claims CLI covers `"brain: focused context"` via `has("context")`, but `newBrainContextCmd` has no `--question` / `-q` flag and never calls `brain.GetFocusedContext`.
3. **Incomplete `UpdateTask` Fields on MCP and CLI (`internal/adapters/inbound/mcp/tools_tasks.go:228-247`, `internal/adapters/inbound/cli/root.go:208-252`)**:
   - Both omit `targetFile`, `providerHint`, and `status` (`DRAFT` ↔ `BACKLOG`), which REST `PUT /api/tasks/{id}` and `OrchestratorService.UpdateTask` support.
4. **`cmd/nexus-submit/main.go` Missing Auth Token & `WAITING` Handling (`cmd/nexus-submit/main.go:184-261`)**:
   - Does not read `NEXUS_API_TOKEN` or send `Authorization: Bearer <token>`.
   - `waitForCompletion` does not report `WAITING` / `waitUntil` while polling.

---

### 2.6 Frontend UI Completeness (`TASK-619`, `TASK-620`, `TASK-621`, `TASK-622`)

1. **Browser API Token Lockout (`frontend/src/views/SettingsView.vue:55-101`, `frontend/src/types/wails.ts:88-107`)**:
   - Generating an HTTP API token in `SettingsView.vue` activates `tokenAuthMiddleware` on the daemon, but `wails.ts` never stores or sends `Authorization: Bearer <token>`, immediately locking out the browser UI with `401 Unauthorized`.
   - `SettingsView.vue:30-49` also renders `queueCap` as static read-only text despite `updateRuntimeConfig({ queueCap })` supporting live updates.
2. **`WAITING` Status & `waitUntil` Missing Across UI (`MissionControlView.vue`, `TaskDetailDrawer.vue`, `useTasks.ts`, `HistoryView.vue`)**:
   - `useTasks.ts:30-36` and `MissionControlView.vue:16-24, 378-379, 414-417` exclude `WAITING` from active counts and the `"Active"` filter pill.
   - `task.waitUntil` is never rendered in `MissionControlView.vue` or `TaskDetailDrawer.vue`.
   - `TaskDetailDrawer.vue:104` only allows cancelling `QUEUED` or `PROCESSING` tasks, hiding the Cancel button for `WAITING` tasks.
   - `MissionControlView.vue:90` shows `task.providerName || '—'` instead of `task.ranBy || task.providerName || task.providerHint || '—'`.
   - `MissionControlView.vue:339-360` and `HistoryView.vue:209-236` omit `TOO_LARGE` and `NO_PROVIDER` from history filters/colors.
3. **`ProvidersView.vue` Duplicate CRUD Tables & Missing Limit Badges / Routing Preview (`frontend/src/views/ProvidersView.vue:85-232`)**:
   - `ProvidersView.vue` mounts `<ProviderStatus>` (which includes its own `"Configured Providers"` CRUD table) and then renders a second `"Configuration"` CRUD table below it.
   - `<ProviderLimitBadges>` (cooldowns and plan quota utilization) is rendered in `MissionControlView.vue` but missing from `ProvidersView.vue` and `ProviderStatus.vue`.
   - Neither `ProvidersView.vue` nor `MissionControlView.vue` displays the routing preview (`GET /api/routing`), despite `CHANGELOG.md:38` claiming it does.
4. **`AgentsView.vue`, `LiveActivityView.vue`, `BacklogList.vue`, and `BrainView.vue` Regressions**:
   - `AgentsView.vue:147-216` renders session cards inline instead of using `AISessionCard.vue`, losing the `"Delegate →"` action (`POST /api/ai-sessions/{id}/delegate`), session termination (`POST /api/ai-sessions/{id}/terminate`), capabilities badges, and routed task history.
   - `LiveActivityView.vue:17-23, 130` binds `"⟳ Scan Now"` to `useDiscovery().scanNow()` (provider scan) instead of `useActivities().refresh()`.
   - `useTasks.ts:53-56` and `BacklogList.vue:126-137` discard the `PromoteResult` (`{ promoted, warning }`) from `promoteTask(id)` and do not display `providerHint`.
   - `BrainView.vue:331`, `MissionControlView.vue`, `AgentsView.vue`, and `LiveActivityView.vue` do not watch `currentProject` from `useProjectState.ts` when the user switches projects in `ProjectSelector.vue`.
5. **No UI, REST, or CLI Policy Editor (`PLAN-074` B3/C4, `TASK-622`)**:
   - `.nexus/policy.json` (`ladder`, `budgets`, `quotaThreshold`, `contextBudget`, `historyBudget`) cannot be viewed or edited through REST (`GET/PUT /api/policy`), CLI (`nexus policy` / `nexus routing`), Wails, or the Frontend UI.

---

### 2.7 VS Code Extension & GitHub Action (`TASK-623`)

1. **VS Code Extension Cooldowns, Quota & Usage (`vscode-extension/src/statusBar.ts`, `commands/index.ts`, `nexusClient.ts`)**:
   - `NexusStatusBar` and `showProvidersCommand` never display `rateLimited`, `rateLimitedUntil`, `cooldownReason`, `quota`, or 24h token usage (`/api/usage`).
2. **VS Code Extension `WAITING` Tasks (`vscode-extension/src/nexusTypes.ts`, `statusBar.ts:92`, `taskQueueProvider.ts:93-105, 183-188`, `workspaceOrchView.ts:315-322`, `extension.ts:229`)**:
   - `Task` in `nexusTypes.ts` lacks `waitUntil` and `retryCount`.
   - `WAITING` tasks are excluded from active counts in `statusBar.ts`, placed under `recent` (finished) instead of `active` in `taskQueueProvider.ts` and `workspaceOrchView.ts`, omitted from `nexus.cancelTask` in `extension.ts`, and `waitUntil` is not shown in `TaskItem` tooltips.
3. **VS Code LM Bridge Usage Reporting (`vscode-extension/src/lmBridge.ts:146-174`)**:
   - Does not include estimated `usage` (`prompt_tokens`, `completion_tokens`, `total_tokens`) in `/v1/chat/completions` responses, so `vscode.lm` calls record 0 tokens in the daemon's usage ledger.
4. **GitHub Action Terminal Status & Auth (`github-action/src/index.ts:189-206`, `submit.ts:24-46`, `types.ts`)**:
   - `NO_PROVIDER` is missing from the terminal `switch (finalTask.status)` in `index.ts:189-206`.
   - `waitUntil` is not logged when a task is `WAITING`, and `api_token` (`Authorization: Bearer`) is not supported for remote daemons.

---

### 2.8 Documentation & `internal/docscheck` (`TASK-624`)

1. **`docs/mcp-integration.md` Lists 40 Tools Instead of 44 (`lines 72-136, 189`)**:
   - Missing `wait_for_task`, `get_onboarding_context`, `get_routing`, and `get_usage`.
   - Uses `"taskId"` instead of `"id"` in the `get_task` example (`docs/mcp-integration.md:189` and `docs/api-reference.md:924`).
2. **Route & Event Table Fixes (`docs/api-reference.md:347-359, 723`, `docs/delegating-subscriptions.md:13`, `docs/getting-started.md:112-120, 187-188`)**:
   - `docs/api-reference.md:723` writes `POST /api/tasks/{id}/promote` instead of `POST /api/tasks/promote/{id}`, and `347-359` omits `task.waiting`, `task.chunk`, and `task.restart`.
   - `docs/delegating-subscriptions.md:13` writes `POST /api/providers/discovered/{id}/promote` instead of `POST /api/providers/promote/{id}`.
   - `docs/getting-started.md:112-120` shows the wrong response shape for `POST /api/tasks` and `187-188` links to `nexus-orchestrator-vscode.vsix` instead of `nexus-orchestrator.vsix`.
   - `internal/docscheck/docscheck_test.go` should check `docs/api-reference.md` directly (without concatenating `CHANGELOG.md`) and verify that `docs/mcp-integration.md` names all 44 MCP tools.
3. **Missing Guides on Docs Site & Stale Statements (`docs/src/data/guides.ts`, `README.md`, `docs/using-nexus.md`, `docs/vscode-model-bridge.md`, `docs/subscription-bridges.md`, `CHANGELOG.md`)**:
   - Add `token-economy.md`, `switching-providers.md`, and `subscription-bridges.md` to `docs/src/data/guides.ts` and `README.md`.
   - Update `README.md`, `docs/using-nexus.md`, `DownloadsView.vue`, `docs/vscode-model-bridge.md`, and `docs/subscription-bridges.md` for attached-daemon desktop mode (`TASK-579`), `copilot` CLI (`TASK-600`), and `vscode.lm` `maxInputTokens`, and merge the duplicate `### Added` section in `CHANGELOG.md`.

---

## 3. Execution Plan

All findings above are translated into **`.claude/plans/PLAN-075.md`** (`TASK-609` through `TASK-624`),
ordered into 5 waves from critical build/runtime fixes to full UI, extension, and documentation parity.
