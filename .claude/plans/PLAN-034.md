---
id: PLAN-034
goal: "Make Nexus routing explicit in the VS Code extension and surface route visibility through logs, status, and queue UI"
status: completed
createdAt: 2026-03-13T01:30:00.000Z
completedAt: 2026-03-13T01:50:00.000Z
---

## Problem

The extension could submit tasks to the Nexus queue, but the primary UX did not make that route explicit. Users could also see Copilot sessions and MCP availability without a clear distinction between direct Copilot activity, MCP-mediated work, and tasks explicitly queued through Nexus.

## Fix Strategy

### Wave 1 — Explicit queue workflow (TASK-236)
- Add a dedicated `Nexus: Send Current Context To Queue` command.
- Let users choose reviewed context files before queueing.
- Keep `Nexus: Compose Manual Task` as the fallback path.

### Wave 2 — Route visibility and logging (TASK-237)
- Introduce a shared Nexus activity log channel.
- Surface route-aware state in the status bar and task queue tree.
- Add a `Nexus: Show Activity Log` command and visible entry points.

### Wave 3 — Verification (TASK-238)
- Build the Go and VS Code extension surfaces.
- Verify the active backlog/history fix still compiles and the extension bundle builds.

## Tasks
- TASK-236: VS Code extension explicit queue workflow
- TASK-237: VS Code extension route visibility and activity log
- TASK-238: Verification for PLAN-033 and PLAN-034