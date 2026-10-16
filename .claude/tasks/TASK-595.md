---
id: TASK-595
planId: PLAN-074
title: 'History capped by tokens, old turns summarized'
role: coder
wave: 5
status: done
createdAt: 2026-10-16
dependsOn: [TASK-594]
---

# TASK-595 — History capped by tokens, old turns summarized

Replace the message-count cap; summarize with a cheap model.

Done when: the failing test came first, `make test` and `make nice` pass, docs and surface matrix updated where an interface changed.

Scope note: a token cap on the history is done (`historyBudget`); summarising old turns with a cheap model is not, because it adds a model call and loses detail; the cap is simple and predictable. Revisit with measurements from `nexus-cli usage`.
