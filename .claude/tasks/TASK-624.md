---
id: TASK-624
planId: PLAN-075
title: 'Fix MCP tool list in docs/mcp-integration.md, route typos in api-reference.md, missing guides in docs site, and tighten docscheck'
role: docs
wave: 5
status: todo
createdAt: 2026-10-19
---

# TASK-624 — Fix MCP tool list in docs/mcp-integration.md, route typos in api-reference.md, missing guides in docs site, and tighten docscheck

1. **`docs/mcp-integration.md` & `docs/api-reference.md` & `docs/delegating-subscriptions.md` & `docs/getting-started.md`**:
   - Add the 4 missing MCP tools (`wait_for_task`, `get_onboarding_context`, `get_routing`, `get_usage`) to `docs/mcp-integration.md` (bringing the table to all 44 tools) and fix `"taskId"` → `"id"` in `get_task` examples (`docs/mcp-integration.md:189`, `docs/api-reference.md:924`).
   - Fix `POST /api/tasks/{id}/promote` → `POST /api/tasks/promote/{id}` in `docs/api-reference.md:723` and add `task.waiting`, `task.chunk`, `task.restart` to the SSE events table (`docs/api-reference.md:347-359`).
   - Fix `POST /api/providers/discovered/{id}/promote` → `POST /api/providers/promote/{id}` in `docs/delegating-subscriptions.md:13`.
   - Fix the `POST /api/tasks` response example and `.vsix` asset filename in `docs/getting-started.md`.
2. **`docs/src/data/guides.ts`, `README.md`, `docs/using-nexus.md`, `DownloadsView.vue`, `docs/vscode-model-bridge.md`, `docs/subscription-bridges.md`, `CHANGELOG.md`**:
   - Register `token-economy.md`, `switching-providers.md`, and `subscription-bridges.md` in `docs/src/data/guides.ts` and link them in `README.md`.
   - Update outdated claims (attached-daemon desktop behavior, `copilot` in `NEXUS_CLI_PROVIDERS`, `vscode.lm` `maxInputTokens`, and `subscription-bridges.md` F1/F4 status) and merge the duplicate `### Added` heading in `CHANGELOG.md`.
3. **`internal/docscheck/docscheck_test.go`**:
   - Assert every MCP tool in `docs/src/data/mcpTools.ts` is documented in `docs/mcp-integration.md` and `docs/api-reference.md`.
4. **`.claude/plans/`, `.claude/tasks/`, & `internal/adapters/outbound/sys_scanner/plan_scanner.go`**:
   - Parse markdown `**Status:** <status>` fallback in `plan_scanner.go` (or normalize the 97 legacy task files in `.claude/tasks/` to YAML frontmatter `status: done`/`closed`) and mark completed prior plans (`PLAN-003`..`PLAN-074`) as `status: completed` so `get_onboarding_context` reports `Active Plan: PLAN-075` and the exact open task count.

Done when: the failing test came first, 100% docs site coverage and `make test` / `make nice` pass.
