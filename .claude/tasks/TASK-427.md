---
id: TASK-427
plan: PLAN-057
status: todo
role: backend
wave: 6
---

# TASK-427: Detect and surface 429 rate-limit state per provider

When a provider returns HTTP 429 during `GenerateCode()` or `Chat()`, store a `RateLimited bool` and `RateLimitedAt time.Time` in the provider's health snapshot. Surface in `ProviderInfo` JSON so the frontend can show a "Rate Limited" badge.

Clear the flag after a configurable cooldown (e.g. 60s) or on next successful request.

**Files:** LLM adapters (llm_lmstudio, llm_ollama, llm_openaicompat, llm_anthropic), `internal/core/services/discovery.go`
**Depends on:** TASK-407
