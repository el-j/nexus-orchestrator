# TASK-053 — Architecture: ProviderConfig domain type + extend Orchestrator port + DiscoveryService safety

**Plan:** PLAN-006  
**Role:** architecture  
**Status:** todo  
**Dependencies:** TASK-052

## Goal

Lay the foundational contracts that all concrete implementations in TASK-054 through TASK-057 depend on.

## Changes

### 1. `internal/core/domain/provider.go` (NEW FILE)

```go
package domain

// ProviderKind identifies the adapter family for a cloud LLM provider.
type ProviderKind string

const (
    ProviderKindLMStudio    ProviderKind = "lmstudio"
    ProviderKindOllama      ProviderKind = "ollama"
    ProviderKindOpenAICompat ProviderKind = "openaicompat"
    ProviderKindAnthropic   ProviderKind = "anthropic"
)

// ProviderConfig is a serialisable configuration record for registering or
// editing an LLM provider at runtime (e.g. through the UI or HTTP API).
type ProviderConfig struct {
    Name    string       `json:"name"`
    Kind    ProviderKind `json:"kind"`
    // BaseURL is the API endpoint root (required for openaicompat / lmstudio / ollama).
    BaseURL string       `json:"baseUrl,omitempty"`
    // APIKey is the bearer token (required for openaicompat / anthropic).
    APIKey  string       `json:"apiKey,omitempty"`
    // Model is the default / active model identifier.
    Model   string       `json:"model,omitempty"`
}
```

### 2. `internal/core/ports/ports.go` — extend `Orchestrator` interface

Add three methods after `CancelTask`:

```go
// RegisterCloudProvider dynamically adds a new LLM backend using the supplied
// configuration. Returns an error if the kind is unknown or the name is already
// registered.
RegisterCloudProvider(cfg domain.ProviderConfig) error

// RemoveProvider deregisters the provider with the given name.
// Returns domain.ErrNotFound when no provider with that name exists.
RemoveProvider(providerName string) error

// GetProviderModels returns the model catalogue of the named provider.
// Returns domain.ErrNotFound when no provider with that name exists.
GetProviderModels(providerName string) ([]string, error)
```

### 3. `internal/core/services/discovery.go` — concurrency + new helpers

- Protect `availableClients` slice with `sync.RWMutex` (read-lock for `DetectActive`, `FindForModel`, `ListProviders`; write-lock for `RegisterProvider`, `RemoveProvider`)
- Add `RemoveProvider(name string) bool` — removes first client whose `ProviderName()` matches (case-insensitive); returns `false` if not found
- Add `GetClientByName(name string) (ports.LLMClient, bool)` — returns first case-insensitive match

## Acceptance

- `go vet ./internal/core/...` passes
- `CGO_ENABLED=1 go build ./cmd/nexus-daemon/...` **fails** (expected — OrchestratorService doesn't implement the new interface yet); that's the signal that TASK-054 is unblocked
