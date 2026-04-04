---
id: TASK-466
plan: PLAN-061
status: done
wave: 4
priority: 4
---

# TASK-466: Implement ContextLimit() for Anthropic and OpenAI-compat adapters

**Problem:** Both `internal/adapters/outbound/llm_anthropic/adapter.go` and `internal/adapters/outbound/llm_openaicompat/adapter.go` implement `ContextLimit()` returning hardcoded `0`. The orchestrator uses `ContextLimit()` to route tasks to the most capable available provider. A value of `0` is indistinguishable from "no limit known", causing model-context-aware routing to silently assign zero capacity to these adapters and potentially routing large-context tasks to less capable providers.

**Fix:** Implement a model-name-keyed lookup table of known context limits for Anthropic and common OpenAI-compatible models. Return the appropriate value for the configured model, falling back to a safe default (e.g. 8192) if the model name is unrecognised.

**Files:**

- `internal/adapters/outbound/llm_anthropic/adapter.go`
- `internal/adapters/outbound/llm_openaicompat/adapter.go`
- `internal/adapters/outbound/llm_anthropic/context_limits.go` (new, or inline in adapter)
- `internal/adapters/outbound/llm_openaicompat/context_limits.go` (new, or inline)

## Checklist

- [ ] Define a `var knownContextLimits = map[string]int{...}` in each adapter package covering at minimum: `claude-3-5-sonnet` (200k), `claude-3-opus` (200k), `claude-3-haiku` (200k), `claude-2.1` (200k), `gpt-4o` (128k), `gpt-4-turbo` (128k), `gpt-4` (8k), `gpt-3.5-turbo` (16k)
- [ ] For the Anthropic adapter: use `a.model` (the configured model name) to look up in `knownContextLimits`; return the value if found, else `200000` (Anthropic's default for modern models)
- [ ] For the OpenAI-compat adapter: use `a.model` to look up; return the value if found, else `8192` as a conservative default
- [ ] Handle inexact matches for versioned model names (e.g. `claude-3-5-sonnet-20241022`) using a `strings.HasPrefix` or prefix scan
- [ ] Add a short unit test per adapter: known model returns correct limit; unknown model returns nonzero default
- [ ] Run `go test ./internal/adapters/outbound/llm_anthropic/... ./internal/adapters/outbound/llm_openaicompat/...`

## Acceptance Criteria

- `ContextLimit()` returns a nonzero value for all configured Anthropic and common OpenAI-compat models
- Unknown model names return a nonzero safe default (not `0`)
- Routing logic can now distinguish actual capacity from "unknown"
- Unit tests pass
