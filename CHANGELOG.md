# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-03-11

### Added

#### MCP Server & Session Isolation (PLAN-001)
- MCP JSON-RPC 2.0 inbound adapter (`internal/adapters/inbound/mcp/`) listening on `:9998`
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

[Unreleased]: https://github.com/el-j/nexusOrchestrator/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/el-j/nexusOrchestrator/releases/tag/v0.2.0
