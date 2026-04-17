---
id: PLAN-071
title: 'Comprehensive Audit Resolution — All 116 Findings'
goal: 'Systematically resolve every finding from the 2026-04-15 five-domain audit: CI pipeline breakage, Go silenced errors, frontend safety gaps, VS Code leaks and missing validation, CLI surface gaps, architecture violations, hardcoded values, dead code, and test coverage deficits.'
status: todo
createdAt: 2026-04-15T00:00:00Z
---

# PLAN-071 — Comprehensive Audit Resolution

## Background

A five-agent parallel audit on 2026-04-15 produced 116 findings across Go backend, Vue 3/TypeScript frontend, VS Code extension, CLI/build/DevOps, and test coverage. Full findings in `.claude/audits/AUDIT-2026-04-15.md`. This plan resolves all findings in dependency order.

## Task Map

| Task     | Wave | Layer    | Priority | Description                                                                                                                |
| -------- | ---- | -------- | -------- | -------------------------------------------------------------------------------------------------------------------------- |
| TASK-554 | 1    | devops   | P0       | Fix CI release pipeline: broken download-artifact@v8, CGO_ENABLED=0 for daemon, enable marketplace publish                 |
| TASK-555 | 1    | backend  | P0       | Fix all 12 silenced Go errors: proc.Kill/Signal, planFileRepo, MCP json.Encode×7, MCP Unmarshal×2, SSE×3                   |
| TASK-556 | 1    | frontend | P0       | Fix frontend critical safety: r.ok in wails.ts×4, p.baseURL null-guard, BrainView ingest path, SettingsView loadConfig     |
| TASK-557 | 2    | backend  | P1       | Fix App architecture: ActivityService concrete→port interface, constructor panics→wiring, httpapi_client context           |
| TASK-558 | 2    | api      | P1       | Fix handleGetSessionTasks O(n) scan → GetTasksBySessionID                                                                  |
| TASK-559 | 3    | backend  | P2       | Extract Go hardcoded constants: daemonAddr, queue cap, purge/watchdog timeouts, MCP version from ldflags                   |
| TASK-560 | 3    | backend  | P2       | Fix LLM context limits: expand claudeContextLimits, fix openaicompat flat 131072; delete tray stub package                 |
| TASK-561 | 4    | frontend | P2       | Wire orphan views into router + fix promote handlers: DiscoveryView, AISessionsView, DiscoveredPlansView                   |
| TASK-562 | 4    | frontend | P2       | Fix frontend leaks + as any: timer cleanup, reconnectTimer, 4× as any casts, MCP URL, agentStatusFilter UI                 |
| TASK-563 | 5    | vscode   | P1       | Add Zod schema validation to all VS Code nexusClient.ts API response paths                                                 |
| TASK-564 | 5    | vscode   | P1       | Fix VS Code memory leaks + unsafe error handling: double poller, sessionMonitor timers, instanceof guards, unawaited .then |
| TASK-565 | 5    | vscode   | P2       | Wire VS Code dead client methods to commands + add menu bindings for brain commands                                        |
| TASK-566 | 6    | cli      | P2       | CLI expansion: --daemon-url/NEXUS_ADDR flag + full ai-sessions command group                                               |
| TASK-567 | 6    | cli      | P2       | CLI expansion: providers config CRUD + discovered/scan/promote subcommands                                                 |
| TASK-568 | 6    | cli      | P2       | CLI expansion: config get/set, plans discovered, logs, brain focused-context, tasks claim/heartbeat                        |
| TASK-569 | 7    | qa       | P3       | Go repo tests: activity_repo (complex dynamic SQL), runtime_config_repo (daemon-startup critical)                          |
| TASK-570 | 7    | qa       | P3       | Go repo tests: model_capability_repo, plan_file_repo — full CRUD coverage                                                  |
| TASK-571 | 7    | qa       | P3       | MCP tool tests: provider config CRUD×4, session lifecycle×3, howto/howto_brief×2                                           |
| TASK-572 | 8    | qa       | P3       | Frontend composable tests: useBrain, useDaemonHealth, useDiscovery, useLogs                                                |
| TASK-573 | 8    | qa       | P3       | Frontend composable tests: useProjectFilter, useProviderActivity, useProviders, useServerUrl                               |
| TASK-574 | 8    | qa       | P3       | VS Code extension tests: nexusClient (recently modified), extension activation, AISessionsTreeProvider                     |
| TASK-575 | 9    | devops   | P3       | CI quality: coverage gate + coverprofile, remove E2E continue-on-error, add Makefile targets                               |
| TASK-576 | 9    | qa       | P3       | Replace time.Sleep synchronisation in 7 Go test files with channel/sync primitives                                         |

## Waves

### Wave 1 — P0 Ship Blockers (TASK-554, TASK-555, TASK-556) — independent, run in parallel

Fix the broken release pipeline, all silenced Go errors, and critical frontend safety holes before any other work.

### Wave 2 — Architecture Corrections (TASK-557, TASK-558) — after Wave 1

Fix the hexagonal boundary violation in app.go and the O(n) session task scan.

### Wave 3 — Hardcoded Values + Dead Code (TASK-559, TASK-560) — after Wave 2

Extract magic literals to named constants/config and remove the dead tray stub.

### Wave 4 — Frontend Gaps (TASK-561, TASK-562) — after Wave 1

Wire unreachable views, fix promote handlers, clear timer leaks and unsafe casts.

### Wave 5 — VS Code Extension (TASK-563, TASK-564, TASK-565) — after Wave 1; TASK-565 after 563+564

Add Zod validation, fix memory leaks, wire dead client methods to UI.

### Wave 6 — CLI Expansion (TASK-566, TASK-567, TASK-568) — after Wave 2; 567+568 after 566

Expand CLI to cover the remaining ~80% of the HTTP API surface.

### Wave 7 — Go Repository + MCP Tests (TASK-569, TASK-570, TASK-571) — after Wave 2

Cover the four zero-test SQLite repos and 11 untested MCP tools.

### Wave 8 — Frontend + VS Code Tests (TASK-572, TASK-573, TASK-574) — after Waves 4+5

Cover 8 frontend composables and the recently-modified VS Code client.

### Wave 9 — CI Quality Gates + Test Reliability (TASK-575, TASK-576) — after Waves 1+7

Add coverage gate, harden E2E gate, and eliminate time.Sleep flakiness.
