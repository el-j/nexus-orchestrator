---
id: TASK-581
planId: PLAN-074
title: 'Typed provider errors: rate limit, quota exhausted, overloaded'
role: coder
wave: 1
status: todo
createdAt: 2026-10-16
---

# TASK-581 — Typed provider errors: rate limit, quota exhausted, overloaded

Add `domain.ProviderError{Kind, RetryAfter, ResetAt}` (kinds: RateLimited, QuotaExhausted, Overloaded, Auth, Network) with `errors.As` helpers. Map HTTP status and headers (`Retry-After`, vendor reset headers) in llm_anthropic, llm_gemini, llm_openaicompat (sync and stream). Tests first per adapter.

Done when: the failing test came first, `make test` and `make nice` pass, docs and surface matrix updated where an interface changed.
