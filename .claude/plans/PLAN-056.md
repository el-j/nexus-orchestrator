# PLAN-056: UX Hardening, Playwright E2E, Token Auth & Universal AI Workflow Discovery

**Status:** active  
**Created:** 2026-03-28  
**Priority:** 1 — Release Quality Gate

## Problem Statement

Post PLAN-055 hardening, four independent problem classes remain that block shipping as a polished, security-conscious product:

### 🔴 Class A — Build / Lint Breakage

1. **`make nice` fails with 2 errcheck violations** in `cmd/nexus-mcp-stdio/main.go`.  
   `os.Stdout.Write(body)` return values are unchecked. The `.golangci.yml` errcheck exclusion list covers `fmt.Fprint*` but not direct `(*os.File).Write` calls. CI is broken until fixed.

### 🟡 Class B — Zero Authentication on Public Ports

2. **MCP server (`:63988`) has no token auth** — any local (or tunnelled) process can call all 31 MCP tools with zero credentials. Users who expose nexus via ngrok, Tailscale, or remote SSH are fully open.
3. **HTTP API (`:63987`) has no token auth** — same exposure. CORS is restricted to localhost but Origin headers are trivially forgeable.
4. **No UI to manage tokens** — even if env vars are set, the SettingsView only shows hardcoded values. Users cannot generate, rotate, or revoke tokens through the app. API keys for cloud providers (OpenAI, Anthropic, Copilot) can only be set via env var — not persisted to the database.
5. **Queue cap display is hardcoded to "50"** — SettingsView does not read the runtime value from the server; this silently misleads users running with `NEXUS_QUEUE_CAP` overrides.

### 🟠 Class C — No Browser-Level E2E Test Coverage

6. **Zero Playwright / browser E2E tests** — the full user journey (submit task, see queue, manage settings, token flow, backlog promotion, provider scan) has never been tested in a real browser. Vitest tests cover composables and component rendering but cannot verify navigation, form submission, clipboard, or multi-step flows.
7. **No CI gate for frontend regression** — only unit tests run in CI; a broken route or empty view would pass CI undetected.

### 🟡 Class D — Incomplete AI Workflow Discovery

8. **Windsurf (`.windsurfrules`) not scanned** — zero references anywhere in the codebase.
9. **`.github/copilot-instructions.md` not matched** — only root-level `copilot-instructions.md` is detected; the canonical Copilot location is `.github/`.
10. **`*.instructions.md` / `*.prompt.md` not depth-scanned** — Copilot scoped instruction files and prompt files ignored.
11. **Aider (`.aider.conf.yml`, `CONVENTIONS.md`) not scanned** — zero references.
12. **`.continue/config.json` / `.continue/config.yaml` not scanned as plan files** — Continue is detected as an agent but its config is not treated as a workflow artifact.
13. **`tasks.json`, `tasks.yaml`, `agent.yaml` generic task files not detected** — many AI tools emit these.
14. **Content heuristics for generic Markdown are basic** — a `README.md` and a `TASKS.md` are both classified the same way. No YAML frontmatter or checkbox structure detection.
15. **`DiscoveredPlansView` has no grouping or filtering** — all file kinds rendered in a flat list; no "AI Tool" grouping or type filter.
16. **No periodic background scan** — plan files are only rescanned on explicit API call; stale data until user refreshes.

### 🔴 Class E — SSE Transport Fragility (observed in live VS Code logs)

17. **`ReadTimeout: 15s` terminates idle SSE connections** — `StartMCPServer` sets `ReadTimeout: 15 * time.Second`. Any SSE session idle more than 15 seconds (no messages being sent) gets killed by the Go HTTP server before the client disconnects. The code comment says "no timeout" but the code contradicts it.
18. **No SSE ping/keepalive** — RFC-compliant SSE `:comment` heartbeats are never sent. Cloud proxies, VS Code's fetch layer, and network middleboxes close idle connections after ~60s, causing spurious `TypeError: terminated` errors.
19. **`POST /messages` on expired session returns ambiguous 400** — When the daemon restarts all in-memory SSE sessions are lost. The client retries using its cached `sessionId`, gets `400 "session not found"` with a plain-text body. VS Code MCP extension logs `Failed to parse message: ""` because it expects JSON. Client falls back to legacy SSE, which also fails until a fresh full reconnect.
20. **VS Code `mcp.json` uses `type:"sse"` by default** — The SSE transport is session-stateful and breaks across restarts. The Streamable HTTP transport (`type:"http"`, endpoint `/mcp`) is stateless and reconnect-safe. Client docs and the in-product `howto` tool never mention the `type:"http"` option.

