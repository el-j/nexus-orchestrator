# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- The daemon serves the full Vue UI at `/app/` (the root redirects there) when it is built with the frontend: `make start` / `start-bg` build it automatically with Node >= 22 (`make web` by hand) and fall back to the light dashboard at `/ui` otherwise. The UI talks to the daemon it was loaded from, whatever host and port that is
- Configurable, self-linking plan and task files: `planFiles` in `.nexus/policy.json` (`plans`, `tasks` folders, `discover`) plus the settings `nexus.planFiles.*` and the command "Nexus: Choose Plan and Task Folders"; without configuration any `PLAN-n.md` / `TASK-n.md` (or `type: plan|task`, or a `plans`/`tasks` folder) anywhere in the project is found, and tasks are linked to plans by planId, the plan's task list, their folder, or mentions; tasks no plan claims are shown as "Unassigned tasks". The daemon's onboarding and the extension share one rule set tested against `testdata/planfiles/cases.json`. See `docs/plan-files.md`
- `scripts/install.sh --desktop` installs the macOS app into `~/Applications` after a checksum check and without the quarantine flag (the app is not notarized, so a browser-downloaded copy is blocked by Gatekeeper); `--beta` / `NEXUS_CHANNEL=beta` picks the newest pre-release instead of the last stable one (which is still from April)
- Claude Pro/Max can be delegated to: provider kind `cli` runs a logged-in AI command-line tool (`claude -p`, verified against the real `claude`) per request, streams its output, and a discovered "Claude CLI" can be promoted. Only known tools may run (`NEXUS_CLI_PROVIDERS` extends the list). See `docs/delegating-subscriptions.md`
- OpenAI-compatible providers use the context length the server reports (`context_length`, `context_window`, `max_model_len`) instead of always assuming 128K; the VS Code bridge reports each editor model's `maxInputTokens`
- MCP `wait_for_task` (42 tools): blocks until a task has finished and streams the model's reply as `notifications/progress` when the caller sends a progress token and `Accept: text/event-stream`; `OrchestratorService` offers its events to in-process adapters through the optional `ports.TaskEventSource`
- Anthropic, Gemini and LM Studio providers stream (`ChatStream`, shared `sse` reader), and the single-prompt path without a session store streams too; a cut-off stream is a failure that falls over to the next provider
- Live output in the clients: VS Code channel `Nexus Live Output` (`nexus.liveOutput.enabled`, command `Nexus: Show Live Task Output`) and a "Live output" block in the task detail drawer; an OpenAI-compatible server that ignores `stream: true` is still accepted (one chunk)
- OpenAI-compatible and Ollama providers stream (`ChatStream`): server-sent events / newline-delimited JSON, a cut-off stream is a failure that falls over to the next provider
- Live output: the engine publishes a task's reply on the event stream while the model writes it (`task.chunk` with `text` and `seq`, `task.restart` before every attempt: retry, next provider, each agent turn). Optional `ports.ChatStreamer` for providers; others send one chunk. Not stored. See `docs/streaming.md`
- VS Code model bridge streams: `POST /v1/chat/completions` with `"stream": true` answers with OpenAI `chat.completion.chunk` server-sent events as the editor's model produces text (a failure mid-stream arrives as an error event, a disconnect cancels the model request)
- VS Code model bridge: with `nexus.lmBridge.enabled` the extension serves an OpenAI-compatible endpoint on loopback (bearer token) in front of `vscode.lm` and registers each model the editor offers (Copilot and others) with the daemon as a `vscode-<vendor>-<model>` provider, kept in step with daemon restarts and model changes. Off by default. See `docs/vscode-model-bridge.md`
- Rule-based workflow: `.nexus/policy.json` per project (default `verify` command, `maxFileLines`, `testsFirst`, `remember`, free-text `rules`). The engine puts a token-bounded project briefing and the rules in front of every task, applies the default verification command, refuses agent writes that break the size or tests-first rules, remembers each finished task in the project knowledge (newest 30) and sends only the newest 20 session messages; the onboarding leads with the rules; `init` creates the file once. See `docs/rulebased-workflow.md`
- Agentic `edit_file` tool: replaces the single exact occurrence of a text in a project file (refused when it is missing or ambiguous), so small changes no longer need the whole file rewritten
- Cross-surface parity matrix (`internal/docscheck/surface_matrix_test.go`): every capability must exist on REST, MCP, CLI, the desktop bindings, the frontend and the VS Code extension, or say why not; CI fails otherwise
- `docs/api-reference.md` documents every REST route, guarded by a test; architecture guide covers the execution engine, provider routing and the surface matrix
- `npm run check:docs` now also covers the docs site and requires a description comment on every Vue component
- Agentic tasks: `agentic` / `maxAgentTurns` on a task let the model read and write project files over several turns and re-verify before finishing; exposed on REST, MCP, `nexus-cli`, `nexus-submit` and the UI
- Routing parity: `providerName`, `providerHint`, `modelId`, `role`, `priority` and `tags` are accepted by `submit_task` / `create_draft` and the CLIs; GitHub Copilot and LM Studio are classified correctly for role routing
- Real-daemon end-to-end suite (`go test -tags e2e ./internal/e2e/`): builds the shipped binaries and drives REST, every MCP tool, the stdio proxy, both CLIs, restart persistence, shutdown and port conflicts
- Playwright `real-daemon` project (daemon plus a fake Ollama), VS Code extension-host tests (`npm run test:host`), a Wails desktop smoke job and a GitHub Action end-to-end job
- Local-only mutation testing for Go (gremlins) and TypeScript (Stryker) via `make mutation` / `scripts/mutation.sh`, documented in `docs/mutation-testing.md`; deliberately not run in CI to save workflow minutes; `scripts/check-tsdoc.mjs` and revive `exported` doc-comment gates, and `scripts/check-file-size.sh` (500-line cap)
- `make dogfood` / `start-bg` / `stop`, root `CLAUDE.md` and `.mcp.json` so sessions in this repository start on the orchestrator, `.nvmrc`, and `docs/agent-workflow-audit.md` (evidence-based review of the agent workflow and its gaps)
- `make` now prints the target overview (default goal is `help`); new `make start`, `status`, `connect` and `init-project` targets and `docs/using-nexus.md` for running one orchestrator across all projects

