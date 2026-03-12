---
id: TASK-218
title: Add LLM adapter unit tests — all 4 providers
role: qa
planId: PLAN-030
status: todo
dependencies: []
createdAt: 2025-07-25T00:00:00.000Z
---

## Context
All 4 LLM provider adapters (`llm_anthropic`, `llm_lmstudio`, `llm_ollama`, `llm_openaicompat`) have ZERO test coverage. These implement the core `ports.LLMClient` interface and are critical path for code generation. HTTP request shaping, response parsing, error handling, and timeout behavior are all untested.

## Files to Read
- `internal/core/ports/ports.go` — `LLMClient` interface definition
- `internal/adapters/outbound/llm_lmstudio/adapter.go` — reference implementation
- `internal/adapters/outbound/llm_ollama/adapter.go`
- `internal/adapters/outbound/llm_anthropic/adapter.go`
- `internal/adapters/outbound/llm_openaicompat/adapter.go`

## Implementation Steps
1. Create `internal/adapters/outbound/llm_lmstudio/adapter_test.go`:
   - Use `httptest.NewServer` to mock LM Studio's OpenAI-compatible API
   - Test `Ping()` — success + connection refused + timeout
   - Test `GetAvailableModels()` — returns model list, handles empty response, handles error
   - Test `GenerateCode()` — correct request body shape, parses response, handles 4xx/5xx
   - Test `Chat()` — multi-turn messages sent correctly, response extracted
   - Test `ContextLimit()` — parsed from model info
2. Create `internal/adapters/outbound/llm_ollama/adapter_test.go`:
   - Same pattern but with Ollama-specific API format (non-OpenAI response shape)
   - Test streaming response parsing if applicable
3. Create `internal/adapters/outbound/llm_anthropic/adapter_test.go`:
   - Test Anthropic-specific headers (`x-api-key`, `anthropic-version`)
   - Test Claude message format (system message handling)
   - Test API key validation
4. Create `internal/adapters/outbound/llm_openaicompat/adapter_test.go`:
   - Test generic OpenAI-compatible format
   - Test custom base URL configuration
5. Each test file: verify interface compliance with `var _ ports.LLMClient = (*Adapter)(nil)`

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] 4 new test files created, one per LLM adapter
- [ ] Each adapter has Ping, GetAvailableModels, GenerateCode, Chat test coverage
- [ ] All tests use `httptest.NewServer` — no real HTTP calls
- [ ] Interface compliance compile-check in each test file

## Anti-patterns to Avoid
- NEVER make real HTTP calls in unit tests — always use httptest
- NEVER test implementation details — test behavior through the interface
- NEVER skip error path testing — test both success and failure
