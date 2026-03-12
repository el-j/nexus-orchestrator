---
id: TASK-216
title: Decompose processNext god method
role: backend
planId: PLAN-030
status: todo
dependencies: [TASK-208, TASK-209]
createdAt: 2025-07-25T00:00:00.000Z
---

## Context
`processNext()` in `orchestrator.go` is ~177 lines handling provider selection, token estimation, session loading, chat vs fallback, retries, file writing, and status updates. This god method violates single responsibility and makes debugging difficult. It should be decomposed into focused helper methods.

## Files to Read
- `internal/core/services/orchestrator.go` — lines ~888-1065 (`processNext` method)
- `internal/core/services/orchestrator.go` — line ~1140 (`estimateTokens` helper)
- `internal/core/services/orchestrator_test.go` — existing test coverage for processNext behavior

## Implementation Steps
1. Extract `selectProviderForTask(task) (ProviderInfo, error)` — provider selection + context limit + token estimation logic
2. Extract `buildChatContext(task, session) ([]domain.Message, error)` — session loading, context file reading, message construction, token budget trimming
3. Extract `executeGeneration(ctx, task, provider, messages) (string, error)` — chat vs fallback dispatch, retry loop
4. Extract `writeTaskOutput(task, result string) error` — file writing + status update to completed
5. Refactor `processNext()` to call these 4 helpers sequentially — keeping it under 30 lines
6. Move `estimateTokens` and `maxResponseTokens` to named constants at package level
7. Preserve all existing behavior — this is a pure refactor with NO behavior changes
8. Run existing tests to verify zero regressions

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `processNext()` is under 40 lines
- [ ] All extracted methods are private (unexported)
- [ ] Zero behavior changes — same test results before and after

## Anti-patterns to Avoid
- NEVER change behavior during a refactor — extract only
- NEVER export helper methods that are implementation details
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/`
