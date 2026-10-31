---
id: TASK-631
planId: PLAN-076
title: 'Bound the task history the UIs fetch (limit/paging for /api/tasks/all)'
role: backend
status: todo
createdAt: 2026-10-31
---

# TASK-631 — Unbounded task history

Measured on a scratch daemon with 20 142 tasks (2 KB logs each): `GET /api/tasks/all` returns every task
(93 MB, 3.2 s) and Mission Control calls it every 30 s and on each event; `GET /api/tasks` returns the
whole queue (4 028 queued tasks: 18.5 MB, 0.63 s). With the 142 tasks of this installation it is
instant (a few ms), so nothing is broken today; it is a scaling limit to remove before the history
grows by two orders of magnitude.

Needs a product decision before building: what History shows beyond the newest N (paging, a date
range, or archiving old tasks). Sketch: an optional `limit`/`before` on `/api/tasks/all`, answered by
the repository with `LIMIT` (an optional interface next to `TaskRepository`, so the many test doubles do
not change), `logs` left out of list rows and fetched with `GET /api/tasks/{id}`; surface matrix,
API reference and changelog updated with it.
