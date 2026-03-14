# TASK-279 — SQLite: additive migrations for PLAN-044

**Plan:** PLAN-044  
**Status:** TODO  
**Layer:** Go · outbound (`internal/adapters/outbound/repo_sqlite/`)  
**Depends on:** TASK-276  

## Objective

Extend the SQLite schema with additive migrations and a new `DiscoveredAgentRepo`.

## Changes

### `internal/adapters/outbound/repo_sqlite/repo.go`

1. Add new `CREATE TABLE IF NOT EXISTS discovered_agents` block alongside existing table creates:
```sql
CREATE TABLE IF NOT EXISTS discovered_agents (
    id               TEXT PRIMARY KEY,
    kind             TEXT NOT NULL,
    name             TEXT NOT NULL,
    detection_method TEXT NOT NULL DEFAULT '',
    process_name     TEXT NOT NULL DEFAULT '',
    cli_path         TEXT NOT NULL DEFAULT '',
    config_path      TEXT NOT NULL DEFAULT '',
    mcp_endpoint     TEXT NOT NULL DEFAULT '',
    is_running       INTEGER NOT NULL DEFAULT 0,
    last_seen        DATETIME NOT NULL
);
```

2. Add index: `CREATE INDEX IF NOT EXISTS idx_discovered_agents_kind ON discovered_agents(kind);`

3. Add four entries to the additive column migration loop (ai_sessions):
```go
{"delegated_to_nexus",  "INTEGER NOT NULL DEFAULT 0"},
{"delegation_timestamp","DATETIME"},
{"agent_capabilities",  "TEXT NOT NULL DEFAULT '[]'"},
{"detection_method",    "TEXT NOT NULL DEFAULT ''"},
```

### `internal/adapters/outbound/repo_sqlite/ai_session_repo.go`

Update `SaveAISession` INSERT/REPLACE to include new columns. Update `scanAISession` to scan new columns. Use `sql.NullString` for `delegation_timestamp` (nullable) → `*time.Time`.

### New file: `internal/adapters/outbound/repo_sqlite/discovered_agent_repo.go`

```go
type DiscoveredAgentRepo struct{ db *sql.DB }

func NewDiscoveredAgentRepo(r *Repository) *DiscoveredAgentRepo

// UpsertDiscoveredAgent inserts or replaces the discovered agent record (keyed on ID = Kind).
func (d *DiscoveredAgentRepo) UpsertDiscoveredAgent(ctx context.Context, a domain.DiscoveredAgent) error

// ListDiscoveredAgents returns all discovered agents ordered by last_seen DESC.
func (d *DiscoveredAgentRepo) ListDiscoveredAgents(ctx context.Context) ([]domain.DiscoveredAgent, error)
```

`UpsertDiscoveredAgent` uses `INSERT OR REPLACE INTO discovered_agents (id, kind, name, detection_method, process_name, cli_path, config_path, mcp_endpoint, is_running, last_seen) VALUES (...)`.

## Acceptance Criteria

- `CGO_ENABLED=1 go test -race ./internal/adapters/outbound/repo_sqlite/...` passes
- An in-memory DB opened with `repo_sqlite.New(":memory:")` successfully creates the `discovered_agents` table
- `ai_sessions` table in an existing DB inherits the four new columns without error
