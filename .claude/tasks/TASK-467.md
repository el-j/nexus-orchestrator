---
id: TASK-467
plan: PLAN-061
status: done
wave: 4
priority: 4
---

# TASK-467: Replace map[string]interface{} with typed structs in LLM adapters

**Problem:** `llm_lmstudio`, `llm_ollama`, and `llm_openaicompat` all build JSON request bodies using `map[string]interface{}`. This bypasses compile-time type checking, makes the API contract invisible, and creates maintenance risk if field names drift. The Anthropic adapter (`llm_anthropic`) already uses typed request/response structs and should be the reference.

**Fix:** Define typed request and response structs for each adapter's API payload. Replace all `map[string]interface{}` usages with the typed structs.

**Files:**

- `internal/adapters/outbound/llm_lmstudio/adapter.go` (+ new `types.go` or inline structs)
- `internal/adapters/outbound/llm_ollama/adapter.go` (+ new `types.go` or inline structs)
- `internal/adapters/outbound/llm_openaicompat/adapter.go` (+ new `types.go` or inline structs)
- `internal/adapters/outbound/llm_anthropic/adapter.go` (reference — do not modify)

## Checklist

- [ ] Audit `llm_anthropic` typed structs as the reference pattern to follow
- [ ] For `llm_lmstudio`: define `chatRequest`, `chatMessage`, `chatResponse`, `chatChoice` structs; replace all `map[string]interface{}` in `GenerateCode` and `Chat` with these structs
- [ ] For `llm_ollama`: define `generateRequest`, `generateResponse`, `chatRequest`, `chatResponse` structs per the Ollama REST API; replace all map usages
- [ ] For `llm_openaicompat`: define `openAIRequest`, `openAIMessage`, `openAIResponse`, `openAIChoice` structs; replace all map usages
- [ ] Ensure all struct fields carry correct `json:"..."` tags matching the provider API spec
- [ ] Confirm no regressions in existing adapter tests; run `go test ./internal/adapters/outbound/llm_lmstudio/... ./internal/adapters/outbound/llm_ollama/... ./internal/adapters/outbound/llm_openaicompat/...`
- [ ] Run `go vet ./...` clean after changes

## Acceptance Criteria

- Zero `map[string]interface{}` usages for JSON request/response bodies in the three adapter packages
- `llm_anthropic` pattern is consistently applied across all four LLM adapters
- Existing tests pass; `go vet ./...` clean
