---
id: TASK-211
title: Add HeartbeatAISession to Wails + CLI PromoteProvider
role: backend
planId: PLAN-030
status: todo
dependencies: [TASK-208]
createdAt: 2025-07-25T00:00:00.000Z
---

## Context
Two missing method implementations break the `ports.Orchestrator` contract: (1) `HeartbeatAISession` is not exposed in the Wails `App` binding (`app.go`), causing runtime errors if the frontend calls it. (2) `PromoteProvider` in the CLI remote client (`cmd/nexus-cli/main.go`) is a no-op stub returning `nil` instead of POSTing to `/api/providers/promote/{id}`.

## Files to Read
- `app.go` — Wails App struct, all method bindings (check for HeartbeatAISession)
- `cmd/nexus-cli/main.go` — RemoteOrchestrator struct, PromoteProvider stub at ~line 304
- `internal/core/ports/ports.go` — Orchestrator interface (full method list)
- `internal/adapters/inbound/httpapi/server.go` — HTTP routes for heartbeat + promote

## Implementation Steps
1. Add `HeartbeatAISession(ctx, id) error` method to `App` struct in `app.go` that delegates to `o.orch.HeartbeatAISession(ctx, id)`
2. Implement `PromoteProvider` in `RemoteOrchestrator` (`cmd/nexus-cli/main.go`): POST to `http://addr/api/providers/promote/{id}`, check response status
3. Verify all `ports.Orchestrator` methods are implemented in both `App` and `RemoteOrchestrator` — grep for any other missing stubs
4. If `wailsbind/bind.go` exists and is a duplicate, either update it to match or remove dead code

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `App` struct in `app.go` implements full `ports.Orchestrator` interface (compile check)
- [ ] `RemoteOrchestrator` in CLI actually calls HTTP endpoint for PromoteProvider
- [ ] No remaining no-op stubs that silently return nil

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER leave stub methods that silently succeed — either implement or return `ErrNotImplemented`
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
