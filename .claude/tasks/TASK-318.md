# TASK-318: Ports + OrchestratorService + SQLite + HTTP API for plan-file discovery

**Plan:** PLAN-048
**Role:** backend
**Dependencies:** TASK-315, TASK-316, TASK-317

## Goal

Wire plan-file discovery into the full stack: OrchestratorService calls ScanPlanFiles, results are persisted in SQLite, and exposed via two HTTP endpoints. Also ensure the updated `AISession.ModelID` field is persisted in SQLite and returned by the API.

## Changes

### 1. `internal/core/services/session_service.go`

Add `GetDiscoveredPlanFiles` method:

```go
func (o *OrchestratorService) GetDiscoveredPlanFiles(ctx context.Context, projectPath string) ([]domain.DiscoveredPlanFile, error)
```

- Checks if `o.agentScanner == nil` — if so returns empty slice, no error
- Calls `o.agentScanner.ScanPlanFiles(ctx, []string{projectPath})`
- If `planFileRepo != nil`: upserts each result, then returns `planFileRepo.ListPlanFiles(ctx, projectPath)`
- Otherwise returns raw scan results
- No cache needed (scans are fast, file-stat only)

Add `planFileRepo` field to `OrchestratorService` struct + `WithPlanFileRepo(r DiscoveredPlanFileRepo) Option` functional option.

### 2. `internal/core/ports/ports.go`

Add outbound port:

```go
type DiscoveredPlanFileRepo interface {
    UpsertPlanFile(ctx context.Context, f domain.DiscoveredPlanFile) error
    ListPlanFiles(ctx context.Context, projectPath string) ([]domain.DiscoveredPlanFile, error)
    DeleteStalePlanFiles(ctx context.Context, olderThan time.Duration) (int, error)
}
```

### 3. `internal/adapters/outbound/repo_sqlite/plan_file_repo.go` (new file)

SQLite table:

```sql
CREATE TABLE IF NOT EXISTS discovered_plan_files (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL UNIQUE,
    kind TEXT NOT NULL,
    format TEXT NOT NULL,
    project_path TEXT NOT NULL,
    summary TEXT,
    last_modified INTEGER NOT NULL,
    is_active INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL
)
```

Implement `DiscoveredPlanFileRepo` with upsert (INSERT OR REPLACE), list (WHERE project_path = ?), and delete stale (WHERE updated_at < ?).

### 4. `internal/adapters/outbound/repo_sqlite/repo.go`

Add migration for `discovered_plan_files` table. Also add migration for `ai_sessions` table: `ALTER TABLE ai_sessions ADD COLUMN model_id TEXT`.

### 5. `internal/adapters/inbound/httpapi/handlers_sessions.go`

Add two handlers:

```go
// GET /api/plans/discovered?projectPath=<path>
func (s *Server) handleGetDiscoveredPlanFiles(w http.ResponseWriter, r *http.Request)

// POST /api/plans/discovered/scan?projectPath=<path>
func (s *Server) handleScanPlanFiles(w http.ResponseWriter, r *http.Request)
```

### 6. `internal/adapters/inbound/httpapi/server.go` — routes

Add to `Handler()`:

```go
r.Get("/api/plans/discovered", s.handleGetDiscoveredPlanFiles)
r.Post("/api/plans/discovered/scan", s.handleScanPlanFiles)
```

### 7. `main.go` + `cmd/nexus-daemon/main.go`

Wire `PlanFileRepo` into orchestrator via `WithPlanFileRepo`.

### 8. AISession ModelID persistence

In `ai_session_repo.go`: add `model_id` to INSERT/SELECT/UPDATE queries for `AISession`.
In `handlers_sessions.go` `handleRegisterAISession`: accept `modelId` from JSON body.

## Acceptance Criteria

- [ ] `GetDiscoveredPlanFiles` wired end-to-end
- [ ] `GET /api/plans/discovered?projectPath=/path/to/project` returns `[]DiscoveredPlanFile` JSON
- [ ] `POST /api/plans/discovered/scan` triggers fresh scan and returns results
- [ ] `discovered_plan_files` SQLite table created by migration
- [ ] `AISession.ModelID` persisted and returned in `GET /api/ai-sessions`
- [ ] `go build ./...` clean, all tests pass
