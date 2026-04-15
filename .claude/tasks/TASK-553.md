---
id: TASK-553
planId: PLAN-070
title: 'Fix GetDiscoveredPlanFiles nil/nil + honest tray stub with feature flag'
role: backend
status: todo
createdAt: 2026-04-14T02:00:00Z
---

# TASK-553 — Backend quality: nil/nil sentinel + tray stub

## Context

### Issue 1: `GetDiscoveredPlanFiles` returns `nil, nil` on unconfigured subsystem

`internal/core/services/session_service.go` (or `orchestrator.go`) — `GetDiscoveredPlanFiles`
returns `nil, nil` in three cases where the scanning subsystem was never wired:

- Both `agentScanner` and `planFileRepo` are nil
- `planFileRepo` is nil after a scan
- `agentScanner` is nil and `planFileRepo` is nil

Callers cannot distinguish "no results found" from "the scanning subsystem was never
initialised". Should return a typed `domain.ErrNotFound` or a sentinel `ErrSubsystemNotWired`.

Fix: define a package-level `var ErrSubsystemNotConfigured = errors.New("plan scanning subsystem not configured")` in `domain` or `ports`. Return it when both repos are nil. Callers (HTTP handler, MCP tool, Wails binding) can then check for it and surface a 503 / helpful message.

### Issue 2: Tray adapter is a non-functional stub with no feature tracking

`internal/adapters/inbound/tray/tray.go` — `Start()` is a no-op, `Enabled()` always returns
`false`, `UpdateStatus()` builds a string and discards it. This is silently non-functional
with no way for the user or operator to know the feature is absent.

Fix: Add a build tag (`//go:build !notray`) or a runtime config flag `NEXUS_TRAY_ENABLED=true`.
When disabled (default), `Enabled()` returns false AND logs a one-time startup message:
`"tray icon disabled (NEXUS_TRAY_ENABLED not set)"`. Add a TODO comment with a GitHub issue
reference explaining what's needed to implement it (main-thread dispatch on macOS/Windows).
The goal is honest: the stub should advertise its own absence, not silently do nothing.

## File Targets

- `internal/core/services/` — `GetDiscoveredPlanFiles` location (read to confirm)
- `internal/core/domain/errors.go` or `ports` — add sentinel if needed
- `internal/adapters/inbound/tray/tray.go`

## Acceptance Criteria

- `go vet ./...` clean
- `CGO_ENABLED=1 go build ./...` clean
- `GetDiscoveredPlanFiles` returns `ErrSubsystemNotConfigured` (not `nil, nil`) when both repos are nil
- `tray.Enabled()` returns false; startup logs one message explaining why
- All existing tests still pass
