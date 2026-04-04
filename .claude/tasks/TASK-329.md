# TASK-329: AIActivityRepository port + SQLite implementation

**Plan:** PLAN-050 · Wave 1
**Status:** DONE
**Agent:** Senior Developer

## Description

Add `AIActivityRepository` port interface: `SaveActivity(ctx, AIActivity) error`, `ListActivities(ctx, since time.Time, filters ActivityFilter) ([]AIActivity, error)`, `PurgeOlderThan(ctx, cutoff time.Time) (int64, error)`.

Implement in `internal/adapters/outbound/repo_sqlite/activity_repo.go` with SQLite table `ai_activities`. Default retention: 24 hours. Index on (timestamp, agent_name, project_path).

## Acceptance

- Port interface in ports.go
- SQLite implementation with auto-migration
- `PurgeOlderThan` works for retention policy
- Tests for CRUD operations

## Completed

Added `AIActivityRepository` port and SQLite `ai_activities` table implementation with 24h retention and CRUD tests.
