package services

import (
	"context"
	"fmt"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"

	"github.com/google/uuid"
)

// GetProviders returns the liveness status of every registered LLM backend.
func (o *OrchestratorService) GetProviders() ([]ports.ProviderInfo, error) {
	return o.discovery.ListProviders(), nil
}

// RegisterCloudProvider builds a new LLM adapter from cfg using the registered
// factory and adds it to the DiscoveryService.
func (o *OrchestratorService) RegisterCloudProvider(cfg domain.ProviderConfig) error {
	o.mu.Lock()
	factory := o.providerFactory
	o.mu.Unlock()
	if factory == nil {
		return fmt.Errorf("orchestrator: no provider factory configured")
	}
	client, err := factory(cfg)
	if err != nil {
		return fmt.Errorf("orchestrator: register cloud provider: %w", err)
	}
	o.discovery.RegisterProvider(client)
	return nil
}

// RemoveProvider deregisters the named provider from DiscoveryService.
// Returns domain.ErrNotFound when no provider with that name is registered.
func (o *OrchestratorService) RemoveProvider(providerName string) error {
	if ok := o.discovery.RemoveProvider(providerName); !ok {
		return fmt.Errorf("orchestrator: remove provider: %w", domain.ErrNotFound)
	}
	return nil
}

// GetProviderModels returns the model catalogue of the named provider.
// Returns domain.ErrNotFound when no provider with that name is registered.
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

// GetDiscoveredProviders returns auto-detected AI tools from the local system
// that have NOT yet been promoted to active/configured providers.
func (o *OrchestratorService) GetDiscoveredProviders() ([]domain.DiscoveredProvider, error) {
	o.scanMu.RLock()
	defer o.scanMu.RUnlock()
	if o.lastScan == nil {
		return []domain.DiscoveredProvider{}, nil
	}
	result := make([]domain.DiscoveredProvider, len(o.lastScan))
	copy(result, o.lastScan)
	return result, nil
}

// TriggerScan requests an immediate re-scan of the local system for AI providers.
func (o *OrchestratorService) TriggerScan(ctx context.Context) ([]domain.DiscoveredProvider, error) {
	o.mu.Lock()
	scanner := o.scanner
	o.mu.Unlock()

	if scanner == nil {
		return []domain.DiscoveredProvider{}, fmt.Errorf("orchestrator: trigger scan: scanner not configured")
	}

	results, err := scanner.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("orchestrator: trigger scan: %w", err)
	}

	o.scanMu.Lock()
	o.lastScan = results
	o.scanMu.Unlock()

	o.discovery.InvalidateHealthCache()

	return results, nil
}

// PromoteProvider converts a discovered provider into an active registered backend.
func (o *OrchestratorService) PromoteProvider(ctx context.Context, discoveredID string) error {
	o.scanMu.RLock()
	var found *domain.DiscoveredProvider
	for i := range o.lastScan {
		if o.lastScan[i].ID == discoveredID {
			discovered := o.lastScan[i]
			found = &discovered
			break
		}
	}
	o.scanMu.RUnlock()

	if found == nil {
		return fmt.Errorf("orchestrator: promote provider: %w", domain.ErrNotFound)
	}
	if found.Status != domain.DiscoveryStatusReachable {
		return fmt.Errorf("orchestrator: promote provider: provider %q is not reachable (status: %s)", found.Name, found.Status)
	}
	if found.BaseURL == "" {
		return fmt.Errorf("orchestrator: promote provider: discovered provider has no base URL")
	}

	cfg := domain.ProviderConfig{
		ID:      found.ID,
		Name:    found.Name,
		Kind:    found.Kind,
		BaseURL: found.BaseURL,
		Enabled: true,
	}

	if o.providerConfigRepo != nil {
		if _, err := o.AddProviderConfig(ctx, cfg); err != nil {
			return fmt.Errorf("orchestrator: promote provider: persist config: %w", err)
		}
		return nil
	}

	if err := o.RegisterCloudProvider(cfg); err != nil {
		return fmt.Errorf("orchestrator: promote provider: %w", err)
	}

	return nil
}

