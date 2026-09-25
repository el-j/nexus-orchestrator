---
id: TASK-564
planId: PLAN-072
title: 'CLI & Daemon Entry Point Test Coverage'
role: qa
status: done
createdAt: 2026-09-25T10:05:00Z
completedAt: 2026-09-25T10:40:00Z
---

# TASK-564 — CLI & Daemon Entry Point Test Coverage

## Context

The audit revealed that `cmd/nexus-cli`, `cmd/nexus-daemon`, `cmd/nexus-mcp-stdio`, `cmd/nexus-submit`, and `internal/bootstrap` had 0.0% package test status. This posed regression risks for flags, initialization sequences, and command routing.

## Completed Work

1. **Created Comprehensive Test Suites**:
   - `internal/bootstrap/providers_test.go`: tested `BuildProviders` (Gemini, Anthropic, OpenAI, Local), `BuildProviderFromConfig` across all kinds, with 72.2% statement coverage.
   - `cmd/nexus-submit/main_test.go`: tested `BuildRequestBody` (`--verify`, `--turns`, `--project`, `--target`, context files), `SubmitTask` HTTP response handling, and `getEnv`.
   - `cmd/nexus-cli/main_test.go`: tested CLI root command construction, version flag, and subcommand discovery (`tasks`, `providers`, `brain`, `status`).
   - `cmd/nexus-mcp-stdio/main_test.go`: tested stdio proxy with live mock HTTP server, token authentication header forwarding, and connection error JSON-RPC formatting.
   - `cmd/nexus-daemon/main_test.go`: tested `resolveConfig` (defaults and environment variable overrides) and `formatBanner`.
2. **Repository-Wide Coverage**:
   - All 4 `cmd/` packages now execute automated unit tests with `-race -count=1`.
   - Entire Go codebase (26 test packages) passes 100% with race detector enabled.
