---
id: TASK-301
title: Fix orchestrator.go bugs — double processNext, PromoteProvider error swallow, hardcoded port
role: backend
planId: PLAN-047
status: todo
dependencies: []
createdAt: 2026-03-25T00:00:00.000Z
---

## Context

Three bugs found in `internal/core/services/orchestrator.go` by the 2026-03-25 audit:

**Bug 1 — double processNext (orchestrator.go ~line 1089)**
`runWorker()` calls `processNext()` twice per wake-up: once as the loop condition and once unconditionally inside the same iteration. A task can be processed after `stopCh` fires because the stop-channel check only occurs between the two calls when the first returns `true`, but the second call runs regardless.

**Bug 2 — PromoteProvider silent persistence failure (~line 483–525)**
`PromoteProvider()` calls `AddProviderConfig()` and, if it errors, logs the error but continues to call `RegisterCloudProvider()` directly (in-memory only). The persistence failure is swallowed; the provider exists only for the current session and vanishes on restart.

**Bug 3 — hardcoded port 63987 in delegationInstruction() (~line 1429–1435)**
The domain service hard-codes the daemon address as `http://127.0.0.1:63987`. This couples core domain logic to a deployment constant. If the daemon runs on any other port, delegated agents receive stale instructions.

## Files to Read

- `internal/core/services/orchestrator.go` — full `runWorker`, `PromoteProvider`, `delegationInstruction` functions

## Implementation Steps

### Fix 1 — runWorker double processNext

Find the `runWorker` function. The pattern looks like:

```go
for processNext(...) {
    select { case <-stopCh: return }
    processNext(...)   // ← BUG: runs unconditionally
}
```

Change so `processNext` is only called once per iteration and the stop-channel is checked before each call:

```go
for {
    select {
    case <-o.stopCh:
        return
    default:
    }
    if !o.processNext(ctx) {
        // nothing to do — wait for signal
        select {
        case <-o.stopCh:
            return
        case <-o.workerSignal:
        }
    }
}
```

Preserve the existing `workerSignal` / `signalWorker` pattern.

### Fix 2 — PromoteProvider error handling

In `PromoteProvider()`, when `AddProviderConfig()` returns an error, return the error immediately — do NOT fall through to `RegisterCloudProvider()`:

```go
if _, err := o.AddProviderConfig(ctx, cfg); err != nil {
    return fmt.Errorf("orchestrator: promote provider: persist config: %w", err)
}
```

Remove the in-memory-only fallback.

### Fix 3 — delegationInstruction configurable address

Add a `daemonAddr string` field to `OrchestratorService` (default `"http://127.0.0.1:63987"`).
Add a functional option `WithDaemonAddr(addr string) Option`.
Replace the hardcoded literal in `delegationInstruction()` with `o.daemonAddr`.
Update the callers in `cmd/nexus-daemon/main.go` and `main.go` to pass `WithDaemonAddr(...)` using the configured port (read from env var `NEXUS_DAEMON_ADDR` or default).

## Acceptance Criteria

- [ ] `go vet ./internal/core/services/...` clean
- [ ] `go test ./internal/core/services/... -race -count=1` all pass
- [ ] No double `processNext` call visible in `runWorker`
- [ ] `PromoteProvider` returns error when `AddProviderConfig` fails
- [ ] `delegationInstruction` uses `o.daemonAddr`, not a hardcoded literal
