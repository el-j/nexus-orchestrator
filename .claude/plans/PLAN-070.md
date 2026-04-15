---
id: PLAN-070
title: 'Comprehensive Gap-Fill — Docs, MCP Tools, Frontend, Tests, Backend Quality'
goal: 'Close all gaps found in the 2026-04-14 five-agent swarm audit: stale docs, 4 missing MCP brain tools, VS Code extension brain gaps, stale Wails bindings, dead UI code, zero-coverage critical paths, and silent error swallowing.'
status: todo
createdAt: 2026-04-14T02:00:00Z
---

# PLAN-070 — Comprehensive Gap-Fill (Audit Findings)

## Background

A five-agent parallel swarm audit completed on 2026-04-14 after PLAN-069 delivered the Brain
layer gap-fill. The audit covered docs, Go backend, frontend, VS Code extension, and test coverage.
Combined critical/high severity findings: **26 confirmed gaps**. No code was written during the
audit; this plan is the full remediation backlog.

## Confirmed Gaps by Domain

### Documentation (Critical)

- `ApiReferenceView.vue`: `mcpTools` lists 6 of 36 tools; `endpoints` missing all 9 brain routes
- `McpIntegrationView.vue`: same 6-tool list; no brain usage examples; no VS Code setup section
- `api-reference.md`: 5 of 9 brain endpoints absent; MCP table stale at 11 tools
- `mcp-integration.md`: zero brain tool usage examples
- `getting-started.md`: no brain/project-knowledge section at all
- `howto` MCP full guide: brain tools absent from its enumeration

### MCP Tooling (High)

- 4 brain HTTP routes added in PLAN-069 have no MCP counterparts:
  `init_project`, `list_knowledge`, `delete_knowledge`, `get_file_map`

### VS Code Extension (High)

- `nexusClient.ts` missing 4 brain HTTP methods (init, list, delete, file-map)
- Only `nexus.brain.ingest` registered; no status, context, search, init commands
- `searchKnowledge()` return type wrong — unwraps `{"results":[...]}` incorrectly
- Brain actions absent from status-bar quick-pick

### Frontend / Wails Bindings (Critical/High)

- `App.d.ts` never regenerated — brain bindings absent; still exposes `WithActivityService`
- `models.ts` dates typed as `any` (stale codegen)
- `wails.ts`: `getProjectContext`, `getFocusedContext`, `searchKnowledge` are dead code — called nowhere
- No `useBrain` composable; no `/brain` route
- `KnowledgeResult` type defined but mismatches actual `ContextSection[]` return
- `ProjectBrainCard`: only ingests single file; no user-visible feedback after ingest
- Wails App doesn't bind: `InitProject`, `ListKnowledge`, `DeleteKnowledge`, `GetFileMap`

### Backend Quality (High/Medium)

- `processNextTask`: silent error swallowing — failed tasks stay PROCESSING forever
- `IngestFromFile`: per-section errors silently dropped; caller gets count=0, err=nil
- `GetDiscoveredPlanFiles`: returns `nil, nil` when subsystem not wired (undiscoverable)
- FTS5 `SearchFTS`: raw query string passed to MATCH without validation
- Tray adapter: fully stubbed — `Start()` no-op, `Enabled()` always false

### Test Coverage (Critical)

- `brain_client.go`: 0% — no `brain_client_test.go` exists
- MCP `brain_tools.go`: 0% — no MCP brain tool test coverage
- `brain_service.go`: `GetFileMap`, `ListKnowledge`, `DeleteKnowledge` at 0%
- E2E `brain_test.go`: FTS search step returns 0 results (silent test failure)
- httpapi provider handlers: 8 functions at 0%
- httpapi activity handlers: 0%
- `httpapi_client/client.go`: 18 of 30 methods at 0%

## Task Map

| Task     | Wave | Domain   | Priority | Description                                                                             |
| -------- | ---- | -------- | -------- | --------------------------------------------------------------------------------------- |
| TASK-540 | 1    | docs-vue | Critical | Rebuild mcpTools + endpoints arrays in ApiReferenceView + McpIntegrationView            |
| TASK-541 | 1    | docs-md  | High     | Update api-reference.md, mcp-integration.md, getting-started.md with brain content      |
| TASK-542 | 1    | mcp      | High     | Add 4 missing brain MCP tools + fix howto full guide enumeration                        |
| TASK-543 | 1    | vscode   | High     | Add missing nexusClient brain methods + commands + fix searchKnowledge return type      |
| TASK-544 | 2    | frontend | Critical | Regenerate App.d.ts + models.ts; add 4 Wails App brain method bindings                  |
| TASK-545 | 2    | frontend | High     | Create useBrain composable + /brain route + wire dead context/search UI code            |
| TASK-546 | 2    | frontend | Medium   | ProjectBrainCard: multi-file ingest + user-visible feedback + init button               |
| TASK-547 | 2    | backend  | High     | Fix processNextTask silent errors + IngestFromFile error aggregation + FTS sanitization |
| TASK-548 | 3    | qa       | Critical | Write brain_client_test.go (all 13 methods)                                             |
| TASK-549 | 3    | qa       | Critical | Write MCP brain_tools unit tests + fix E2E FTS search failure                           |
| TASK-550 | 3    | qa       | High     | Brain service missing method coverage (GetFileMap, ListKnowledge, DeleteKnowledge)      |
| TASK-551 | 3    | qa       | High     | httpapi provider handler tests + activity handler tests                                 |
| TASK-552 | 4    | qa       | Medium   | httpapi_client/client.go coverage + session lifecycle (terminate, heartbeat, purge)     |
| TASK-553 | 4    | backend  | Medium   | GetDiscoveredPlanFiles nil/nil fix + Tray adapter honest stub with feature flag         |

## Waves

### Wave 1 — Independent surface fixes (TASK-540..543)

Docs and MCP/VS Code are all independent. Run in parallel.

### Wave 2 — Frontend + backend quality (TASK-544..547)

Frontend bindings depend on Wave 1 docs being accurate. Backend quality fixes are independent
but logically grouped here.

### Wave 3 — Test coverage: brain + critical paths (TASK-548..551)

Tests depend on Wave 1 (correct implementation) and Wave 2 (bindings correct).

### Wave 4 — Remaining test coverage + backend quality (TASK-552..553)

Independent of Wave 3; lower urgency.

## Acceptance Criteria

- `CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go build ./...` — clean
- `CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go test -race -count=1 ./...` — all green
- `CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go test -tags=integration -race -count=1 ./internal/e2e/...` — all green
- `cd docs && npm run build` — clean
- `cd frontend && vue-tsc --noEmit` — clean
- `cd vscode-extension && npm run compile` — clean
- `ApiReferenceView.vue` lists all 36 MCP tools and all 9 brain HTTP endpoints
- E2E FTS search step asserts ≥1 result and passes
- `brain_client.go` coverage ≥ 80%
- `brain_tools.go` coverage ≥ 80%
