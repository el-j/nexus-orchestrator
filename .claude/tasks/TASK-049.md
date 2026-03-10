---
id: TASK-049
title: "DiscoveryService: model-aware routing with FindForModel() and failover"
role: backend
planId: PLAN-005
status: todo
dependencies: [TASK-044]
createdAt: 2026-03-10T06:00:00.000Z
---

## Context
Currently `DiscoveryService.DetectActive()` just returns the first alive provider.
We need `FindForModel(modelID string)` that selects a provider by model availability,
respects `ProviderHint`, and `ListProviders()` must return the extended `ProviderInfo`
(with `ActiveModel` and `Models` fields introduced in TASK-044).

## Files to Read
- `internal/core/services/discovery.go`
- `internal/core/ports/ports.go` (after TASK-044)

## Implementation Steps

Replace `internal/core/services/discovery.go` **in full** with:

```go
package services

import (
    "fmt"
    "nexus-orchestrator/internal/core/ports"
)

// DiscoveryService probes registered LLM clients and routes requests to
// appropriate providers.
type DiscoveryService struct {
    availableClients []ports.LLMClient
}

// NewDiscoveryService creates a DiscoveryService with the supplied LLM adapters.
func NewDiscoveryService(clients ...ports.LLMClient) *DiscoveryService {
    return &DiscoveryService{availableClients: clients}
}

// DetectActive returns the first LLM provider that responds to a Ping,
// or nil when none are reachable.  Used when no specific model is requested.
func (s *DiscoveryService) DetectActive() ports.LLMClient {
    for _, client := range s.availableClients {
        if client.Ping() {
            return client
        }
    }
    return nil
}

// FindForModel finds the best alive provider that carries modelID.
// Selection order:
//   1. Alive provider whose ActiveModel() matches modelID (exact, case-insensitive)
//   2. Alive provider that lists modelID in GetAvailableModels()
//
// providerHint, when non-empty, is tried first before falling back to others.
// Returns an error when no provider can serve the model.
func (s *DiscoveryService) FindForModel(modelID, providerHint string) (ports.LLMClient, error) {
    if modelID == "" {
        c := s.DetectActive()
        if c == nil {
            return nil, fmt.Errorf("discovery: no active LLM provider available")
        }
        return c, nil
    }

    candidates := s.orderedCandidates(providerHint)

    // Pass 1: provider with matching ActiveModel (fastest, no extra network call)
    for _, c := range candidates {
        if !c.Ping() {
            continue
        }
        if equalFold(c.ActiveModel(), modelID) {
            return c, nil
        }
    }

    // Pass 2: provider that lists the model in GetAvailableModels
    for _, c := range candidates {
        if !c.Ping() {
            continue
        }
        models, err := c.GetAvailableModels()
        if err != nil {
            continue
        }
        for _, m := range models {
            if equalFold(m, modelID) {
                return c, nil
            }
        }
    }

    return nil, fmt.Errorf("discovery: model %q not available on any registered provider", modelID)
}

// ListProviders returns the liveness status of every registered backend,
// including the active model and model list (best-effort; errors are silenced).
func (s *DiscoveryService) ListProviders() []ports.ProviderInfo {
    result := make([]ports.ProviderInfo, 0, len(s.availableClients))
    for _, c := range s.availableClients {
        alive := c.Ping()
        info := ports.ProviderInfo{
            Name:        c.ProviderName(),
            Active:      alive,
            ActiveModel: c.ActiveModel(),
        }
        if alive {
            if models, err := c.GetAvailableModels(); err == nil {
                info.Models = models
            }
        }
        result = append(result, info)
    }
    return result
}

// RegisterProvider adds an LLM client to the discovery pool at runtime.
func (s *DiscoveryService) RegisterProvider(c ports.LLMClient) {
    s.availableClients = append(s.availableClients, c)
}

// orderedCandidates returns clients with the hinted provider first.
func (s *DiscoveryService) orderedCandidates(hint string) []ports.LLMClient {
    if hint == "" {
        return s.availableClients
    }
    var hinted, rest []ports.LLMClient
    for _, c := range s.availableClients {
        if equalFold(c.ProviderName(), hint) {
            hinted = append(hinted, c)
        } else {
            rest = append(rest, c)
        }
    }
    return append(hinted, rest...)
}

func equalFold(a, b string) bool {
    return len(a) == len(b) && strings.EqualFold(a, b)
}
```

**Note**: Add `"strings"` to the import block.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `go build ./...` exits 0
- [ ] `FindForModel("", "")` returns first alive provider (same as `DetectActive`)
- [ ] `FindForModel("gpt-4o", "")` checks ActiveModel then GetAvailableModels
- [ ] `FindForModel("unknown-model", "")` returns a descriptive error
- [ ] `ListProviders()` populates `ActiveModel` and `Models` in returned `ProviderInfo`
- [ ] `RegisterProvider()` method exists and appends to the pool
- [ ] Provider hint is tried first, then remaining providers in registration order

## Anti-patterns to Avoid
- NEVER call `GetAvailableModels()` until Pass 1 fails (avoid unnecessary network calls)
- NEVER hold mutex locks in DiscoveryService (it has no mutable shared state post-construction)
- NEVER fail silently — return descriptive wrapped errors
