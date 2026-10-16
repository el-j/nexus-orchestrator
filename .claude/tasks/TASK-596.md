---
id: TASK-596
planId: PLAN-074
title: 'Prompt caching for the stable prefix'
role: coder
wave: 5
status: dropped
createdAt: 2026-10-16
dependsOn: [TASK-584]
---

# TASK-596 — Prompt caching for the stable prefix

Anthropic cache control on rules and briefing; keep the prefix byte-stable.

Done when: the failing test came first, `make test` and `make nice` pass, docs and surface matrix updated where an interface changed.

Status: dropped. Anthropic prompt caching needs the rules and briefing in the system block with `cache_control`; Nexus puts them in the user message, so it would be a restructure of every prompt. It mostly helps API-key users, while the goal of PLAN-074 is subscriptions, which cache on their own (the `claude` client reported cached tokens in every call). Not worth the risk now.
