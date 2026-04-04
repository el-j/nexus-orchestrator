---
id: TASK-461
plan: PLAN-061
status: done
wave: 1
priority: 1
---

# TASK-461: Fix main.go (Wails) — wire WithRuntimeConfigRepo

**Problem:** The Wails GUI entry point `main.go` never calls `bootstrap.WithRuntimeConfigRepo(runtimeConfigRepo)` when constructing the orchestrator. The daemon entry point `cmd/nexus-daemon/main.go` does this correctly (lines 64–65). As a result, all runtime configuration changes made in the desktop app — API tokens, queue capacity overrides — are written nowhere and silently lost on every app restart. Users of the desktop build experience phantom persistence: the UI accepts input but the app forgets it immediately.

**Fix:** Mirror the daemon's bootstrap call in `main.go` by passing `runtimeConfigRepo` to the `WithRuntimeConfigRepo` option, ensuring the desktop app and the daemon behave identically regarding persisted configuration.

**Files:**

- `main.go`
- `cmd/nexus-daemon/main.go` (reference — do not modify)

## Checklist

- [x] Open `main.go` and `cmd/nexus-daemon/main.go` side by side; identify every `bootstrap.With*` option call in the daemon that is absent in `main.go`
- [x] Confirm `runtimeConfigRepo` is already constructed in `main.go` (or add its construction using `repo_sqlite.NewRuntimeConfigRepo(db)` if missing, matching the daemon)
- [x] Add `bootstrap.WithRuntimeConfigRepo(runtimeConfigRepo)` to the `buildOrchestrator` (or equivalent) call in `main.go` at the matching position
- [x] Verify no other `bootstrap.With*` calls are missing from the desktop wiring compared to the daemon
- [x] Build the desktop target (`go build -tags desktop .`) and confirm it compiles with zero errors
- [x] Manual smoke test: change a queue capacity value in the running desktop app, restart, confirm the value is persisted

## Acceptance Criteria

- `main.go` wires `WithRuntimeConfigRepo` identically to `cmd/nexus-daemon/main.go`
- `go build -tags desktop .` succeeds
- Runtime config values survive a desktop app restart
