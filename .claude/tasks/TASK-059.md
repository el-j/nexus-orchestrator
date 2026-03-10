# TASK-059 — Verify: full build + test + smoke validation

**Plan:** PLAN-006  
**Role:** verify  
**Status:** todo  
**Dependencies:** TASK-058

## Goal

Confirm that the entire PLAN-006 implementation is correct, complete, and production-ready.

## Checklist

### Build

```sh
CGO_ENABLED=1 go build ./cmd/nexus-daemon/...   # must output nothing (no errors)
CGO_ENABLED=1 go build ./cmd/nexus-cli/...       # must output nothing
go vet ./...                                       # must output nothing
```

### Tests

```sh
CGO_ENABLED=1 go test -race -count=1 -v ./internal/...
```

Must show:
- All prior tests still passing
- New PLAN-006 tests all green
- No data races detected

### Interface compliance

Verify `OrchestratorService` fully satisfies `ports.Orchestrator`:
```sh
grep -n "RegisterCloudProvider\|RemoveProvider\|GetProviderModels" \
    internal/core/services/orchestrator.go
```
Must return 3 matches.

### Port coverage

Verify `NexusApp` in `wailsbind/bind.go` exposes all 3 new methods:
```sh
grep -n "RegisterCloudProvider\|RemoveProvider\|GetProviderModels" \
    internal/adapters/inbound/wailsbind/bind.go app.go
```
Must return 6 matches (3 in each file).

### HTTP endpoints

Verify route registration in server.go:
```sh
grep -n "api/providers" internal/adapters/inbound/httpapi/server.go
```
Must return ≥ 4 lines (existing GET, new POST, DELETE, GET models).

### Dashboard

Verify dashboard.go contains the new UI elements:
```sh
grep -c "add-provider\|provider-form\|model-picker\|badge-noprovider\|badge-toolarge" \
    internal/adapters/inbound/httpapi/dashboard.go
```
Must return ≥ 5 matches.

### Domain

Verify `ProviderConfig` and `ProviderKind` are in domain:
```sh
grep -n "ProviderConfig\|ProviderKind" internal/core/domain/provider.go
```
Must return ≥ 6 lines.

## Final sign-off

Update `.claude/orchestrator.json`:
- All TASK-053 through TASK-059: `status` → `"done"`
- PLAN-006: `status` → `"completed"`, set `completedAt`
- `counters.nextTaskId` → `60`
- `counters.nextPlanId` → `7`
