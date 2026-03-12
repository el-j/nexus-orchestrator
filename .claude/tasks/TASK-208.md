---
id: TASK-208
title: Fix TaskEvent type safety — AISessionStatus cast
role: architecture
planId: PLAN-030
status: todo
dependencies: []
createdAt: 2025-07-25T00:00:00.000Z
---

## Context
`RegisterAISession`, `DeregisterAISession`, and `runSessionCleanup` in `orchestrator.go` cast `domain.AISessionStatus` to `domain.TaskStatus` when broadcasting `TaskEvent`. These are semantically distinct enum types — mixing them breaks type safety and could confuse event subscribers expecting valid `TaskStatus` values.

## Files to Read
- `internal/core/domain/task.go` — TaskStatus enum, TaskEvent struct
- `internal/core/domain/session.go` — AISessionStatus enum
- `internal/core/services/orchestrator.go` — lines 620-660, 870-880 (cast sites)
- `internal/core/ports/ports.go` — EventBroadcaster interface

## Implementation Steps
1. Define a new `AISessionEvent` struct in `internal/core/domain/session.go` with `AISessionID`, `AISessionStatus`, `Timestamp` fields
2. Extend the `EventBroadcaster` interface in `ports.go` to support `BroadcastAISessionEvent(AISessionEvent)` or use a generic event envelope
3. Update `RegisterAISession`, `DeregisterAISession`, and `runSessionCleanup` to broadcast the new event type instead of casting to `TaskStatus`
4. Update all `EventBroadcaster` implementations (SSE hub in httpapi, Wails binding) to handle the new event type
5. Remove all `domain.TaskStatus(aiSessionStatus)` casts
6. Update test mocks to implement the new broadcast method

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] No `domain.TaskStatus(...)` casts of `AISessionStatus` remain in codebase
- [ ] New event type is properly handled in SSE hub and Wails binding
- [ ] Event subscribers can distinguish task events from AI session events

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/`
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
