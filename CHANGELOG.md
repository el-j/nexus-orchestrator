# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/el-j/nexusOrchestrator/compare/v0.9.2...HEAD
[0.9.2]: https://github.com/el-j/nexusOrchestrator/compare/v0.9.1...v0.9.2
[0.9.1]: https://github.com/el-j/nexusOrchestrator/compare/v0.9.0...v0.9.1
[0.9.0]: https://github.com/el-j/nexusOrchestrator/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/el-j/nexusOrchestrator/releases/tag/v0.8.0
[0.2.0]: https://github.com/el-j/nexusOrchestrator/releases/tag/v0.2.0
