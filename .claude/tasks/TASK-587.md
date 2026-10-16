---
id: TASK-587
planId: PLAN-074
title: 'WAITING state: hold when every candidate is cooled down'
role: coder
wave: 2
status: done
createdAt: 2026-10-16
dependsOn: [TASK-582, TASK-586]
---

# TASK-587 — WAITING state: hold when every candidate is cooled down

New task status `WAITING` with `waitUntil`; engine holds instead of failing, resumes on its own at the reset; SSE, REST, MCP, CLI, UI and extension show it; surface matrix, API reference, changelog.

Done when: the failing test came first, `make test` and `make nice` pass, docs and surface matrix updated where an interface changed.
