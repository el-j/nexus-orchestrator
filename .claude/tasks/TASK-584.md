---
id: TASK-584
planId: PLAN-074
title: 'Usage on the LLM port'
role: coder
wave: 1
status: done
createdAt: 2026-10-16
dependsOn: [TASK-583]
---

# TASK-584 — Usage on the LLM port

`ports.Usage{InputTokens, OutputTokens, CachedTokens, CostUSD}`; adapters return it (Anthropic, Gemini, OpenAI-compatible, Claude driver; streaming included). Core keeps it with the task result.

Done when: the failing test came first, `make test` and `make nice` pass, docs and surface matrix updated where an interface changed.

Note: LM Studio's adapter now reports usage too (found while using the daemon with a real local model: the ledger showed only Claude).
