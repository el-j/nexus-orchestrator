---
id: TASK-641
planId: PLAN-077
title: 'e2e for the 16 untouched REST routes'
role: testing
status: todo
createdAt: 2026-10-31
---

# TASK-641 — e2e for the 16 untouched REST routes

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md).

/.well-known/nexus.json, /api/activities, /api/activities/timeline, /api/ai-sessions/discovered,
/api/brain/file-map, /api/config, /api/howto, /api/logs, /api/plans/discovered (+scan), /api/policy,
/api/providers/discovered (+scan), /api/providers/promote/{id}, /app, /ui. Add an e2e test that
fails when a registered route is not mentioned in any e2e source.
