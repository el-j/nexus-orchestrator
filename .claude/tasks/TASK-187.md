---
id: TASK-187
title: Update CHANGELOG for v0.9.1 distribution fixes
role: planning
planId: PLAN-025
status: todo
dependencies: [TASK-184, TASK-185, TASK-186]
createdAt: 2026-03-12T10:00:00.000Z
---

## Context
PLAN-025 introduces distribution hardening and VS Code extension improvements that deserve a CHANGELOG entry under a new `[Unreleased]` section, and eventually a v0.9.1 patch release. This task also finalises the orchestrator.json with correct nextTaskId and nextPlanId.

## Files to Read
- `CHANGELOG.md` — current top entries
- `.claude/orchestrator.json` — plan metadata to update

## Implementation Steps

1. **Update `CHANGELOG.md`**: Add a new `[Unreleased]` section above the existing `[0.9.0]` entry:

   ```markdown
   ## [Unreleased]

   ### Fixed

   - `make build-all` cross-compilation now works with zig 0.15.x: added `-tags netgo,osusergo` and `-extldflags='-static'` to Linux targets, eliminating musl `__errno_location` / `pthread_*` linker errors
   - VS Code extension now auto-registers the `nexus-orchest` MCP server via `contributes.mcpServers` (VS Code 1.99+) — no manual `.vscode/mcp.json` required
   - VS Code extension rebuilt at v0.2.0: bundles `SessionMonitor` (PLAN-022) and AI session auto-registration

   ### Changed

   - VS Code extension minimum engine version bumped to `^1.99.0` to support `contributes.mcpServers`
   ```

2. **Update `orchestrator.json`**: After TASK-188 is recorded, the state should have:
   - `nextTaskId: 189`
   - `nextPlanId: 26`
   - `activePlanId: null`
   - All TASK-184 through TASK-188 marked `done`
   - `PLAN-025.status: "completed"`, `PLAN-025.completedAt: <now>`

3. **Verify TASK-183 file status**: The `.claude/tasks/TASK-183.md` still says `status: todo` in the frontmatter (orchestrator.json says `done`). Update the file frontmatter to `status: done`.

## Acceptance Criteria
- [ ] `CHANGELOG.md` has `[Unreleased]` section with PLAN-025 entries
- [ ] `.claude/tasks/TASK-183.md` frontmatter `status: done`
- [ ] `orchestrator.json` has `nextTaskId: 189`, `nextPlanId: 26`, `activePlanId: null`

## Anti-patterns to Avoid
- NEVER skip updating `orchestrator.json` when a plan completes — the counters break the next session
- NEVER use a version number in CHANGELOG before it is tagged — use `[Unreleased]` until `git tag v0.9.1`
