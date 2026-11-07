---
id: TASK-632
planId: PLAN-077
title: 'Remove dead frontend code'
role: frontend
status: done
createdAt: 2026-10-31
---

# TASK-632 — Remove dead frontend code

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md).

DiscoveryView, AISessionsView, TaskQueue, AISessionCard, ProviderStatus are imported nowhere (router
does not route the two views). Delete them and their specs, keep coverage at 100 %, update docs and
the surface matrix if they are named. Decide per wails.ts export (getTask, claimTask,
updateTaskStatus, registerAISession, heartbeatAISession, authHeaders) whether it stays as REST/MCP
parity (document it) or goes.

Done: removed `DiscoveryView`, `AISessionsView`, `TaskQueue`, `AISessionCard`, `ProviderStatus` with their specs and the stale `ProviderStatus` stubs in the Providers specs. The six unused `wails.ts` exports (`getTask`, `claimTask`, `updateTaskStatus`, `registerAISession`, `heartbeatAISession`, `authHeaders`) stay: they are the desktop/REST client side of the surface matrix (REST and MCP offer the same calls to other agents), and `wails.spec.ts` tests them. Frontend: 795 tests, 100 % coverage.
