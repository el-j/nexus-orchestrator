# TASK-277 — Ports: `AgentScanner` interface + Orchestrator extensions

**Plan:** PLAN-044  
**Status:** TODO  
**Layer:** Go · ports (`internal/core/ports/`)  
**Depends on:** TASK-276  

## Objective

Add the `AgentScanner` outbound port and two new methods to the `Orchestrator` inbound port. No concrete types or adapter imports in this package.

## Changes

### `internal/core/ports/ports.go`

Add new outbound port:
```go
// AgentScanner scans the local system for running AI agent tools.
// Distinct from SystemScanner which probes LLM server endpoints.
type AgentScanner interface {
    ScanAgents(ctx context.Context) ([]domain.DiscoveredAgent, error)
}
```

Add to `Orchestrator` interface:
```go
// GetDiscoveredAgents returns agents detected by the last AgentScanner run,
// triggering an on-demand scan if the cache is older than 30 seconds.
GetDiscoveredAgents(ctx context.Context) ([]domain.DiscoveredAgent, error)

// DelegateToNexus marks the AISession as delegated to the orchestrator queue
// and returns a canonical delegation instruction string for the caller to deliver
// to the external agent. Returns domain.ErrNotFound if sessionID does not exist.
DelegateToNexus(ctx context.Context, sessionID string) (string, error)
```

## Acceptance Criteria

- `go build ./internal/core/ports/...` succeeds
- Compile-time interface assertion in `services/orchestrator.go` still passes after stub implementations are added
- `go vet ./...` clean
