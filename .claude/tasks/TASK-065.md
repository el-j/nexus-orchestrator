---
id: TASK-065
title: "LLM adapters: check HTTP status before decode + empty response guards"
role: backend
planId: PLAN-007
status: todo
dependencies: []
createdAt: 2026-03-10T10:00:00.000Z
---

## Context
LM Studio and Ollama adapters decode JSON without checking HTTP status code first. 4xx/5xx errors produce confusing JSON decode errors. Ollama returns empty string without error check. All adapters lack response body size limits.

## Files to Read
- `internal/adapters/outbound/llm_lmstudio/adapter.go`
- `internal/adapters/outbound/llm_ollama/adapter.go`
- `internal/adapters/outbound/llm_openaicompat/adapter.go`
- `internal/adapters/outbound/llm_anthropic/adapter.go`

## Implementation Steps
1. In `llm_lmstudio` and `llm_ollama`, add HTTP status checks before `json.NewDecoder().Decode()` in `GetAvailableModels()`, `ContextLimit()`, `GenerateCode()`, and `Chat()`. Return wrapped error with status code for non-2xx.
2. In `llm_ollama`, after `GenerateCode()` decode, check if `result.Response` is empty and return a descriptive error.
3. In all 4 adapters, wrap response bodies with `io.LimitReader(resp.Body, 10<<20)` (10 MB) before decoding to prevent unbounded memory allocation from rogue providers.
4. Ensure all `resp.Body.Close()` calls are deferred immediately after checking the error from the HTTP call.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] LM Studio and Ollama adapters check HTTP status before JSON decode
- [ ] All adapters have 10 MB response body limit
- [ ] Ollama GenerateCode returns error on empty response

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/`
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
