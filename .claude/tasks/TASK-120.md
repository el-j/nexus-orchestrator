---
id: TASK-120
title: Create CHANGELOG.md following Keep-a-Changelog convention
role: devops
planId: PLAN-016
status: todo
dependencies: []
createdAt: 2026-03-11T00:00:00.000Z
---

## Context

No `CHANGELOG.md` exists at the repository root. A changelog is required for:
1. Human-readable release history visible in GitHub Releases
2. Consumer reference for API/behaviour changes between versions
3. Convention: `softprops/action-gh-release@v2` can optionally extract notes from it

## Format

Use [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format.
Sections per release: `Added`, `Changed`, `Fixed`, `Security`, `Removed`.

## History to document

| Plan | Features |
|------|----------|
| PLAN-001 | MCP server (JSON-RPC 2.0), per-project session isolation, `SessionRepository`, `Chat()` on all LLM adapters |
| PLAN-004 | Context-window guard (token pre-flight check, `StatusTooLarge`) |
| PLAN-005 | Smart multi-provider routing (`ModelID`/`ProviderHint`, `FindForModel()` with failover, `llm_openaicompat`, `llm_anthropic`) |
| PLAN-006 | UI provider + model control (`RegisterCloudProvider`, `RemoveProvider`, HTTP CRUD `/api/providers`, Wails binding) |
| PLAN-007 | Security hardening: path traversal fix (fs_writer), request body limits, SQLite WAL+FK+busy_timeout, XSS fix in dashboard, goroutine lifecycle moved to adapters |
| PLAN-008 | Comprehensive E2E + unit tests (MCP, HTTP API, provider CRUD, cancel, SSE lifecycle) |
| PLAN-009 | GitHub Pages documentation site, command-aware task routing (`CommandType`: plan/execute/auto) |
| PLAN-010 | Cross-platform release pipeline (GitHub Actions), Wails desktop for macOS/Windows/Linux, install.sh |
| PLAN-011 | Industry-grade hardening: version injection (`-X main.version`), improved install script resilience |
| PLAN-012 | Semantic versioning (GitVersion), MIT license, zig 0.14.0 cross-compiler |
| PLAN-013 | CI updated to latest action versions (`gittools/actions@v4.3.3`, checkout/setup-go@v6) |
| PLAN-014 | Unified `publish.yml` pipeline (replaces version.yml + release.yml + desktop.yml); fixed GITHUB_TOKEN cross-trigger bug |
| PLAN-015 | Production-grade Node20 GitHub Action with TypeScript, agent identity loading, 24 unit tests |
| PLAN-016 | Release pipeline finalization (delete conflicting workflows, fix downloads.md, ship CHANGELOG) |

## Definition of Done

- `CHANGELOG.md` exists at repo root
- Follows Keep-a-Changelog format with `## [Unreleased]` at top
- Has a `## [0.2.0]` section grouping all shipped features
- Has `[Unreleased]: https://github.com/el-j/nexusOrchestrator/compare/v0.2.0...HEAD` link
