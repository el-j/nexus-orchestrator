---
id: TASK-588
planId: PLAN-074
title: 'e2e: failover, cooldown and wait through the real daemon'
role: tester
wave: 2
status: done
createdAt: 2026-10-16
dependsOn: [TASK-587]
---

# TASK-588 — e2e: failover, cooldown and wait through the real daemon

Fake provider answers 429 with Retry-After: the task fails over, then waits, then completes. Black-box test in internal/e2e.

Done when: the failing test came first, `make test` and `make nice` pass, docs and surface matrix updated where an interface changed.
