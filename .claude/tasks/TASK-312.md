---
id: TASK-312
title: Extract remoteOrchestrator from cmd/nexus-cli/main.go to internal/adapters/outbound/httpapi_client/
role: architecture
planId: PLAN-047
status: todo
dependencies: [TASK-302]
createdAt: 2026-03-25T00:00:00.000Z
---

## Context

`cmd/nexus-cli/main.go` contains a 580-line `remoteOrchestrator` struct that fully implements `ports.Orchestrator` over HTTP. It is currently embedded in a `cmd/` file, making it impossible to test independently. The actual `main()` function is only ~10 lines.

Moving it to `internal/adapters/outbound/httpapi_client/client.go` follows hexagonal architecture (outbound adapter) and allows unit testing without launching a daemon.

## Files to Read

- `cmd/nexus-cli/main.go` — full file; the `remoteOrchestrator` struct and all its methods
- `internal/adapters/inbound/cli/root.go` — to understand how CLI commands call the orchestrator

## Implementation Steps

1. **Create `internal/adapters/outbound/httpapi_client/client.go`**:

```go
// Package httpapi_client provides an HTTP-backed implementation of ports.Orchestrator
// that delegates all calls to a running nexus daemon.
package httpapi_client

import (
    "nexus-orchestrator/internal/core/ports"
    // ...
)

// Client implements ports.Orchestrator over HTTP.
type Client struct {
    base   string
    client *http.Client
}

// New creates a Client targeting the given daemon base URL (e.g. "http://127.0.0.1:63987").
func New(baseURL string) *Client {
    return &Client{base: baseURL, client: &http.Client{Timeout: 30 * time.Second}}
}

// Ensure interface compliance at compile time.
var _ ports.Orchestrator = (*Client)(nil)

// ... all method implementations moved from cmd/nexus-cli/main.go
```

2. **In `cmd/nexus-cli/main.go`**: delete the `remoteOrchestrator` struct and all its methods. Replace the instantiation with:

```go
orch := httpapi_client.New(daemonURL)
```

3. Adjust the import path and any references to the old type name (`remoteOrchestrator` → `*httpapi_client.Client`).

4. Optionally add a basic test file `internal/adapters/outbound/httpapi_client/client_test.go` that uses `httptest.NewServer` to verify a couple of methods make the right HTTP calls.

## Acceptance Criteria

- [ ] `internal/adapters/outbound/httpapi_client/client.go` exists
- [ ] `cmd/nexus-cli/main.go` is ≤30 lines
- [ ] `go vet ./... ` clean
- [ ] `go build ./cmd/nexus-cli/...` exits 0
- [ ] `go test ./... -race -count=1` all pass
- [ ] `var _ ports.Orchestrator = (*httpapi_client.Client)(nil)` compile-time check present
