---
id: TASK-593
planId: PLAN-074
title: 'Budgets per provider and day'
role: coder
wave: 4
status: done
createdAt: 2026-10-16
dependsOn: [TASK-585, TASK-582]
---

# TASK-593 — Budgets per provider and day

Policy budget; reaching it acts like a cooldown.

Done when: the failing test came first, `make test` and `make nice` pass, docs and surface matrix updated where an interface changed.

Done: `budgets` in the policy, enforced from the ledger as a cooldown until local midnight (budget.go), shown by the routing preview.
