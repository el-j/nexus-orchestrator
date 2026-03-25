---
id: TASK-309
title: Add repo_sqlite tests — GetStaleProcessing, GetTasksBySessionID, AppendRoutedTaskID, PurgeDisconnected, GetAISessionByExternalID, discovered_agent_repo
role: testing
planId: PLAN-047
status: todo
dependencies: [TASK-304]
createdAt: 2026-03-25T00:00:00.000Z
---

## Context

`internal/adapters/outbound/repo_sqlite/` has 64% coverage. Six methods have zero test coverage, and the entire `discovered_agent_repo.go` file is untested.

**Untested methods:**

- `GetStaleProcessing(ctx, threshold)` — threshold cutoff, empty, multi-row
- `GetTasksBySessionID(sessionID)` — empty, populated
- `AppendRoutedTaskID(ctx, sessionID, taskID)` — success, dedup, not-found
- `PurgeDisconnected(ctx, olderThan)` — boundary, zero rows, multi-row
- `GetAISessionByExternalID(ctx, externalID)` — found, not-found, ORDER BY last_activity
- `DiscoveredAgentRepo.UpsertDiscoveredAgent` — insert + update
- `DiscoveredAgentRepo.ListDiscoveredAgents` — empty + populated

## Files to Read

- `internal/adapters/outbound/repo_sqlite/repo.go` — GetStaleProcessing, GetTasksBySessionID
- `internal/adapters/outbound/repo_sqlite/ai_session_repo.go` — AppendRoutedTaskID, PurgeDisconnected, GetAISessionByExternalID
- `internal/adapters/outbound/repo_sqlite/discovered_agent_repo.go` — full file
- `internal/adapters/outbound/repo_sqlite/repo_test.go` — existing test helpers (how to open in-memory SQLite, create repo, etc.)

## Implementation Steps

Add test functions to the appropriate `_test.go` files. Use in-memory SQLite (`:memory:`) following the existing test pattern.

### Key scenarios to cover

**GetStaleProcessing:**

```go
// Insert one task with UpdatedAt = now-10min (stale), one with now-1min (fresh)
// Call GetStaleProcessing(ctx, 5*time.Minute)
// Assert only the stale task is returned
```

**AppendRoutedTaskID — deduplication:**

```go
// Append same taskID twice
// List session, assert RoutedTaskIDs has it only once
```

**PurgeDisconnected — boundary:**

```go
// Create 2 disconnected sessions: one last_activity=now-3h, one=now-1h
// PurgeDisconnected(ctx, 2*time.Hour)
// Assert only the 3h-old session was deleted, count=1
```

**GetAISessionByExternalID:**

```go
// Insert session with ExternalID="ext-1"
// Assert GetAISessionByExternalID returns it
// Assert GetAISessionByExternalID("missing") returns domain.ErrNotFound
```

**DiscoveredAgentRepo:**

```go
// UpsertDiscoveredAgent — insert, then upsert same ID with updated field, assert update applied
// ListDiscoveredAgents — empty DB returns empty slice (not nil, not error)
```

## Acceptance Criteria

- [ ] `go vet ./internal/adapters/outbound/repo_sqlite/...` clean
- [ ] `go test ./internal/adapters/outbound/repo_sqlite/... -race -count=1` all pass
- [ ] Coverage improves from 64% to ≥80%
- [ ] All 7 methods/types listed above have at least 2 test cases each