// AddProviderConfig persists a new provider config and, when Enabled, instantiates
// and registers an adapter via the configured providerFactory.
func (o *OrchestratorService) AddProviderConfig(ctx context.Context, cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	o.mu.Lock()
	repo := o.providerConfigRepo
	factory := o.providerFactory
	o.mu.Unlock()

	if cfg.ID == "" {
		cfg.ID = uuid.NewString()
	}
	now := time.Now()
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = now
	}
	cfg.UpdatedAt = now

	if repo != nil {
		if err := repo.SaveProviderConfig(ctx, cfg); err != nil {
			return domain.ProviderConfig{}, fmt.Errorf("orchestrator: add provider config: %w", err)
		}
	}

	if cfg.Enabled && factory != nil {
		client, err := factory(cfg)
		if err != nil {
			return domain.ProviderConfig{}, fmt.Errorf("orchestrator: add provider config: build adapter: %w", err)
		}
		o.discovery.RegisterProvider(client)
	}

	return cfg, nil
}

// UpdateProviderConfig overwrites an existing provider config and refreshes its
// in-process adapter registration.
func (o *OrchestratorService) UpdateProviderConfig(ctx context.Context, cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	o.mu.Lock()
	repo := o.providerConfigRepo
	factory := o.providerFactory
	o.mu.Unlock()

	if repo == nil {
		return domain.ProviderConfig{}, fmt.Errorf("orchestrator: update provider config: no config repo configured")
	}

	old, err := repo.GetProviderConfig(ctx, cfg.ID)
	if err != nil {
		return domain.ProviderConfig{}, fmt.Errorf("orchestrator: update provider config: %w", err)
	}

	cfg.CreatedAt = old.CreatedAt
	cfg.UpdatedAt = time.Now()

	if err := repo.SaveProviderConfig(ctx, cfg); err != nil {
		return domain.ProviderConfig{}, fmt.Errorf("orchestrator: update provider config: %w", err)
	}

	// Remove old adapter registration (name may have changed).
	o.discovery.RemoveProvider(old.Name)

	if cfg.Enabled && factory != nil {
		client, err := factory(cfg)
		if err != nil {
			return domain.ProviderConfig{}, fmt.Errorf("orchestrator: update provider config: build adapter: %w", err)
		}
		o.discovery.RegisterProvider(client)
	}

	return cfg, nil
}

// RemoveProviderConfig deletes the persisted provider config identified by id
// and deregisters its adapter.
func (o *OrchestratorService) RemoveProviderConfig(ctx context.Context, id string) error {
	o.mu.Lock()
	repo := o.providerConfigRepo
	o.mu.Unlock()

	if repo == nil {
		return fmt.Errorf("orchestrator: remove provider config: no config repo configured")
	}

	cfg, err := repo.GetProviderConfig(ctx, id)
	if err != nil {
		return fmt.Errorf("orchestrator: remove provider config: %w", err)
	}

	if err := repo.DeleteProviderConfig(ctx, id); err != nil {
		return fmt.Errorf("orchestrator: remove provider config: %w", err)
	}

	o.discovery.RemoveProvider(cfg.Name)
	return nil
}

// ListProviderConfigs returns all persisted provider configuration records.
// Returns an empty slice when no repository is configured.
func (o *OrchestratorService) ListProviderConfigs(ctx context.Context) ([]domain.ProviderConfig, error) {
	o.mu.Lock()
	repo := o.providerConfigRepo
	o.mu.Unlock()

	if repo == nil {
		return []domain.ProviderConfig{}, nil
	}

	cfgs, err := repo.ListProviderConfigs(ctx)
	if err != nil {
		return nil, fmt.Errorf("orchestrator: list provider configs: %w", err)
	}
	if cfgs == nil {
		return []domain.ProviderConfig{}, nil
	}
	return cfgs, nil
}
