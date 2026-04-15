---
id: PLAN-069
title: 'Brain Layer Gap-Fill — API, Client, CLI, Wails, Tests, Docs'
goal: 'Close all confirmed gaps in the Nexus Brain implementation: missing HTTP routes, stub client methods, Wails binding bug, howto_brief phantom params, missing CLI subcommands, and absent test coverage across all layers.'
status: todo
createdAt: 2026-04-13T16:00:00Z
---

# PLAN-069 — Brain Layer Gap-Fill

## Background

A swarm audit (4 parallel agents) completed on 2026-04-13 after PLAN-066/067/068 landed the Brain
Project Context Intelligence layer. The audit found 11 confirmed gaps across backend, client,
desktop, CLI, MCP documentation, and tests. No code was written; this plan captures the full
remediation backlog.

## Confirmed Gaps

| #   | Layer           | Symptom                                                                                                                                            |
| --- | --------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | HTTP API        | Missing routes: `POST /api/brain/init`, `GET /api/brain/knowledge`, `DELETE /api/brain/knowledge/{id}`, `GET /api/brain/file-map`                  |
| 2   | Outbound client | 4 `brain_client.go` methods always return `"not implemented in client"`                                                                            |
| 3   | Wails app.go    | `GetFocusedContext` calls `GetContext` instead of `GetFocusedContext`; `WithBrainService`/`WithActivityService` are exported (exposed to Wails JS) |
| 4   | MCP howto_brief | Wrong parameter names: `project_path`, `task_id`, `session_id` instead of `projectPath`, `id`, `sessionId`                                         |
| 5   | CLI             | Missing brain subcommands: `init`, `list`, `delete`, `context`, `file-map`                                                                         |
| 6   | HTTP docs       | `buildHowToDoc` does not list the 4 missing brain routes                                                                                           |
| 7   | Tests           | No `brain_handlers_test.go` exists — zero HTTP handler coverage                                                                                    |
| 8   | Tests           | No SQLite knowledge repo tests — FTS5 ranking, upsert dedup, project isolation untested                                                            |
| 9   | Tests           | `brain_service_test.go` missing `SearchKnowledge`, `InitProject`, `GetStatus` paths                                                                |
| 10  | E2E             | `brain_test.go` covers ingest/status/context/focused but missing `search_knowledge` MCP + re-ingest dedup                                          |
| 11  | Quality         | SQLite `SearchFTS` has implicit BM25 ordering; `UpdateKnowledge` does not recompute `token_count`                                                  |

## Task Map

| Task     | Wave | Layer   | Priority | Description                                                               |
| -------- | ---- | ------- | -------- | ------------------------------------------------------------------------- |
| TASK-529 | 1    | backend | Critical | Add 4 missing brain HTTP routes + handlers                                |
| TASK-530 | 1    | backend | Critical | Implement 4 stub methods in brain_client.go                               |
| TASK-531 | 1    | backend | Critical | Fix app.go GetFocusedContext bug + unexport With\*Service                 |
| TASK-532 | 1    | mcp     | High     | Fix howto_brief phantom parameter names                                   |
| TASK-533 | 2    | cli     | High     | Add missing brain CLI subcommands (init, list, delete, context, file-map) |
| TASK-534 | 2    | docs    | High     | Add 4 missing brain routes to buildHowToDoc                               |
| TASK-535 | 3    | qa      | High     | Write brain HTTP handler tests                                            |
| TASK-536 | 3    | qa      | High     | Write SQLite knowledge repository tests                                   |
| TASK-537 | 3    | qa      | Medium   | Expand brain_service_test.go (SearchKnowledge, InitProject, GetStatus)    |
| TASK-538 | 4    | qa      | Medium   | Add search_knowledge MCP + re-ingest E2E cases to brain_test.go           |
| TASK-539 | 4    | quality | Medium   | Fix SQLite BM25 explicit ORDER BY + UpdateKnowledge token_count recompute |

## Waves

### Wave 1 — Server-side foundation + runtime bug fixes (TASK-529..532)

All independent. Fix the broken public surface before tests depend on it.

### Wave 2 — CLI + docs (TASK-533..534)

CLI subcommands depend on Wave 1 HTTP routes existing. Docs update follows route additions.

### Wave 3 — Unit + handler test coverage (TASK-535..537)

All depend on Wave 1 having a correct implementation to test against.

### Wave 4 — E2E + quality (TASK-538..539)

E2E depends on routes existing (Wave 1). Quality fix is standalone but logically last.

## Acceptance Criteria

- `CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go build ./...` — clean
- `CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go test -race -count=1 ./...` — all green
- `CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go test -race -count=1 -tags=integration ./internal/e2e/...` — all green
- `howto_brief` parameter names match actual MCP tool schemas
- All 4 stub `brain_client.go` methods make real HTTP calls
- `app.go` `GetFocusedContext` calls `brainSvc.GetFocusedContext` not `GetContext`
- `With*Service` methods are unexported
