# TASK-054 — Backend: OrchestratorService implements 3 new port methods

**Plan:** PLAN-006  
**Role:** backend  
**Status:** todo  
**Dependencies:** TASK-053

## Goal

Make `OrchestratorService` satisfy the extended `ports.Orchestrator` interface by implementing the three new methods.

## Changes

### `internal/core/services/orchestrator.go`

Add the following three methods to `OrchestratorService`:

```go
// RegisterCloudProvider builds the appropriate LLM adapter from cfg and
// registers it with the DiscoveryService.
func (o *OrchestratorService) RegisterCloudProvider(cfg domain.ProviderConfig) error

// RemoveProvider deregisters the named provider from DiscoveryService.
func (o *OrchestratorService) RemoveProvider(providerName string) error

// GetProviderModels returns the model catalogue of the named provider.
func (o *OrchestratorService) GetProviderModels(providerName string) ([]string, error)
```

### `RegisterCloudProvider` implementation details

Switch on `cfg.Kind`:
- `"lmstudio"` → `llm_lmstudio.NewLMStudioAdapter(cfg.BaseURL)`
- `"ollama"` → `llm_ollama.NewOllamaAdapter(cfg.BaseURL, cfg.Model)`
- `"openaicompat"` → `llm_openaicompat.NewAdapter(cfg.Name, cfg.BaseURL, cfg.APIKey, cfg.Model)`
- `"anthropic"` → `llm_anthropic.NewAdapter(cfg.APIKey, cfg.Model)`
- fallback → `fmt.Errorf("orchestrator: unknown provider kind %q", cfg.Kind)`

After constructing, call `o.discovery.RegisterProvider(client)`.

⚠️ Imports: add `llm_lmstudio`, `llm_ollama`, `llm_openaicompat`, `llm_anthropic` to orchestrator.go. This is acceptable because the orchestrator is the composition root for service-layer wiring — it knows about adapters only via constructor injection here.

Actually — to keep hexagonal purity, **do NOT import adapters in services**. Instead, expose a `ProviderFactory func(domain.ProviderConfig) (ports.LLMClient, error)` field on `OrchestratorService` and inject the factory at construction time (from the entry points). This keeps the service package free of adapter imports.

Revised plan:
1. Add `providerFactory func(domain.ProviderConfig) (ports.LLMClient, error)` field to `OrchestratorService`
2. Add `WithProviderFactory(fn func(domain.ProviderConfig) (ports.LLMClient, error)) *OrchestratorService` option setter
3. `RegisterCloudProvider` calls `o.providerFactory(cfg)` — returns `fmt.Errorf("orchestrator: no provider factory configured")` if nil
4. In `cmd/nexus-daemon/main.go` and `main.go`, call `orch.WithProviderFactory(buildProviderFromConfig)` after constructing the orchestrator, where `buildProviderFromConfig` is a new function in the entry point file

### `RemoveProvider` implementation

```go
func (o *OrchestratorService) RemoveProvider(providerName string) error {
    if ok := o.discovery.RemoveProvider(providerName); !ok {
        return fmt.Errorf("orchestrator: remove provider: %w", domain.ErrNotFound)
    }
    return nil
}
```

### `GetProviderModels` implementation

```go
func (o *OrchestratorService) GetProviderModels(providerName string) ([]string, error) {
    client, ok := o.discovery.GetClientByName(providerName)
    if !ok {
        return nil, fmt.Errorf("orchestrator: get provider models: %w", domain.ErrNotFound)
    }
    models, err := client.GetAvailableModels()
    if err != nil {
        return nil, fmt.Errorf("orchestrator: get provider models: %w", err)
    }
    return models, nil
}
```

## Acceptance

- `go vet ./internal/core/...` passes
- `CGO_ENABLED=1 go build ./cmd/nexus-daemon/...` passes (OrchestratorService now satisfies interface)
- `CGO_ENABLED=1 go test -race ./internal/core/services/...` >= prior pass count
