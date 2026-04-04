---
id: TASK-314
title: Split httpapi/server.go into server.go + handlers_tasks.go + handlers_providers.go + handlers_sessions.go
role: architecture
planId: PLAN-047
status: todo
dependencies: [TASK-303, TASK-308]
createdAt: 2026-03-25T00:00:00.000Z
---

## Context

`internal/adapters/inbound/httpapi/server.go` is ~854 lines with ~40 handler methods covering 4 resource domains. Navigating to a specific handler requires knowing line numbers. Splitting by resource domain makes the route table in `Handler()` the single place to understand the full API surface.

Same approach as TASK-313 — same package, same struct, pure file split.

## Proposed Split

| File                    | Contents                                                                                                                                                                                                                                                                           |
| ----------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `server.go`             | `Server` struct, `NewServer`, `Handler()` (route table), middleware, `StartServer`, `writeJSONError`, `writeJSON`, SSE broadcaster types                                                                                                                                           |
| `handlers_tasks.go`     | handleSubmitTask, handleGetTask, handleGetQueue, handleGetAllTasks, handleCancelTask, handleCreateDraft, handleGetBacklog, handlePromoteTask, handleUpdateTask, handleClaimTask, handleUpdateTaskStatus, handleHeartbeatTask, handleGetSessionTasks                                |
| `handlers_providers.go` | handleGetProviders, handleRegisterCloudProvider, handleRemoveProvider, handleGetProviderModels, handleAddProviderConfig, handleUpdateProviderConfig, handleRemoveProviderConfig, handleListProviderConfigs, handleGetDiscoveredProviders, handleTriggerScan, handlePromoteProvider |
| `handlers_sessions.go`  | handleRegisterAISession, handleListAISessions, handleDeregisterAISession, handleTerminateAISession, handleHeartbeatAISession, handlePurgeDisconnectedSessions, handleGetDiscoveredAgents, handleDelegateToNexus, handleSSE, handleDashboard, handleHowTo, handleWellKnownNexus     |

## Implementation Steps

1. Read the full `internal/adapters/inbound/httpapi/server.go`.
2. Create 3 new files (same package `httpapi`).
3. Move handler methods to appropriate files; ensure imports follow the code.
4. Keep `Handler()` route registration and `Server` struct in `server.go`.
5. Run `go build` and `go test` — fix any missing imports.

## Acceptance Criteria

- [ ] `server.go` ≤150 lines (struct + Handler() + middleware + helpers)
- [ ] 3 handler files exist with domain-grouped handlers
- [ ] `go vet ./internal/adapters/inbound/httpapi/...` clean
- [ ] `go test ./internal/adapters/inbound/httpapi/... -race -count=1` all pass
- [ ] `go build ./cmd/nexus-daemon/...` exits 0
