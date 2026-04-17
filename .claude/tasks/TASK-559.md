---
id: TASK-559
title: Extract Go hardcoded constants — daemonAddr, queue cap, timeouts, MCP version from ldflags
role: backend
planId: PLAN-071
status: todo
dependencies: [TASK-557]
createdAt: 2026-04-15T00:00:00Z
---

## Context

Nine magic literals are scattered across the codebase: the daemon address is baked into core service code; queue cap 50, purge cutoff 2h, and watchdog 5min are duplicated; the MCP server hardcodes version "1.0.0" ignoring ldflags. These should be named constants or config-driven values to prevent drift and make the service configurable.

## Files to Read

- `internal/core/services/orchestrator.go`
- `internal/core/services/session_service.go`
- `internal/adapters/inbound/mcp/server.go`
- `cmd/nexus-daemon/main.go`
- `cmd/nexus-cli/main.go`

## Implementation Steps

1. Create `internal/core/services/defaults.go` (new file) with named constants:
   ```go
   const (
       DefaultQueueCap            = 50
       DefaultPurgeDisconnectAge  = 2 * time.Hour
       DefaultWatchdogStale       = 5 * time.Minute
   )
   ```
2. Replace all magic literals `50`, `2*time.Hour`, `5*time.Minute` in `orchestrator.go` and `session_service.go` with the new constants
3. In `orchestrator.go:144`: move `daemonAddr` default out of core service — it should come from an `OrchestratorOptions` field set at wiring time in `cmd/nexus-daemon/main.go` (the Options pattern already exists in the codebase; add `DaemonAddr string` to it)
4. In `mcp/server.go`: add a package-level `var Version = "dev"` that is set via ldflags in the Makefile (`-X 'nexus-orchestrator/internal/adapters/inbound/mcp.Version=$(VERSION)'`); replace the hardcoded `"1.0.0"` with this var
5. In Makefile: add the `-X` ldflags for the MCP version var to the `build-daemon` target (alongside any existing ldflags)
6. In `cmd/nexus-cli/main.go`: add `NEXUS_ADDR` env-var check: `if addr := os.Getenv("NEXUS_ADDR"); addr != "" { baseURL = addr }`

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] No naked `50`, `2*time.Hour`, `5*time.Minute` literals in orchestrator.go or session_service.go (use the new constants)
- [ ] MCP `"1.0.0"` literal replaced with a var settable by ldflags
- [ ] `NEXUS_ADDR` env var overrides daemon URL in nexus-cli
- [ ] `daemonAddr` is no longer hardcoded in core service code

## Anti-patterns to Avoid

- NEVER import adapters from core services
- NEVER add the ldflags var to `internal/core/` — it belongs in the adapter package