| #   | Issue                                   | Root Cause                                                                             | Fix Strategy                                                        |
| --- | --------------------------------------- | -------------------------------------------------------------------------------------- | ------------------------------------------------------------------- |
| 1   | errcheck lint failure                   | `os.Stdout.Write` return ignored; not in exclusion list                                | Replace with `fmt.Fprint(os.Stdout, ...)` (already excluded)        |
| 2   | No MCP token auth                       | Server only validates Origin header; no Bearer token support                           | Add `NEXUS_MCP_TOKEN` env var + `tokenAuthMiddleware`               |
| 3   | No HTTP API token auth                  | Same origin-only model; no middleware                                                  | Add `NEXUS_API_TOKEN` env var + middleware                          |
| 4   | No token UI                             | SettingsView has no token section; tokens only via env var                             | Add RuntimeConfig table + Token management UI section               |
| 5   | Hardcoded queue cap                     | SettingsView hardcodes "50"; no config read endpoint                                   | Add `GET /api/config` endpoint + SettingsView reads it              |
| 6   | No Playwright tests                     | Never set up; only Vitest for unit/component                                           | Install @playwright/test, write E2E flows, CI integration           |
| 7   | Windsurf / Aider / Continue not scanned | `classifyFile()` pattern table incomplete                                              | Add cases for all missing tool formats                              |
| 8   | No recursive scan for instructions      | `scanDir()` is one-level for the root; does not walk `.github/` or subfolders          | Add depth-3 recursive walk for `*.instructions.md`, `*.prompt.md`   |
| 9   | Weak markdown heuristics                | All unknown `.md` files map to `PlanFileKindMarkdown` — no structural analysis         | Parse YAML frontmatter + checkbox/heading patterns                  |
| 10  | Flat DiscoveredPlansView                | Vue component renders flat array; no group-by                                          | Group by mapped AI tool name; add filter checkboxes                 |
| 11  | No background scan                      | `GetDiscoveredPlanFiles()` triggers fresh scan every call; no periodic refresh         | Timer goroutine in OrchestratorService (5-min interval)             |
| 17  | SSE ReadTimeout kills idle sessions     | `StartMCPServer` sets `ReadTimeout: 15s`; contradicts code comment saying "no timeout" | Set `ReadTimeout: 0`; wrap only `/mcp` POST handler with timeout    |
| 18  | No SSE ping keepalive                   | `handleSSE` never sends `: comment` heartbeats; proxies close idle connections         | Ticker sending `: ping\n\n` every 15s inside SSE loop               |
| 19  | 400 plain-text on missing session       | `handleSSEMessage` uses `http.Error()` which returns plain text; client can't parse it | Return `204 No Content` or JSON-RPC error with correct Content-Type |
| 20  | VS Code uses SSE not Streamable HTTP    | `mcp.json` template uses `type:"sse"` which is session-stateful                        | Update docs + howto text to prefer `type:"http"` → `/mcp`           |

## Tasks

| Task     | Wave | Role     | Description                                                                                      |
| -------- | ---- | -------- | ------------------------------------------------------------------------------------------------ |
| TASK-386 | 1    | backend  | Fix errcheck lint: replace `os.Stdout.Write` with `fmt.Fprint` in nexus-mcp-stdio                |
| TASK-387 | 2    | backend  | Add bearer token middleware to MCP server (`NEXUS_MCP_TOKEN` env var)                            |
| TASK-388 | 2    | backend  | Add bearer token middleware to HTTP API server (`NEXUS_API_TOKEN` env var)                       |
| TASK-389 | 2    | backend  | RuntimeConfig: SQLite `settings` table + port + service + `GET/PUT /api/config` endpoint         |
| TASK-390 | 3    | frontend | SettingsView: Token Management section (generate/rotate/copy, enable toggle)                     |
| TASK-391 | 3    | frontend | SettingsView: read live queue cap + server config from `GET /api/config` instead of hardcode     |
| TASK-392 | 4    | testing  | Playwright: install, `playwright.config.ts`, test server fixture, npm scripts, Makefile target   |
| TASK-393 | 4    | testing  | Playwright E2E: Dashboard + task lifecycle (submit → queue → cancel)                             |
| TASK-394 | 4    | testing  | Playwright E2E: Settings + token management (add provider, generate token, toggle)               |
| TASK-395 | 4    | testing  | Playwright E2E: Backlog flow (create draft → promote → NO_PROVIDER warning visible)              |
| TASK-396 | 4    | testing  | Playwright E2E: Providers view + discovery trigger + plan file scan                              |
| TASK-397 | 5    | ci       | Playwright CI: Makefile `test-e2e` target + GitHub Actions job with HTML report artifact         |
| TASK-398 | 6    | backend  | Discovery: add Windsurf, Aider, Continue-config, Copilot .github/, tasks.json/yaml kinds         |
| TASK-399 | 6    | backend  | Discovery: depth-3 recursive scan for `*.instructions.md`, `*.prompt.md`, `.github/` subdir      |
| TASK-400 | 6    | backend  | Discovery: content heuristics — YAML frontmatter + checkbox/heading pattern classifier           |
| TASK-401 | 7    | frontend | DiscoveredPlansView: group by AI tool, filter checkboxes, tool badges, active indicator          |
| TASK-402 | 7    | backend  | MCP `get_discovered_plans`: enrich response with `detectedTools[]`, `totalActive`, `scanRoots[]` |
| TASK-403 | 7    | backend  | Discovery: periodic background scan goroutine (5-min timer, stopCh pattern)                      |
| TASK-404 | 8    | backend  | Fix SSE 400 on restart: remove ReadTimeout, add 15s ping keepalive, fix missing-session response |
| TASK-405 | 8    | testing  | SSE tests: ping keepalive, missing-session parseable response, reconnect after session loss      |
| TASK-406 | 8    | docs     | Update MCP client setup docs + howto tool text to recommend Streamable HTTP for VS Code          |

