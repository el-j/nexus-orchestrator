---
id: TASK-641
planId: PLAN-077
title: 'e2e for the 16 untouched REST routes'
role: testing
status: done
createdAt: 2026-10-31
---

# TASK-641 — e2e for the 16 untouched REST routes

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md).

/.well-known/nexus.json, /api/activities, /api/activities/timeline, /api/ai-sessions/discovered,
/api/brain/file-map, /api/config, /api/howto, /api/logs, /api/plans/discovered (+scan), /api/policy,
/api/providers/discovered (+scan), /api/providers/promote/{id}, /app, /ui. Add an e2e test that
fails when a registered route is not mentioned in any e2e source.

Done: `internal/e2e/rest_system_test.go` and `rest_coordination_test.go` run the real daemon through the dashboard and root redirect, the discovery beacon and how-to, settings and policy (including a refused value), logs and the activity views, plan, provider and agent discovery and scans, a session with its heartbeat, claimed task, task heartbeat and terminate, a stored provider configuration (edit, models, remove) and a knowledge entry (store, file map, delete). `TestEveryRESTRouteIsMentionedByAnE2ETest` in `internal/docscheck` fails when a registered route (read from `routes.go`) appears in no source of `internal/e2e` or `frontend/e2e`; 27 route entries were uncovered when it was written, none is now. The match is per route path with `{params}` as wildcards, so it proves a route is mentioned, not that every verb and status is covered.
