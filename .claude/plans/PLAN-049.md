# PLAN-049: Developer Experience — `make nice` + `make dev` fix

**Created:** 2026-03-27
**Goal:** Add a `make nice` target for one-command Go+Frontend formatting/linting/fixing, and fix `make dev` so all services start correctly and the frontend connects without lag or errors.

## Context (Rephrased Requirements)

1. **`make nice`** — A single Makefile target that auto-formats Go code (`gofmt -w`), runs `go vet`, auto-fixes lint issues (`golangci-lint run --fix`), and type-checks/lints the frontend (`vue-tsc --noEmit`). One command to make everything pretty, correct, and clean.

2. **`make dev`** — A working full-stack dev mode that starts the daemon (HTTP API on :63987 + MCP on :63988) with hot-reload, waits for the daemon to be healthy, then starts the Vite frontend dev server (:63989) with correct proxy configuration. The current `make dev` has bugs:
   - Vite proxies `/mcp` to port 63987 (HTTP API) instead of 63988 (MCP server) — wrong port
   - `/.well-known/nexus.json` not proxied at all
   - No readiness check — frontend starts before daemon compiles, causing connection errors until retry polling kicks in
   - The Makefile banner comment claims `/api+/mcp → :63987` which is wrong for MCP

## Root Causes Identified

| #   | Issue                                       | File                      | Fix                                             |
| --- | ------------------------------------------- | ------------------------- | ----------------------------------------------- |
| 1   | Vite `/mcp` proxy targets :63987 not :63988 | `frontend/vite.config.ts` | Change to `http://127.0.0.1:63988`              |
| 2   | `/.well-known/` not proxied                 | `frontend/vite.config.ts` | Add proxy rule                                  |
| 3   | No daemon readiness wait                    | `Makefile` `dev` target   | Add curl health-check loop before starting Vite |
| 4   | No `make nice` target                       | `Makefile`                | New composite target                            |
| 5   | Makefile banner text misleading             | `Makefile`                | Fix documentation                               |

## Tasks (Wave-based execution)

### Wave 1 — Independent fixes (parallel)

- **TASK-323**: Implement `make nice` Makefile target (Go: gofmt -w, go vet, golangci-lint fix; Frontend: vue-tsc --noEmit)
- **TASK-324**: Fix Vite proxy config (MCP port, add /.well-known/ proxy)
- **TASK-325**: Rewrite `make dev` target with health-check wait and correct banner

### Wave 2 — Validation

- **TASK-326**: Test `make nice` runs clean on codebase
- **TASK-327**: Test `make dev` starts all services and frontend connects

## Status: COMPLETE
