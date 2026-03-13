# TASK-264 — Add PurgeDisconnected to AISessionRepository port + SQLite impl

**Plan**: PLAN-041  
**Status**: done

## What
Add `PurgeDisconnected(ctx context.Context, olderThan time.Duration) (int, error)` to:
1. `ports.AISessionRepository` interface in `internal/core/ports/ports.go`
2. `repo_sqlite.AISessionRepo` implementation in `internal/adapters/outbound/repo_sqlite/ai_session_repo.go`

The method deletes rows where `status = 'disconnected'` AND `last_activity < now - olderThan`.
Returns count of deleted rows.
