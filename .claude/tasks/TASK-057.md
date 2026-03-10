# TASK-057 — Wails binding: expose provider management + model picker to frontend

**Plan:** PLAN-006  
**Role:** devops  
**Status:** todo  
**Dependencies:** TASK-053, TASK-054

## Goal

Expose the three new orchestrator methods to the Wails 2 JS frontend through both `app.go` (the root Wails struct) and `internal/adapters/inbound/wailsbind/bind.go`.

## Changes

### `app.go`

Add three exported methods to `App`:

```go
// RegisterCloudProvider adds a new LLM backend from the UI.
func (a *App) RegisterCloudProvider(cfg domain.ProviderConfig) error {
    return a.orchestrator.RegisterCloudProvider(cfg)
}

// RemoveProvider deregisters the named LLM backend.
func (a *App) RemoveProvider(name string) error {
    return a.orchestrator.RemoveProvider(name)
}

// GetProviderModels returns the model catalogue of the named provider.
func (a *App) GetProviderModels(providerName string) ([]string, error) {
    return a.orchestrator.GetProviderModels(providerName)
}
```

### `internal/adapters/inbound/wailsbind/bind.go`

Add the same three methods to `NexusApp`:

```go
// RegisterCloudProvider adds a new LLM provider at runtime.
func (n *NexusApp) RegisterCloudProvider(cfg domain.ProviderConfig) error {
    return n.orch.RegisterCloudProvider(cfg)
}

// RemoveProvider deregisters the named provider.
func (n *NexusApp) RemoveProvider(name string) error {
    return n.orch.RemoveProvider(name)
}

// GetProviderModels returns available models for the named provider.
func (n *NexusApp) GetProviderModels(providerName string) ([]string, error) {
    return n.orch.GetProviderModels(providerName)
}
```

### Entry-point wiring — `main.go` and `cmd/nexus-daemon/main.go`

Add `buildProviderFromConfig` factory function that the orchestrator uses for `RegisterCloudProvider`:

```go
func buildProviderFromConfig(cfg domain.ProviderConfig) (ports.LLMClient, error) {
    switch cfg.Kind {
    case domain.ProviderKindLMStudio:
        return llm_lmstudio.NewLMStudioAdapter(cfg.BaseURL), nil
    case domain.ProviderKindOllama:
        return llm_ollama.NewOllamaAdapter(cfg.BaseURL, cfg.Model), nil
    case domain.ProviderKindOpenAICompat:
        return llm_openaicompat.NewAdapter(cfg.Name, cfg.BaseURL, cfg.APIKey, cfg.Model), nil
    case domain.ProviderKindAnthropic:
        return llm_anthropic.NewAdapter(cfg.APIKey, cfg.Model), nil
    default:
        return nil, fmt.Errorf("unknown provider kind: %q", cfg.Kind)
    }
}
```

Wire it: `orchestratorSvc.WithProviderFactory(buildProviderFromConfig)` immediately after `NewOrchestrator(...)`.

## Type-safety note

`domain.ProviderConfig` is serialisable to JSON. Wails 2 auto-generates TypeScript bindings from exported method signatures. The frontend will receive a fully typed `ProviderConfig` interface — no manual binding needed.

## Acceptance

- `go vet ./...` (excluding Wails-tag-gated main.go) passes
- `CGO_ENABLED=1 go build ./cmd/nexus-daemon/...` passes
- `CGO_ENABLED=1 go test -race ./internal/adapters/inbound/...` >= prior pass count
