---
id: TASK-302
title: Wire CLI remoteOrchestrator stubs — GetDiscoveredAgents, DelegateToNexus, TerminateAISession force
role: backend
planId: PLAN-047
status: done
dependencies: []
createdAt: 2026-03-25T00:00:00.000Z
completedAt: 2026-03-25T09:00:00.000Z
---

## Context

In `cmd/nexus-cli/main.go`, the `remoteOrchestrator` struct implements `ports.Orchestrator` over HTTP. Two methods are complete no-ops:

- `GetDiscoveredAgents()` — returns `(nil, nil)` with no HTTP call (line ~610)
- `DelegateToNexus()` — returns `("", nil)` with no HTTP call (line ~614)

Additionally, `TerminateAISession()` accepts a `force bool` parameter but never forwards it to the server.

## Files to Read

- `cmd/nexus-cli/main.go` — full `remoteOrchestrator` type, especially lines 480–616
- `internal/adapters/inbound/httpapi/server.go` — to confirm route paths

## Implementation Steps

### Fix GetDiscoveredAgents

Wire up a real `GET /api/ai-sessions/discovered` call (same pattern as other methods):

```go
func (r *remoteOrchestrator) GetDiscoveredAgents(ctx context.Context) ([]domain.DiscoveredAgent, error) {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.base+"/api/ai-sessions/discovered", nil)
    if err != nil {
        return nil, err
    }
    resp, err := r.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    var agents []domain.DiscoveredAgent
    if err := json.NewDecoder(resp.Body).Decode(&agents); err != nil {
        return nil, err
    }
    return agents, nil
}
```

### Fix DelegateToNexus

Wire up a real `POST /api/ai-sessions/{id}/delegate` call:

```go
func (r *remoteOrchestrator) DelegateToNexus(ctx context.Context, sessionID string) (string, error) {
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.base+"/api/ai-sessions/"+sessionID+"/delegate", nil)
    // ... do req, decode string response
}
```

Check the HTTP handler's response format to match the JSON field name.

### Fix TerminateAISession force flag

In the existing `TerminateAISession` implementation, add `?force=true` to the URL when `force` is true:

```go
url := r.base + "/api/ai-sessions/" + id + "/terminate"
if force {
    url += "?force=true"
}
```

Or pass it as JSON body `{"force": true}` — match whatever the server handler expects (check `handleTerminateAISession`).

## Acceptance Criteria

- [ ] `go vet ./cmd/nexus-cli/...` clean
- [ ] `go build ./cmd/nexus-cli/...` exits 0
- [ ] `GetDiscoveredAgents` makes a real HTTP GET call
- [ ] `DelegateToNexus` makes a real HTTP POST call
- [ ] `TerminateAISession` forwards `force` to the server
