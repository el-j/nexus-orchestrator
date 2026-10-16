---
id: TASK-597
planId: PLAN-074
title: 'Verify before asking a frontier model again'
role: coder
wave: 5
status: todo
createdAt: 2026-10-16
dependsOn: [TASK-594]
---

# TASK-597 — Verify before asking a frontier model again

Run the verification command first; send only the failing output on retry.

Done when: the failing test came first, `make test` and `make nice` pass, docs and surface matrix updated where an interface changed.
