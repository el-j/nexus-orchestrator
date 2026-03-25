---
id: TASK-304
title: Fix repo_sqlite swallowed errors — tags JSON, PurgeDisconnected RowsAffected, discovered_agent LastSeen
role: backend
planId: PLAN-047
status: todo
dependencies: []
createdAt: 2026-03-25T00:00:00.000Z
---

## Context

Three silenced errors in the SQLite adapter layer found in the 2026-03-25 audit:

**Issue 1 — repo.go: tags JSON unmarshal silenced**
`_ = json.Unmarshal([]byte(tagsJSON), &t.Tags)` — a corrupted tags column in the DB produces a task with `nil` Tags and no indication of data corruption.

**Issue 2 — ai_session_repo.go: PurgeDisconnected RowsAffected error discarded**
`n, _ := res.RowsAffected()` — on failure, the deleted count silently returns 0.

**Issue 3 — discovered_agent_repo.go: LastSeen parse error silenced**
`a.LastSeen, _ = time.Parse(time.RFC3339, lastSeenStr)` — a malformed timestamp silently produces zero `time.Time`.

## Files to Read

- `internal/adapters/outbound/repo_sqlite/repo.go` — find tags unmarshal pattern
- `internal/adapters/outbound/repo_sqlite/ai_session_repo.go` — find PurgeDisconnected
- `internal/adapters/outbound/repo_sqlite/discovered_agent_repo.go` — find LastSeen parse

## Implementation Steps

### Fix 1 — tags JSON (repo.go)

Replace:

```go
_ = json.Unmarshal([]byte(tagsJSON), &t.Tags)
```

With:

```go
if tagsJSON != "" && tagsJSON != "null" {
    if err := json.Unmarshal([]byte(tagsJSON), &t.Tags); err != nil {
        log.Printf("repo_sqlite: task %s: unmarshal tags: %v", t.ID, err)
    }
}
```

Do not return an error — log and continue so one corrupted row doesn't break all task queries.

### Fix 2 — RowsAffected (ai_session_repo.go)

Replace:

```go
n, _ := res.RowsAffected()
```

With:

```go
n, err := res.RowsAffected()
if err != nil {
    return 0, fmt.Errorf("ai_session_repo: purge disconnected: rows affected: %w", err)
}
```

### Fix 3 — LastSeen parse (discovered_agent_repo.go)

Replace:

```go
a.LastSeen, _ = time.Parse(time.RFC3339, lastSeenStr)
```

With:

```go
if lastSeenStr != "" {
    if parsed, err := time.Parse(time.RFC3339, lastSeenStr); err != nil {
        log.Printf("repo_sqlite: discovered_agent %s: parse last_seen %q: %v", a.ID, lastSeenStr, err)
    } else {
        a.LastSeen = parsed
    }
}
```

## Acceptance Criteria

- [ ] `go vet ./internal/adapters/outbound/repo_sqlite/...` clean
- [ ] `go test ./internal/adapters/outbound/repo_sqlite/... -race -count=1` all pass
- [ ] No `_ = json.Unmarshal` in repo.go tags path
- [ ] `PurgeDisconnected` returns error when `RowsAffected()` fails
- [ ] `LastSeen` parse error is logged, not silently discarded
