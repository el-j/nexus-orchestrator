---
id: TASK-493
plan: PLAN-064
status: done
wave: 1
priority: 1
---

# TASK-493: Add execution_engine.go unit tests

## Description

`internal/core/services/execution_engine.go` contains every concrete decision made during task execution: provider selection by hint, chat-vs-generate branching, token estimation, code extraction, and file output. Despite this, it has zero isolation tests. Every execution path in production runs through it, making this the highest-risk untested file in the codebase.

## Checklist

- [ ] Create `internal/core/services/execution_engine_test.go`
- [ ] Test `selectProviderForTask`: matching hint selects correct provider; non-matching hint falls back; no hint uses first available provider
- [ ] Test `buildChatContext`: empty session history produces one system message; populated session history is prepended in order; project path appears in system message
- [ ] Test `executeGeneration`: mock `LLMClient.Chat()` returns response -> result captured; mock `Chat()` returns error -> fallback to `GenerateCode()` attempted; both fail -> error returned
- [ ] Test `estimateTokens`: known ASCII string returns expected estimate; empty string returns 0; non-ASCII characters handled without panic
- [ ] Test `extractCode`: input with fenced ``` block returns only inner content; input without code block returned unchanged; multiple code blocks returns first block only
- [ ] Test `writeTaskOutput`: absolute path outside project root is rejected (traversal guard); valid relative path writes file; parent directories created if missing
- [ ] Use mock structs matching the style in `orchestrator_test.go` (implement `ports.LLMClient`, `ports.FileWriter` interfaces inline)
- [ ] All tests pass under `CGO_ENABLED=1 go test -race ./internal/core/services/...`

## Files

- `internal/core/services/execution_engine_test.go` (create)
- `internal/core/services/execution_engine.go` (reference only)

## Acceptance Criteria

- `execution_engine_test.go` exists with minimum 8 test cases
- Each exported/unexported function named above has >=1 happy-path and >=1 error-path test
- `go test -race ./internal/core/services/...` exits 0 with no data-race warnings
- No production code is modified to accommodate tests (use interface mocks only)
