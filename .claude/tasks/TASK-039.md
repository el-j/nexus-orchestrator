---
id: TASK-039
title: "Architecture: add ContextLimit() to LLMClient port + StatusTooLarge domain constant"
role: architecture
planId: PLAN-004
status: todo
dependencies: []
createdAt: 2026-03-10T00:00:00.000Z
---

## Context
Before sending a task to an LLM, the orchestrator needs to know the loaded model's context-window size so it can abort early with a clear user-visible error (`StatusTooLarge`) instead of silently failing after a 60-300s timeout.  This task adds the `ContextLimit() int` method to the `LLMClient` port and the new `StatusTooLarge` domain constant.

## Files to Read
- `internal/core/domain/task.go`
- `internal/core/ports/ports.go`

## Implementation Steps
1. In `internal/core/domain/task.go`, add:
   ```go
   StatusTooLarge TaskStatus = "TOO_LARGE"
   ```
   after the existing status constants.

2. In `internal/core/ports/ports.go`, add to the `LLMClient` interface:
   ```go
   // ContextLimit returns the maximum number of input tokens the currently
   // loaded model can accept.  Returns 0 if unknown (no pre-flight check is
   // performed when the limit is 0).
   ContextLimit() int
   ```
   Place it between `GetAvailableModels` and `GenerateCode` for readability.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0  
  (NOTE: build will fail until TASK-040 and TASK-041 implement `ContextLimit()` on the two adapters — that is expected and acceptable at this stage)
- [ ] `domain.StatusTooLarge` constant exists and equals `"TOO_LARGE"`
- [ ] `LLMClient.ContextLimit() int` is declared in `ports.go`

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER add logic or HTTP calls to the domain or ports layers — only type declarations
