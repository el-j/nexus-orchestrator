---
id: TASK-292
title: 'CHANGELOG.md — document v0.10.0 (PLAN-044 + pre-commit + polish)'
role: docs
planId: PLAN-045
status: todo
dependencies: []
createdAt: 2026-03-14T18:00:00.000Z
---

## Context

`CHANGELOG.md` last documented release is `[0.9.4] — 2026-03-12`. Since then two major
work streams shipped:

1. **PLAN-044 — Universal AI Takeover** (2026-03-14): entire Go + TS + Vue feature set
   adding AI agent detection, session delegation, `DiscoveredAgent` domain type, etc.
2. **PLAN-045 polish** (2026-03-14): pre-commit hooks (husky + lint-staged + prettier),
   shared `frontend/src/utils/time.ts`, `domain.ts` type fixes.

The `[Unreleased]` section is currently empty.

## Implementation Steps

1. Replace the empty `## [Unreleased]` line with a full `## [0.10.0] — 2026-03-14` section.

2. The entry should use standard Keep a Changelog format with sub-headers:
   `### Added`, `### Changed`, `### Fixed`

3. Content to include under `### Added`:
   - `DiscoveredAgent` domain type (`AgentKind` with 10 constants: claude-cli,
     claude-desktop, antigravity, cline, continue, codegpt, cursor, copilot, aichat, generic)
   - `AgentScanner` outbound port + `ports.Orchestrator` methods `GetDiscoveredAgents`,
     `DelegateToNexus`
   - `sys_scanner` adapter: 5-strategy `ScanAgents` (fs-config, vscode-ext, MCP port
     sweep 8 ports, process-flag, process)
   - SQLite `discovered_agents` table + 4 new `ai_sessions` columns
     (delegated_to_nexus, delegation_timestamp, agent_capabilities, detection_method)
   - HTTP API: `GET /api/ai-sessions/discovered`, `POST /api/ai-sessions/{id}/delegate`
   - VS Code extension: `AgentDetector` (4-strategy, 30 s poll), `AISessionsTreeProvider`
     (`nexus.aiSessions` view), `nexus.delegateToNexus` command (3 delivery paths:
     cli / mcp / copilot), `nexus.delegateAllSessions` command
   - Frontend `AIAgentsView.vue` + `AISessionCard.vue` with live task timeline
   - Root-level pre-commit tooling: husky v9, lint-staged, prettier (gofmt on `.go`,
     prettier on TS/Vue/JSON/MD)

4. Content to include under `### Changed`:
   - `OrchestratorService` gains `SetAgentScanner` + `SetDiscoveredAgentRepo` setters;
     `GetDiscoveredAgents` uses 30 s TTL cache
   - `domain.ts` `Task` type: added `retryCount?: number`
   - `frontend/src/utils/time.ts`: centralised `relativeTime` / `timeAgo` / `formatDate`;
     removed duplicates from 6 Vue files

5. Restore `## [Unreleased]` as a blank section _above_ the new version entry so it is
   ready for future changes.

## Acceptance Criteria

- [ ] `## [Unreleased]` blank section exists at top of changelog (above new version)
- [ ] `## [0.10.0] — 2026-03-14` entry present
- [ ] At minimum 10 bullet points across Added/Changed/Fixed
- [ ] Markdown renders without lint errors
