# PLAN-025: Distribution Hardening & VS Code Extension v0.2.0

**Status:** active
**Created:** 2026-03-12
**Goal:** Fix the broken `make build-all` cross-compilation (zig 0.15.2 musl regression); auto-register the nexus-orchest MCP server via `contributes.mcpServers` in the VS Code extension (eliminating the need for manual `.vscode/mcp.json`); rebuild and package the extension at v0.2.0 with all PLAN-022 features (SessionMonitor, AI session registration) baked in.

---

## Problem Summary

| # | Issue | Impact |
|---|-------|--------|
| P1 | `make build-all` fails — zig 0.15.2 musl linker missing `__errno_location` and glibc symbols | CI releases cannot cross-compile |
| P2 | VS Code extension `dist/extension.js` is stale (696 lines, pre-PLAN-022) — SessionMonitor not included | Extension silently broken for users |
| P3 | Extension has no `contributes.mcpServers` — users must manually create `.vscode/mcp.json` | Poor DX; MCP error persists until restart |
| P4 | `orchestrator.json` had `nextTaskId: null`, `nextPlanId: null`, `PLAN-002: todo` | Tooling breaks on next plan creation |

---

## Architecture

No new ports or domain types. All changes are at the distribution / adapter layer:

```
Makefile           (build layer)
↓
vscode-extension/package.json   (contributes.mcpServers auto-config)
↓
vscode-extension/dist/          (rebuilt bundle with SessionMonitor)
↓
vscode-extension/nexus-orchestrator-0.2.0.vsix   (redistributable)
```

---

## Execution Plan — 3 Waves

### Wave 1 — Independent (run in parallel)
| Task | Role | Description |
|------|------|-------------|
| TASK-184 | devops | Fix `make build-all` — zig 0.15.x musl `-tags netgo,osusergo -extldflags='-static'` |
| TASK-185 | architecture | VS Code extension `package.json` — `contributes.mcpServers` + `nexus.mcpPort` config |

### Wave 2 — After Wave 1 (parallel)
| Task | Role | Description |
|------|------|-------------|
| TASK-186 | devops | Rebuild extension dist, bump to v0.2.0, repackage VSIX |
| TASK-187 | planning | Fix orchestrator.json metadata counters |

### Wave 3 — After Wave 2
| Task | Role | Description |
|------|------|-------------|
| TASK-188 | qa | Full QA: build-all, extension packaging, test suite green |

---

## Acceptance Criteria
- [ ] `make build-linux-amd64` exits 0
- [ ] `make build-linux-arm64` exits 0
- [ ] `make build-darwin-amd64` exits 0 (or documented as unsupported cross-arch on darwin host)
- [ ] `make build-windows-amd64` exits 0
- [ ] VS Code extension installs and auto-registers `nexus-orchest` MCP server on activation
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `nexus-orchestrator-0.2.0.vsix` exists and is < 2MB
