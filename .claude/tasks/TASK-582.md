---
id: TASK-582
planId: PLAN-074
title: 'Provider cooldown store; chain skips cooled-down providers'
role: coder
wave: 1
status: done
createdAt: 2026-10-16
dependsOn: [TASK-581]
---

# TASK-582 — Provider cooldown store; chain skips cooled-down providers

Per provider/model cooldown-until from typed errors (A2/A3). `ProviderInfo.RateLimited` becomes true with `rateLimitedUntil`; remove the dead `discovery` assignments. The chain skips candidates in cooldown and logs why. Cleared on success or expiry.

Done when: the failing test came first, `make test` and `make nice` pass, docs and surface matrix updated where an interface changed.
