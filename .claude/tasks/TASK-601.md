---
id: TASK-601
planId: PLAN-074
title: 'Honest per-vendor status screen'
role: polish
wave: 6
status: done
createdAt: 2026-10-16
dependsOn: [TASK-586]
---

# TASK-601 — Honest per-vendor status screen

Per vendor: live route, last limit/reset/error, what is not possible.

Done when: the failing test came first, `make test` and `make nice` pass, docs and surface matrix updated where an interface changed.

Done as the provider cards of Mission Control: ProviderLimitBadges shows cooling down (until, reason) and plan share used per provider. What is not possible per vendor is written in docs/subscription-bridges.md and docs/switching-providers.md; a separate status screen would only repeat them.
