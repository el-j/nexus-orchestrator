---
id: PLAN-072
title: 'Universal Control Plane & Context Broker Foundation'
goal: 'Transform Nexus Orchestrator into the ultra-fast, local-first Control Plane and Context Broker for heterogeneous agents: refactor monolithic MCP tool dispatcher, implement fast token-budgeted onboarding context handshake, add native Google Gemini adapter and provider failover, add real-time filesystem auto-invalidation, and eliminate entry-point test gaps.'
status: completed
createdAt: 2026-09-25T10:05:00Z
completedAt: 2026-09-25T10:44:00Z
---

# PLAN-072 — Universal Control Plane & Context Broker Foundation

## Background & Architecture Strategy

Following the architectural assessment, Nexus Orchestrator will focus on being the **missing local infrastructure layer**: an ultra-fast, local-first Control Plane and Memory Server that plugs into external agents (Claude Code, Cursor, Aider, Cline) via MCP, tracks state in SQLite, and provides deterministic execution gates and fresh, compact context.

This plan addresses the concrete architectural gaps identified in the audit:

1. **Boss File Refactoring:** The monolithic `internal/adapters/inbound/mcp/tools.go` (1,241 lines) is split into clean domain submodules.
2. **Onboarding Context Handshake:** A dedicated, token-budgeted (< 800 token) onboarding payload synthesizer (`get_onboarding_context`) delivering immediate Tier 0/1 topology, invariants, and active task state without repo dumping.
3. **Provider Matrix Expansion:** Native `llm_gemini` outbound adapter using Google GenAI API, plus a smart provider fallback chain (`Frontier Cloud` $\to$ `Fallback Cloud` $\to$ `Local Ollama`).
4. **Active Workspace Auto-Invalidation:** A debounced `fsnotify` filesystem watcher keeping SQLite FTS5 index fresh when files are modified externally.
5. **Entrypoint Test Coverage & Polish:** Unit tests for `cmd/*` and `internal/bootstrap` to eliminate 0% coverage areas, plus UI/docs polish.

## Task Map

| Task         | Wave | Domain    | Priority | Description                                                                                                                             |
| :----------- | :--- | :-------- | :------- | :-------------------------------------------------------------------------------------------------------------------------------------- |
| **TASK-559** | 1    | mcp       | High     | Decompose monolithic `mcp/tools.go` into domain modules (`tools_tasks.go`, `tools_brain.go`, `tools_providers.go`, `tools_sessions.go`) |
| **TASK-560** | 1    | brain/mcp | High     | Token-Budgeted Project Onboarding Handshake (`get_onboarding_context` MCP tool + HTTP endpoint)                                         |
| **TASK-561** | 2    | llm       | High     | Native Google Gemini LLM Outbound Adapter (`internal/adapters/outbound/llm_gemini`)                                                     |
| **TASK-562** | 2    | services  | High     | Smart Provider Fallback Chain & Role-Driven Router (Frontier $\to$ Cloud Fallback $\to$ Local Ollama)                                   |
| **TASK-563** | 3    | scanner   | Medium   | Real-Time Workspace Change Invalidation Watcher (`fsnotify` debounced background re-indexer)                                            |
| **TASK-564** | 4    | qa        | Medium   | CLI & Daemon Entry Point Test Coverage (`cmd/nexus-cli`, `cmd/nexus-daemon`, `cmd/nexus-submit`, `internal/bootstrap`)                  |
| **TASK-565** | 4    | polish    | Medium   | UI & Docs Polish: Wire `DiscoveryView.vue` to `ProviderConfigForm`, remove screenshot placeholder, update `README.md` to v0.10.0        |

## Waves

- **Wave 1 — Architecture Cleanup & Context Broker Protocol (`TASK-559` – `TASK-560`)**:
  Refactors the 1,241-line MCP god file and establishes the ultra-compact onboarding context injection protocol.
- **Wave 2 — Cloud Provider Matrix & Fallback Router (`TASK-561` – `TASK-562`)**:
  Adds native Google Gemini support and multi-provider failover chains.
- **Wave 3 — Real-Time Cache Invalidation (`TASK-563`)**:
  Implements the background filesystem watcher to solve context staleness.
- **Wave 4 — Test Coverage & Polish (`TASK-564` – `TASK-565`)**:
  Covers the 0% `cmd/` packages and closes outstanding UI/documentation gaps.

## Acceptance Criteria

- `tools.go` decomposed; all individual files $< 450$ lines; all 40 MCP tools functional.
- `get_onboarding_context` MCP tool returns Tier 0/1 summary within specified token budget (default 800 tokens).
- Google Gemini models callable directly via `llm_gemini` adapter.
- Provider failover automatically falls back to secondary when primary fails or rate-limits.
- Modifying a watched source file updates the knowledge index without manual intervention.
- `cmd/` packages have unit test coverage $\ge 50\%$.
- All Go test suites pass with `-race -count=1`.
