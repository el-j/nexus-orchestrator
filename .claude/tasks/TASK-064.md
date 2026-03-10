---
id: TASK-064
title: "Type safety: typed MessageRole + TaskEvent.Type enum + domain String() methods"
role: architecture
planId: PLAN-007
status: todo
dependencies: []
createdAt: 2026-03-10T10:00:00.000Z
---

## Context
`Message.Role` is an untyped `string` in the domain. `TaskEvent.Type` is also an untyped `string` with ad-hoc formatting. `TaskStatus` and `ProviderKind` lack `String()` methods. This hurts type safety, debuggability, and maintainability.

## Files to Read
- `internal/core/domain/session.go`
- `internal/core/domain/task.go`
- `internal/core/domain/provider.go`
- `internal/core/ports/ports.go`
- `internal/core/services/orchestrator.go`

## Implementation Steps
1. In `domain/session.go`, add `type MessageRole string` with constants `RoleUser MessageRole = "user"` and `RoleAssistant MessageRole = "assistant"`. Change `Message.Role` field type from `string` to `MessageRole`.
2. In `ports/ports.go`, add `type EventType string` with constants `EventTaskQueued EventType = "task.queued"`, `EventTaskProcessing`, `EventTaskCompleted`, `EventTaskFailed`, `EventTaskCancelled`, `EventTaskTooLarge`, `EventTaskNoProvider`. Change `TaskEvent.Type` from `string` to `EventType`.
3. Add `String()` methods on `TaskStatus` and `ProviderKind` that return `string(t)`.
4. Update `orchestrator.go` to use `domain.RoleUser`, `domain.RoleAssistant`, and `ports.EventTask*` constants instead of string literals.
5. Fix all callsites in tests and adapters that use raw string literals for roles/events.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `domain.RoleUser` and `domain.RoleAssistant` are typed constants
- [ ] `ports.EventTaskQueued` etc. are typed constants
- [ ] `TaskStatus("QUEUED").String()` returns `"QUEUED"`

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/`
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
