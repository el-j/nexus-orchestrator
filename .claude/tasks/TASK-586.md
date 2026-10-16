---
id: TASK-586
planId: PLAN-074
title: 'Quota threshold routing from the client's own reading'
role: coder
wave: 2
status: todo
createdAt: 2026-10-16
dependsOn: [TASK-582, TASK-583]
---

# TASK-586 — Quota threshold routing from the client's own reading

Store `utilization` and `resetsAt` per provider from the Claude driver; policy `quotaThreshold` (default 0.9) routes away before the limit; shown in `GET /api/providers`.

Done when: the failing test came first, `make test` and `make nice` pass, docs and surface matrix updated where an interface changed.