## Wave Plan

```
Wave 1 (TASK-386)                  — Lint fix. Unblocks `make nice` and CI. ✅ DONE
Wave 2 (TASK-387, 388, 389)        — Backend token auth + config storage. Foundation for UX.
Wave 3 (TASK-390, 391)             — Settings UX. Depends on Wave 2 backend.
Wave 4 (TASK-392..396)             — Playwright setup + all E2E flows. Needs working UI from Wave 3.
Wave 5 (TASK-397)                  — CI integration. Needs Wave 4 tests passing.
Wave 6 (TASK-398, 399, 400)        — Discovery backend extension. Independent of Waves 2–5.
Wave 7 (TASK-401, 402, 403)        — Discovery UI + MCP enrichment + background scan. Needs Wave 6.
Wave 8 (TASK-404, 405, 406)        — SSE transport hardening + reconnect fix + client docs update.
```

Waves 1–5 can proceed in parallel with Waves 6–7.
Wave 8 is independent and can run in parallel with any other wave.

## Success Criteria

- [ ] `make nice` exits 0 with zero lint issues
- [ ] MCP server rejects unauthenticated requests when `NEXUS_MCP_TOKEN` is set (401)
- [ ] HTTP API rejects unauthenticated requests when `NEXUS_API_TOKEN` is set (401)
- [ ] User can generate, copy, and rotate access tokens from SettingsView without touching env vars
- [ ] `SettingsView` shows live queue cap read from server
- [ ] Playwright suite covers ≥5 user flows and runs headless in CI (Chromium)
- [ ] `.windsurfrules`, `.aider.conf.yml`, `.continue/config.json`, `*.instructions.md`, `.github/copilot-instructions.md` all appear in DiscoveredPlanFiles scan results
- [ ] DiscoveredPlansView groups files by AI tool with filter panel
- [ ] Background plan file scanner updates every 5 minutes without manual API call
- [ ] SSE connections survive daemon restart without 400 errors (client reconnects cleanly)
- [ ] VS Code `mcp.json` using `type:"http"` connects without 400/terminated log noise
- [ ] `howto_brief` MCP tool explicitly recommends Streamable HTTP for VS Code

## Key Files

| Layer      | File                                                          | Change                                      |
| ---------- | ------------------------------------------------------------- | ------------------------------------------- |
| MCP server | `internal/adapters/inbound/mcp/server.go`                     | Add tokenAuthMiddleware                     |
| HTTP API   | `internal/adapters/inbound/httpapi/server.go`                 | Add tokenAuthMiddleware                     |
| Ports      | `internal/core/ports/ports.go`                                | Add ConfigRepository interface              |
| Services   | `internal/core/services/config_service.go` (NEW)              | RuntimeConfig CRUD service                  |
| SQLite     | `internal/adapters/outbound/repo_sqlite/config_repo.go` (NEW) | settings table adapter                      |
| Frontend   | `frontend/src/views/SettingsView.vue`                         | Token management + live config              |
| Discovery  | `internal/adapters/outbound/sys_scanner/plan_scanner.go`      | New file kinds + recursive scan             |
| Domain     | `internal/core/domain/discovered_agent.go`                    | New PlanFileKind constants                  |
| Frontend   | `frontend/src/views/DiscoveredPlansView.vue`                  | Grouped view with filter                    |
| Tests      | `frontend/e2e/` (NEW)                                         | Playwright test suite                       |
| CI         | `.github/workflows/e2e.yml` (NEW)                             | Playwright CI job                           |
| MCP SSE    | `internal/adapters/inbound/mcp/sse.go`                        | Add ping ticker, fix 400 on missing session |
| MCP server | `internal/adapters/inbound/mcp/server.go`                     | Remove ReadTimeout from SSE path            |
| Docs       | `docs/mcp-integration.md`                                     | VS Code Streamable HTTP setup guide         |
| MCP tools  | `internal/adapters/inbound/mcp/tools.go`                      | howto + howto_brief VS Code setup           |
