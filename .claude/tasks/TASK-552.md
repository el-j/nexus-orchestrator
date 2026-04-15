---
id: TASK-552
planId: PLAN-070
title: 'httpapi_client/client.go coverage + session lifecycle tests'
role: qa
status: todo
createdAt: 2026-04-14T02:00:00Z
---

# TASK-552 — httpapi_client client.go + session service lifecycle coverage

## Context

Two medium-severity test coverage gaps:

### `httpapi_client/client.go` — 18 of 30 methods at 0%

The main orchestrator HTTP client (`internal/adapters/outbound/httpapi_client/client.go`) has
18 completely untested methods including `GetAllTasks`, `GetProviders`, `GetRuntimeConfig`,
`UpdateRuntimeConfig`, `RegisterCloudProvider`, `RemoveProvider`, `GetProviderModels`,
`ListProviderConfigs`, `UpdateProviderConfig`, `GetDiscoveredProviders`, `TriggerScan`,
`GetBacklog`, `UpdateTask`, `ListAISessions`, `DeregisterAISession`,
`PurgeDisconnectedSessions`, `GetDiscoveredAgents`, `GetDiscoveredPlanFiles`.

### `session_service.go` — lifecycle methods at 0%

`TerminateAISession`, `HeartbeatAISession`, `PurgeDisconnectedSessions` in
`internal/core/services/session_service.go` (or equivalent) are not covered.

## Work Required

### `internal/adapters/outbound/httpapi_client/client_test.go` (extend existing)

Add `httptest.NewServer` based tests for the 10 highest-priority uncovered methods:

1. `GetAllTasks` — GET `/api/tasks/all`, decode array
2. `GetBacklog` — GET `/api/tasks/backlog`
3. `UpdateTask` — PUT `/api/tasks/{id}`
4. `ListAISessions` — GET `/api/ai-sessions`
5. `DeregisterAISession` — DELETE `/api/ai-sessions/{id}`
6. `PurgeDisconnectedSessions` — DELETE `/api/ai-sessions`
7. `GetDiscoveredProviders` — GET `/api/providers/discovered`
8. `TriggerScan` — POST `/api/providers/discovered/scan`
9. `GetRuntimeConfig` — GET `/api/runtime-config` (verify URL path)
10. `UpdateRuntimeConfig` — PUT `/api/runtime-config`

For each: assert correct HTTP method + path, test 200 decode + non-200 error.

### Session service lifecycle tests (extend `ai_session_service_test.go` or equivalent)

Add tests:

1. `TestAISession_Heartbeat_UpdatesTTL` — register session, heartbeat, verify TTL extended
2. `TestAISession_Terminate_SetsStatus` — register, terminate, verify status = disconnected
3. `TestAISession_Purge_RemovesDisconnected` — register, disconnect, purge, verify empty list

## File Targets

- `internal/adapters/outbound/httpapi_client/client_test.go` (extend)
- Session service test file (extend existing)

## Acceptance Criteria

- `CGO_ENABLED=1 go test -race -count=1 ./internal/adapters/outbound/httpapi_client/...` green
- `CGO_ENABLED=1 go test -race -count=1 ./internal/core/services/...` green
- `client.go` coverage improves from ~24% to ≥ 50%
