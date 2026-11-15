---
id: TASK-632
planId: PLAN-077
title: 'Remove dead frontend code'
role: frontend
status: todo
createdAt: 2026-10-31
---

# TASK-632 — Remove dead frontend code

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md).

DiscoveryView, AISessionsView, TaskQueue, AISessionCard, ProviderStatus are imported nowhere (router
does not route the two views). Delete them and their specs, keep coverage at 100 %, update docs and
the surface matrix if they are named. Decide per wails.ts export (getTask, claimTask,
updateTaskStatus, registerAISession, heartbeatAISession, authHeaders) whether it stays as REST/MCP
parity (document it) or goes.