### Changed

- The sidebar of the UI shows five things instead of nine: Mission Control, Agents, Providers, Projects and Settings. Activity, Plans, History, Backlog and Brain are tabs of one Projects hub (`/projects`, `/projects/plans`, ...); the old paths (`/plans`, `/history`, `/backlog`, `/brain`) redirect to their tab
- Removed PrimeVue, `@primevue/themes` and `primeicons` from the frontend and the docs site (epic #64). Both now use internal primitives (`UiIcon`, `UiButton`, `UiSkeleton`, `UiSelect`, `UiDrawer`, `UiToaster`/`useToast`, `UiTabs`) with design tokens, a focus trap, reduced-motion support and an SVG icon set; see `docs/ui-foundation.md`. Bundles shrank by 42-50 % (frontend JS 190 -> 95 KiB gz, docs JS 167 -> 97 KiB gz) and CI now enforces bundle budgets (`scripts/check-bundle-budget.mjs`) and bans a UI kit (`scripts/check-no-ui-kit.mjs`).
- Astro feasibility study (`docs/astro-feasibility.md`): no-go for the frontend, no-go with checkpoints for the docs.
- Playwright regression matrix for keyboard, focus, ARIA and motion (`frontend/e2e/ui-foundation.spec.ts`).

- Repository ports (`TaskRepository`, `SessionRepository`, `ModelCapabilityRepository`) take a `context.Context`; `repo_sqlite` uses the `*Context` database calls and the `noctx` exclusion is gone
- Oversized source files split along their seams (engine, orchestrator, SQLite repository, scanners, brain service, extension client, several views)
- The GitHub Action's `command` input defaults to `auto` instead of `execute`
- Coverage ratchets: frontend, docs site, GitHub Action and extension are at 100% on every metric and gated there; Go merged coverage is 97% (96.9% in CI) with the gate at 96%

### Fixed

- The UI and the editor showed state that was not true: one repository appeared as several projects and "agents" (a Claude session started in `frontend/`, `src/` or a temp directory became its own project and discovered agent), every non-Copilot language model of the editor (`copilotcli/gpt-5.4`, ...) was listed as an active agent in every window, and tasks an agent had marked processing or done by hand looked like work Nexus was doing. Activities now belong to the repository (nearest `.git`), activity in temp directories is ignored, discovered sessions at sub-folders are removed, the extension no longer reports language models as agents, and a task says who ran it: `ranBy` (read from the engine's log) in the API, "Run by Nexus on X" or "Status set by an agent — Nexus did not run this task" in the task drawer and the editor tooltip
- Task rows another tool wrote straight into the database (an agent running `sqlite3`: `IN_PROGRESS`, times as text or in seconds) are repaired at daemon start, so the watchdog and the queue queries see them too
- One hand-edited task row made `GET /api/tasks/all` answer 500 ("converting driver.Value type string to a int64"), so every client, the VS Code extension included, showed the daemon as offline: an agent had run `UPDATE tasks SET status='IN_PROGRESS', updated_at=datetime('now')` on the database. The repository now reads timestamps stored as text (SQLite datetime, ISO 8601, a number in text), reads `IN_PROGRESS`/`DONE`/`ERROR`/... as PROCESSING/COMPLETED/FAILED, and a list skips a row it cannot decode (logged) instead of failing; asking for that task by id still reports it
- The extension's task view said "daemon offline" for any failure; a daemon that answers `/api/health` but fails the list now shows "Nexus daemon error" with the reason
- `init_project` / `nexus init` only learned from `CLAUDE.md`; it now also reads `AGENTS.md`, `GEMINI.md` and `.github/copilot-instructions.md` when they exist, so a project worked on by another agent is onboarded too
- The VS Code "Workspace Agents" view said "No orchestrator.json found" for a repository whose agent (here Gemini) wrote plans and tasks as markdown under `.claude/plans` and `.claude/tasks` but no `orchestrator.json`: the view now reads those files (and `.nexus/`), merges them with the JSON when there is one, and picks the active plan; the daemon's onboarding context does the same instead of reporting "Active Plan: None"
- Registering the same provider again (the VS Code bridge after a restart) replaced nothing and left a stale duplicate; the same name and model now replace each other
- The daemon kept running (and printed its ready banner) when the REST or MCP port was already in use; it now exits non-zero
- The Action failed on the first run in any project because `execute` requires a completed plan
- Views and composables that unmounted before their first fetch finished left a poller running for the rest of the session
- `PromoteTask` was declared as returning nothing in the desktop binding while Go returns the promote result
- The "Deterministic Daemon E2E" and Playwright CI jobs could not fail (a skipped script, `continue-on-error`); both now run for real
- `make stop` was a no-op; the zig cross-builds (`build-linux-*`, `build-windows-amd64`) failed to link; `make build-gui` reported success when Wails failed; `make nice` and the frontend builds failed on a fresh checkout or Node 20 with unclear errors
- Binaries built with `make build` (and every other Makefile build) lacked SQLite FTS5, so `nexus-daemon` exited at start-up with `no such module: fts5`
- `make nice` type-checked nothing in the frontend (references-only tsconfig) and ran `gofmt -w .` over `node_modules`; it now runs the real frontend and docs type-checks, the TSDoc gate and prettier

## [0.10.0] — 2026-03-14

### Added

- `DiscoveredAgent` domain type and `AgentKind` string type with 10 constants (`claude-cli`, `claude-desktop`, `antigravity`, `cline`, `continue`, `codegpt`, `cursor`, `copilot`, `aichat`, `generic`)
- `AgentScanner` outbound port interface; `Orchestrator` interface gains `GetDiscoveredAgents` and `DelegateToNexus` methods
- `sys_scanner` adapter: `ScanAgents()` with 5 strategies — fs-config, vscode-extensions dir, MCP port sweep (8 ports), process-flag via `pgrep`, and process pattern matching
- SQLite `discovered_agents` table (persisted agent registry) + 4 additive columns on `ai_sessions`: `delegated_to_nexus`, `delegation_timestamp`, `agent_capabilities`, `detection_method`
- HTTP API: `GET /api/ai-sessions/discovered` and `POST /api/ai-sessions/{id}/delegate` endpoints
- VS Code extension: `AgentDetector` class — 4 detection strategies, 30 s polling, session lifecycle register/heartbeat/deregister
- VS Code extension: `AISessionsTreeProvider` — `nexus.aiSessions` tree view with colour-coded status icons
- VS Code extension: `nexus.delegateToNexus` command with 3 delivery paths (cli: writes `.nexus-delegate.md`; mcp: `submitTask`; copilot: `chat.open`)
- VS Code extension: `nexus.delegateAllSessions` command
- `vscode-extension/package.json`: new `nexus.enableMCPPortSweep` configuration property
- Frontend: `AIAgentsView.vue` — AI agents dashboard with 10 s polling and "Delegate All Active" button
- Frontend: `AISessionCard.vue` — per-session card with collapsible live task timeline via EventSource
- Root-level pre-commit tooling: husky v9, lint-staged v15, prettier v3 (`gofmt -w` on `.go`; `prettier --write` on TS/Vue/JSON/MD/CSS)

### Changed

- `OrchestratorService` gains `SetAgentScanner()` and `SetDiscoveredAgentRepo()` option setters; `GetDiscoveredAgents()` uses a 30 s result-cache TTL
- Both entry points (`main.go`, `cmd/nexus-daemon/main.go`) wire `AgentScanner` and `DiscoveredAgentRepo`
- `frontend/src/utils/time.ts` (new): unified `relativeTime`/`timeAgo`/`formatDate` helpers — duplicate inline functions removed from 6 Vue components
- `frontend/src/types/domain.ts` `Task` interface: added `retryCount?: number`

### Fixed

- Test mocks for `ports.Orchestrator` in `internal/adapters/inbound/cli/root_test.go` and `internal/adapters/inbound/mcp/server_test.go` now implement the full interface (added `GetDiscoveredAgents` and `DelegateToNexus` stubs)

## [0.9.4] — 2026-03-12

### Added

- VS Code extension: "Workspace Agents" read-only sidebar tree view (`nexus.workspaceAgents`) showing orchestration state from `.claude/orchestrator.json` in all open workspace folders
- VS Code extension: `nexus.refreshWorkspaceAgents` command with toolbar refresh button
- VS Code extension: `WorkspaceScanner` — file-system watcher on `.claude/orchestrator.json` per workspace folder; auto-refreshes the tree on file changes
- `DiscoveryService`: `InvalidateHealthCache()` public method; called by `TriggerScan` so a manual scan always returns fresh provider data

### Fixed

- Provider health-check over-polling: `DiscoveryService.ListProviders()` previously called `Ping()` (and `GetAvailableModels()`) on every request; now uses a 30-second TTL cache per provider
- Circuit-breaker backoff: providers with 3+ consecutive `Ping()` failures are checked with exponentially increasing intervals (30s → 60s → 120s → … capped at 10 min), preventing repeated hammering of unreachable backends
- `FindForModel()` no longer calls `GetAvailableModels()` live on each candidate — uses cached model lists

### Changed

- `frontend/src/composables/useProviders.ts`: polling interval 5 s → 30 s (matches backend cache TTL)
- `vscode-extension/src/extension.ts`: status bar polling interval 10 s → 30 s (status bar calls `/api/providers` which is now cached)

## [0.9.3] — 2026-03-12

### Added

- Wails desktop bindings for AI Sessions (`ListAISessions`, `RegisterAISession`, `DeregisterAISession`) in `app.go`
- CORS middleware in HTTP API for Wails WebView and local browser access (`wails://wails.localhost`, `http://localhost:*`)
- `registerAISession()` function in `wails.ts` (previously missing entirely)
- VS Code extension: `OutputChannel` for `Nexus Orchestrator` — session monitor logs visible in Output panel
- `dist/desktop/` and `dist/vscode/` centralized output directories for GUI and VS Code VSIX

### Fixed

- AI Sessions view shows "Load failed" — `listAISessions`/`deregisterAISession` in `wails.ts` were making raw hardcoded HTTP calls (`http://127.0.0.1:63987`) bypassing the Wails IPC binding pattern; fixed to use `isWails()` with proper `window.go` bindings
- Task Queue view rendered black — `DashboardView` used `h-full` inside a flex container with `padding-bottom: 208px`, and the task list `div` lacked `min-h-0`, preventing `overflow-auto` from activating; `TaskSubmitForm` lacked `flex-shrink-0` and was collapsed to zero height
- VS Code Copilot not recognized — `SessionMonitor.detectAndRegister()` was called once at activation with no retry; added exponential backoff (2 s / 5 s / 10 s) so Copilot models are detected even when the extension host activates before Copilot finishes loading
- `make build-all` now builds all artifacts: CLI/daemon (all platforms) + Wails GUI + VS Code extension + frontend

### Changed

- Makefile: added `DIST_DESKTOP` and `DIST_VSCODE` variables; `build-gui` copies Wails output to `dist/desktop/`; `build-vscode` outputs VSIX to `dist/vscode/`; `build-all` includes `build-frontend build-vscode build-gui`
- `.github/workflows/publish.yml`: all `build/bin/` artifact paths → `dist/desktop/`; VSIX path → `dist/vscode/nexus-orchestrator.vsix`
- `.gitignore`: added `/dist/desktop/` and `/dist/vscode/`

## [0.9.2] - 2026-03-12

### Added

- GUI **Task History** view (`HistoryView.vue`) — browse completed/failed/cancelled tasks with status filter (All / Completed / Failed / Cancelled) and `TaskDetailDrawer` detail panel; sidebar "History" nav item
- GUI **Settings** view (`SettingsView.vue`) — Provider Connections section (list/add/edit/remove provider configs), Queue cap display, Server addresses with copy-to-clipboard; sidebar "Settings" nav item

### Changed

- README Features section rewritten as a grouped matrix covering all v0.9.x capabilities across 6 categories (Core, LLM Backends, Provider Discovery, Task Management, Interfaces, Observability)
- README Dogfooding section replaced with generic 4-step workflow; removed stale PLAN-002 task file references
- README Quick Start: added VS Code Extension install subsection cross-linking `vscode-extension/README.md`

### Fixed

- **Orchestrator startup recovery** — tasks stuck in `PROCESSING` at daemon crash are automatically re-queued on next startup (`recoverStuckTasks()` in `NewOrchestrator()`)
- **Path normalization** — `ProjectPath` is now cleaned to an absolute path before storage (prevents relative-path task lookup mismatches)
- **Queue cap** — `SubmitTask()` returns `ErrQueueFull` when queued task count ≥ cap (default 50, configurable via `WithQueueCap(n)`)
- **Retry limit** — failing LLM calls are retried up to 3 times (`maxRetries = 3`) before task moves to `StatusFailed`; `RetryCount` persisted on `domain.Task`

## [0.9.1] - 2026-03-12

### Fixed

- `make build-all` cross-compilation now works with zig 0.15.x — added `-tags netgo,osusergo` and `-extldflags='-static'` to Linux targets, eliminating musl `__errno_location` / `pthread_*` linker errors
- VS Code extension now auto-registers the `nexus-orchest` MCP server via `contributes.mcpServers` (VS Code 1.99+) — no manual `.vscode/mcp.json` required
- VS Code extension rebuilt at v0.2.0: bundles `SessionMonitor` (PLAN-022) and AI session auto-registration with daemon

### Changed

- VS Code extension minimum engine version bumped to `^1.99.0` to support `contributes.mcpServers`
- Linux cross-compilation uses `-tags netgo,osusergo` (pure-Go DNS/user) and `-extldflags='-static'` (static musl) to support zig 0.15.x

## [0.9.0] - 2026-03-12

### Added

#### Universal AI Session Orchestration (PLAN-022)

- `AISession` domain type with `AISessionStatus` (`active`/`idle`/`disconnected`) and `AISessionSource` (`mcp`/`vscode`/`http`) enums; `IsTerminal()` predicate on status
- `AISessionRepository` outbound port (5 methods: save, get, list, update-status, delete) and `AISessionMonitor` inbound port
- SQLite `ai_sessions` table with additive schema migration; `AISessionRepo` outbound adapter sharing the existing `*sql.DB`
- `RegisterAISession`, `ListAISessions`, `DeregisterAISession` methods on `OrchestratorService` and the `Orchestrator` port interface
- HTTP API: `POST /api/ai-sessions` (201), `GET /api/ai-sessions` (200), `DELETE /api/ai-sessions/{id}` (204/404) — external AI agents can self-register
- MCP tools `register_session` and `get_ai_sessions` (total: 14 tools) — MCP clients announce themselves via JSON-RPC 2.0
- GUI "AI Sessions" view (`AISessionsView.vue`) with SSE real-time updates and 5s polling fallback; status-coloured session cards; "Disconnect" action; sidebar nav item
- VS Code extension `SessionMonitor` — detects GitHub Copilot via `vscode.lm.selectChatModels`, auto-registers with daemon, 60s heartbeat, graceful deregister on extension deactivate; session count added to status bar tooltip
- `AISessionRepo` wired into both daemon (`cmd/nexus-daemon/main.go`) and Wails desktop (`main.go`) entry points

#### System-Wide Provider Discovery + Background Service Mode (PLAN-023)

- `DiscoveredProvider` domain type with `DiscoveryMethod` (`port`/`cli`/`process`) and `DiscoveryStatus` (`reachable`/`installed`/`running`) enums
- `SystemScanner` outbound port; `sys_scanner` adapter probes 6 TCP ports, 5 CLI tools, and 4 running processes with 8-goroutine semaphore and 5s deadline
- `GetDiscoveredProviders`, `TriggerScan`, `PromoteProvider` methods on the `Orchestrator` port and service; 30s periodic re-scan (configurable via `NEXUS_SCAN_INTERVAL`)
- HTTP API: `GET /api/providers/discovered`, `POST /api/providers/discovered/scan`, `POST /api/providers/promote/{id}`
- MCP tools `discover_providers` and `promote_provider`
- In-app log console (`LogHub` SSE fan-out with 500-entry ring buffer; `GET /api/logs`; `LogPanel.vue` with drag-resize, level filter, auto-scroll)
- `DiscoveryView.vue` with provider discovery panel and sidebar nav
- Wails `HideWindowOnClose: true` — desktop app hides to tray instead of quitting; `OnBeforeClose` hook

#### Multi-Project Planning & Idea Staging (PLAN-024)

- `DRAFT` and `BACKLOG` task statuses; `Priority` (`critical`/`high`/`medium`/`low`) and `Tags []string` fields on `Task`; `ProviderName` per-task routing override
- SQLite additive migrations for new columns
- Backlog CRUD on HTTP API (`POST /api/tasks/draft`, `GET /api/tasks/backlog`, `POST /api/tasks/promote/{id}`), MCP, CLI, and Wails binding
- `BacklogView.vue`, `BacklogList.vue`, `ProjectSelector.vue` GUI components; `TaskSubmitForm.vue` split-button for submit vs. save-to-backlog

### Fixed

- MCP JSON Schema: `"type": "array"` properties (`contextFiles`, `tags`) now correctly include `"items": {"type": "string"}` — fixes Copilot/Claude Desktop tool validation error
- All download links on docs site now resolve correctly — prefix corrected from
  `nexusOrchestrator-*` to `nexus-orchestrator-*` to match actual GitHub Release artifact names
- macOS Desktop download links now point to `.zip` format (matching pipeline output);
  previously linked to `.tar.gz` which would 404
- Checksum verification examples on downloads page now reference correct file names

### Added (standalone)

- macOS Gatekeeper / quarantine workaround instructions on Downloads page
  (prominent warning section explaining "Apple could not verify" is expected for
  unsigned open-source apps, with right-click and `xattr` solutions)
- macOS first-run setup instructions added to Getting Started guide

## [0.8.0] - 2026-03-12

### Added

#### Provider Management (PLAN-018)

- Configurable Ollama and LM Studio base URLs — override defaults via `NEXUS_OLLAMA_BASE_URL` and `NEXUS_LMSTUDIO_BASE_URL` environment variables; also configurable in the GUI
- `BaseURL` and `Error` fields added to `ProviderInfo` so the frontend can display the live endpoint and error state
- SQLite-persisted provider CRUD (`ProviderConfigRepository` port + `repo_sqlite` implementation)
- Providers management panel in the Wails GUI: add, edit, delete configured providers with preset URLs for LM Studio, Ollama, OpenAI, Anthropic, and custom OpenAI-compatible endpoints
- API key field masked (`***`) in all HTTP responses to prevent credential leakage
- VS Code extension (`vscode-extension/`) with task submit, task queue view, and status bar provider indicator

#### Release Pipeline (PLAN-019)

- `@vscode/vsce` packaging for the VS Code extension added to `publish.yml` — VSIX artifact included in every GitHub Release
- `build-vscode` job added to `ci.yml` for smoke-test validation of the extension on every PR
- VS Code extension download section added to the docs Downloads page and Homepage feature list

### Fixed

#### GUI Provider Display (PLAN-020)

- `ProviderInfo` TypeScript fields corrected from PascalCase (`Name`, `Active`, `ActiveModel`, `Models`) to camelCase (`name`, `active`, `activeModel`, `models`) to match Go JSON serialisation — provider cards were rendering blank before this fix
- `AppSidebar.vue` navigation wired with `defineEmits` — clicking "Providers" now correctly switches the view
- `ProvidersView.vue` created as a full-page provider management view (discovered providers grid + configured provider CRUD)

## [0.2.0] - 2026-03-11

### Added

#### MCP Server & Session Isolation (PLAN-001)

- MCP JSON-RPC 2.0 inbound adapter (`internal/adapters/inbound/mcp/`) listening on `:63988`
- 6 MCP tools: `submit_task`, `get_task`, `get_queue`, `cancel_task`, `get_providers`, `health`
- Per-project session isolation via `domain.Session` and `domain.Message` types
- `SessionRepository` port and SQLite implementation (`repo_sqlite.SessionRepo`)
- `Chat([]domain.Message)` multi-turn method on all LLM adapters
- `OrchestratorService` fallback to `GenerateCode()` when `sessionRepo == nil`

#### Context-Window Guard (PLAN-004)

- Token pre-flight check that blocks tasks exceeding the model's context window
- `StatusTooLarge` domain constant for rejected oversized tasks
- `GetContextWindowSize()` method on the `LLMClient` port

#### Smart Multi-Provider Routing (PLAN-005)

- `ModelID` and `ProviderHint` fields on `domain.Task` for targeted routing
- `FindForModel()` with automatic failover in `DiscoveryService`
- `llm_openaicompat` outbound adapter (OpenAI / GitHub Copilot / Azure OpenAI)
- `llm_anthropic` outbound adapter (Anthropic Claude)
- `StatusNoProvider` domain constant for tasks with no available provider
- Environment variable provider configuration: `NEXUS_OPENAI_BASE_URL`, `NEXUS_OPENAI_API_KEY`, `NEXUS_ANTHROPIC_API_KEY`

#### UI Provider & Model Control (PLAN-006)

- `ProviderConfig` domain type for runtime cloud provider registration
- `RegisterCloudProvider`, `RemoveProvider`, `GetProviderModels` methods on `Orchestrator` port
- HTTP CRUD endpoints: `POST /api/providers/{id}`, `GET /api/providers/{id}`, `DELETE /api/providers/{id}`
- Task submission form and provider management panel in the Wails GUI dashboard

#### E2E & Unit Tests (PLAN-008)

- MCP integration tests
- HTTP API coverage tests (provider CRUD, cancel, error paths)
- SSE event lifecycle tests
- Process-level smoke test script (`scripts/e2e-smoke.sh`)

#### GitHub Pages Documentation (PLAN-009)

- GitHub Pages documentation site covering architecture, API reference, getting started, and MCP integration
- `CommandType` field on `domain.Task` (`plan` / `execute` / `auto`) with smart validation in the orchestrator
- `CommandType` support in the HTTP API and MCP `submit_task` tool

#### Cross-Platform Release Pipeline (PLAN-010)

- GitHub Actions release pipeline producing CLI and daemon binaries for 5 platforms and Wails desktop for 4 platforms
- macOS code signing and notarization support (activated when `APPLE_CERTIFICATE` secret is set)
- `scripts/install.sh` one-line installer
- Downloads landing page on GitHub Pages

#### Industry-Grade Hardening (PLAN-011)

- Version injection via `ldflags -X main.version` for all binary builds
- Checksum verification and improved error handling in `install.sh`

#### Semantic Versioning (PLAN-012)

- `GitVersion.yml` configuration for GitVersion 6.x semantic versioning
- MIT `LICENSE` file

#### Unified Publish Pipeline (PLAN-014)

- `publish.yml` unified workflow replacing three separate workflows; eliminates `GITHUB_TOKEN` cross-trigger bug
- Desktop archive naming uses `-desktop-` prefix to avoid collision with CLI artifacts

#### Production GitHub Action (PLAN-015)

- `github-action/` — production-grade Node20/TypeScript GitHub Action for submitting tasks from CI
- Built-in agent identity loading from `el-j/agency-agents`
- 24 unit tests covering action logic

#### Release Finalization (PLAN-016)

- `CHANGELOG.md` (this file)

### Changed

#### CI Action Versions (PLAN-013)

- Updated all GitHub Actions to latest stable versions: `gittools/actions@v4.3.3`, zig 0.14.0 cross-compiler

### Fixed

#### Security Hardening (PLAN-007)

- Path traversal vulnerability in `fs_writer`: attacker-controlled filenames could escape the project directory
- Arbitrary file read in `ReadContextFiles()` via the same path traversal vector
- HTTP request body size limit added to prevent DoS via oversized payloads
- SQLite hardening: WAL journal mode, `PRAGMA busy_timeout`, `PRAGMA foreign_keys=ON`, connection pool limits applied
- XSS vulnerability in HTML dashboard (`onclick` handler JavaScript string context was unescaped)
- Goroutine lifecycle moved out of `internal/core/services/` into inbound adapter layer (hexagonal architecture compliance)
- Systematic error swallowing in `processNext()`: all `repo.UpdateStatus` / `repo.UpdateLogs` failures are now logged
- Post-`Stop()` use-after-free: `SubmitTask()` now returns `ErrStopped` after shutdown

#### Release Pipeline (PLAN-016)

- Removed `version.yml` and `release.yml` (superseded by `publish.yml`); fixes release pipeline trigger regression

### Security

- Resolved path traversal in `fs_writer` and `ReadContextFiles()` — untrusted input is now validated against the project root before any file operation
- HTTP body size cap prevents memory exhaustion from malformed or malicious API requests
- SQLite `PRAGMA foreign_keys=ON` prevents referential integrity violations; WAL mode reduces write contention

[Unreleased]: https://github.com/el-j/nexus-orchestrator/compare/v0.9.2...HEAD
[0.9.2]: https://github.com/el-j/nexus-orchestrator/compare/v0.9.1...v0.9.2
[0.9.1]: https://github.com/el-j/nexus-orchestrator/compare/v0.9.0...v0.9.1
[0.9.0]: https://github.com/el-j/nexus-orchestrator/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/el-j/nexus-orchestrator/releases/tag/v0.8.0
[0.2.0]: https://github.com/el-j/nexus-orchestrator/releases/tag/v0.2.0
