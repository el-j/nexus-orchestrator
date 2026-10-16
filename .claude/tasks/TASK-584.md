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

Note: LM Studio's own adapter does not report usage yet (local model, no quota); the other adapters do.
