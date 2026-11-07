package services_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

func TestOrchestrator_RegisterCloudProvider_NoFactory(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	// No factory wired — must return an error.
	err := orch.RegisterCloudProvider(domain.ProviderConfig{Name: "x", Kind: domain.ProviderKindOllama})
	if err == nil {
		t.Fatal("expected error when no provider factory is configured")
	}
}

func TestOrchestrator_RegisterCloudProvider_FactoryError(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	orch.WithProviderFactory(func(cfg domain.ProviderConfig) (ports.LLMClient, error) {
		return nil, errors.New("unsupported kind")
	})

	err := orch.RegisterCloudProvider(domain.ProviderConfig{Name: "bad", Kind: "unknown"})
	if err == nil {
		t.Fatal("expected error from factory")
	}
}

func TestOrchestrator_RegisterCloudProvider_Success(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	stub := &mockLLMClient{alive: true, name: "new-provider", models: []string{"llama3"}}
	orch.WithProviderFactory(func(cfg domain.ProviderConfig) (ports.LLMClient, error) {
		return stub, nil
	})

	if err := orch.RegisterCloudProvider(domain.ProviderConfig{Name: "new-provider", Kind: domain.ProviderKindOllama}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it was added to discovery.
	providers, err := orch.GetProviders()
	if err != nil {
		t.Fatalf("GetProviders: %v", err)
	}
	found := false
	for _, p := range providers {
		if p.Name == "new-provider" {
			found = true
		}
	}
	if !found {
		t.Error("registered provider not visible via GetProviders()")
	}
}

func TestOrchestrator_RemoveProvider_NotFound(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	err := orch.RemoveProvider("ghost")
	if err == nil {
		t.Fatal("expected error when removing non-existent provider")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected domain.ErrNotFound, got: %v", err)
	}
}

func TestOrchestrator_RemoveProvider_Success(t *testing.T) {
	repo := newMemRepo()
	stub := &mockLLMClient{alive: true, name: "removeme"}
	discovery := services.NewDiscoveryService(stub)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	if err := orch.RemoveProvider("removeme"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	providers, _ := orch.GetProviders()
	for _, p := range providers {
		if p.Name == "removeme" {
			t.Error("provider still present after removal")
		}
	}
}

func TestOrchestrator_GetProviderModels_NotFound(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	_, err := orch.GetProviderModels("ghost")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected domain.ErrNotFound, got: %v", err)
	}
}

func TestOrchestrator_GetProviderModels_Success(t *testing.T) {
	repo := newMemRepo()
	stub := &mockLLMClient{alive: true, name: "mycloud", models: []string{"model-a", "model-b"}}
	discovery := services.NewDiscoveryService(stub)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	models, err := orch.GetProviderModels("mycloud")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0] != "model-a" || models[1] != "model-b" {
		t.Errorf("unexpected models: %v", models)
	}
}

// TestOrchestrator_GetDiscoveredProviders_EmptyWithoutScanner verifies that
// GetDiscoveredProviders returns a non-nil empty slice (not nil, not an error)
// when no scanner has been configured.
func TestOrchestrator_GetDiscoveredProviders_EmptyWithoutScanner(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	providers, err := orch.GetDiscoveredProviders()
	if err != nil {
		t.Fatalf("GetDiscoveredProviders: unexpected error: %v", err)
	}
	if providers == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
	if len(providers) != 0 {
		t.Errorf("expected empty slice, got %d providers", len(providers))
	}
}

// TestOrchestrator_TriggerScan_WithMockScanner verifies that TriggerScan
// delegates to the configured scanner and returns its results.
func TestOrchestrator_TriggerScan_WithMockScanner(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	scanner := &mockScanner{
		results: []domain.DiscoveredProvider{
			{ID: "port-1234", Name: "LM Studio", Kind: domain.ProviderKindLMStudio, Status: domain.DiscoveryStatusReachable},
			{ID: "cli-ollama", Name: "Ollama CLI", Kind: domain.ProviderKindOllama, Status: domain.DiscoveryStatusInstalled},
		},
	}
	orch.WithSystemScanner(scanner)

	results, err := orch.TriggerScan(context.Background())
	if err != nil {
		t.Fatalf("TriggerScan: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 providers from mock scanner, got %d", len(results))
	}
}

// TestOrchestrator_TriggerScan_NoScanner_ReturnsError verifies that calling
// TriggerScan without a configured scanner returns an error containing
// "scanner not configured".
func TestOrchestrator_TriggerScan_NoScanner_ReturnsError(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	_, err := orch.TriggerScan(context.Background())
	if err == nil {
		t.Fatal("expected error when no scanner configured, got nil")
	}
	if !strings.Contains(err.Error(), "scanner not configured") {
		t.Errorf("expected 'scanner not configured' in error, got: %v", err)
	}
}

// TestOrchestrator_PromoteProvider_NotFound verifies that PromoteProvider
// returns domain.ErrNotFound when the given ID does not match any scanned provider.
func TestOrchestrator_PromoteProvider_NotFound(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	scanner := &mockScanner{
		results: []domain.DiscoveredProvider{
			{ID: "port-1234", Name: "LM Studio", Kind: domain.ProviderKindLMStudio, Status: domain.DiscoveryStatusReachable},
			{ID: "cli-ollama", Name: "Ollama CLI", Kind: domain.ProviderKindOllama, Status: domain.DiscoveryStatusInstalled},
		},
	}
	orch.WithSystemScanner(scanner)
	if _, err := orch.TriggerScan(context.Background()); err != nil {
		t.Fatalf("TriggerScan: %v", err)
	}

	err := orch.PromoteProvider(context.Background(), "nonexistent-id")
	if err == nil {
		t.Fatal("expected error for nonexistent provider ID, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected domain.ErrNotFound, got: %v", err)
	}
}

// TestOrchestrator_PromoteProvider_NotReachable verifies that PromoteProvider
// returns an error (but not ErrNotFound) when the provider's status is not Reachable.
func TestOrchestrator_PromoteProvider_NotReachable(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	scanner := &mockScanner{
		results: []domain.DiscoveredProvider{
			{ID: "cli-ollama", Name: "Ollama CLI", Kind: domain.ProviderKindOllama, Status: domain.DiscoveryStatusInstalled},
		},
	}
	orch.WithSystemScanner(scanner)
	if _, err := orch.TriggerScan(context.Background()); err != nil {
		t.Fatalf("TriggerScan: %v", err)
	}

	err := orch.PromoteProvider(context.Background(), "cli-ollama")
	if err == nil {
		t.Fatal("expected error for non-reachable provider, got nil")
	}
	if errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected a 'not reachable' error, got ErrNotFound instead")
	}
}

// TestOrchestrator_PromoteProvider_Reachable verifies that PromoteProvider
// succeeds when the provider has DiscoveryStatusReachable and a factory is set.
func TestOrchestrator_PromoteProvider_Reachable(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	orch.WithProviderFactory(func(cfg domain.ProviderConfig) (ports.LLMClient, error) {
		return &mockLLMClient{alive: true, name: cfg.Name}, nil
	})

	scanner := &mockScanner{
		results: []domain.DiscoveredProvider{
			{
				ID:      "port-1234",
				Name:    "LM Studio",
				Kind:    domain.ProviderKindLMStudio,
				Status:  domain.DiscoveryStatusReachable,
				BaseURL: "http://127.0.0.1:1234",
			},
		},
	}
	orch.WithSystemScanner(scanner)
	if _, err := orch.TriggerScan(context.Background()); err != nil {
		t.Fatalf("TriggerScan: %v", err)
	}

	if err := orch.PromoteProvider(context.Background(), "port-1234"); err != nil {
		t.Fatalf("PromoteProvider: unexpected error: %v", err)
	}
}

func TestOrchestrator_PromoteProvider_PersistsEnabledWithoutDuplicateRegistration(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	cfgRepo := newMemProviderConfigRepo()
	orch.WithProviderConfigRepo(cfgRepo)

	var (
		factoryMu    sync.Mutex
		factoryCalls int
		seenConfigs  []domain.ProviderConfig
	)
	orch.WithProviderFactory(func(cfg domain.ProviderConfig) (ports.LLMClient, error) {
		factoryMu.Lock()
		defer factoryMu.Unlock()
		factoryCalls++
		seenConfigs = append(seenConfigs, cfg)
		return &mockLLMClient{alive: true, name: cfg.Name}, nil
	})

	scanner := &mockScanner{
		results: []domain.DiscoveredProvider{{
			ID:      "port-1234",
			Name:    "LM Studio",
			Kind:    domain.ProviderKindLMStudio,
			Status:  domain.DiscoveryStatusReachable,
			BaseURL: "http://127.0.0.1:1234",
		}},
	}
	orch.WithSystemScanner(scanner)
	if _, err := orch.TriggerScan(context.Background()); err != nil {
		t.Fatalf("TriggerScan: %v", err)
	}

	if err := orch.PromoteProvider(context.Background(), "port-1234"); err != nil {
		t.Fatalf("PromoteProvider: %v", err)
	}

	saved, err := cfgRepo.GetProviderConfig(context.Background(), "port-1234")
	if err != nil {
		t.Fatalf("GetProviderConfig: %v", err)
	}
	if !saved.Enabled {
		t.Fatal("expected promoted provider to be persisted with Enabled=true")
	}

	factoryMu.Lock()
	defer factoryMu.Unlock()
	if factoryCalls != 1 {
		t.Fatalf("factory calls: want 1, got %d", factoryCalls)
	}
	if len(seenConfigs) != 1 {
		t.Fatalf("seen configs: want 1, got %d", len(seenConfigs))
	}
	if !seenConfigs[0].Enabled {
		t.Fatal("expected enabled config passed to provider factory")
	}

	providers, err := orch.GetProviders()
	if err != nil {
		t.Fatalf("GetProviders: %v", err)
	}
	count := 0
	for _, provider := range providers {
		if provider.Name == "LM Studio" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected one live promoted provider, got %d", count)
	}
}

func TestNewOrchestrator_NilPanics(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	writer := &noopWriter{}

	mustPanic(t, "discovery is required", func() {
		services.NewOrchestrator(nil, repo, writer, nil)
	})

	mustPanic(t, "repo is required", func() {
		services.NewOrchestrator(discovery, nil, writer, nil)
	})

	mustPanic(t, "writer is required", func() {
		services.NewOrchestrator(discovery, repo, nil, nil)
	})
}

// TestPromoteProvider_EmptyBaseURL verifies that PromoteProvider returns an
// error when the discovered provider is reachable but has no BaseURL.
func TestPromoteProvider_EmptyBaseURL(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	orch.WithProviderFactory(func(cfg domain.ProviderConfig) (ports.LLMClient, error) {
		return &mockLLMClient{alive: true, name: cfg.Name}, nil
	})

	scanner := &mockScanner{
		results: []domain.DiscoveredProvider{
			{
				ID:      "no-url",
				Name:    "Headless",
				Kind:    domain.ProviderKindLMStudio,
				Status:  domain.DiscoveryStatusReachable,
				BaseURL: "", // intentionally empty
			},
		},
	}
	orch.WithSystemScanner(scanner)
	if _, err := orch.TriggerScan(context.Background()); err != nil {
		t.Fatalf("TriggerScan: %v", err)
	}

	err := orch.PromoteProvider(context.Background(), "no-url")
	if err == nil {
		t.Fatal("expected error for empty BaseURL, got nil")
	}
	if !strings.Contains(err.Error(), "no base URL") {
		t.Errorf("expected 'no base URL' in error message, got: %v", err)
	}
}

// Registering the same provider twice (the VS Code extension re-registers its models after a
// restart) must not leave two copies behind; a different model of the same vendor is a
// different provider and stays.
func TestOrchestrator_RegisterCloudProvider_ReplacesTheSameProvider(t *testing.T) {
	orch := services.NewOrchestrator(services.NewDiscoveryService(), newMemRepo(), &noopWriter{}, nil)
	defer orch.Stop()
	orch.WithProviderFactory(func(cfg domain.ProviderConfig) (ports.LLMClient, error) {
		return &mockLLMClient{alive: true, name: cfg.Name, activeModel: cfg.Model, baseURL: cfg.BaseURL}, nil
	})
	reg := func(name, model, url string) {
		t.Helper()
		if err := orch.RegisterCloudProvider(domain.ProviderConfig{Name: name, Model: model, BaseURL: url}); err != nil {
			t.Fatal(err)
		}
	}
	reg("vscode-copilot-gpt", "gpt", "http://127.0.0.1:1/v1")
	reg("vscode-copilot-gpt", "gpt", "http://127.0.0.1:1/v1") // again, same
	reg("vscode-copilot-gpt", "gpt", "http://127.0.0.1:2/v1") // the bridge came back on another port: the new one wins
	reg("Anthropic", "sonnet", "")
	reg("Anthropic", "opus", "") // same vendor, other model: a separate provider

	counts := map[string]int{}
	providers, _ := orch.GetProviders()
	for _, p := range providers {
		counts[p.Name]++
	}
	if counts["vscode-copilot-gpt"] != 1 || counts["Anthropic"] != 2 {
		t.Errorf("providers = %v; want one bridge provider and both Anthropic models", counts)
	}
	if c, ok := orch.GetClientForTest("vscode-copilot-gpt"); !ok || c.BaseURL() != "http://127.0.0.1:2/v1" {
		t.Errorf("the latest registration must win, got %v", c)
	}
}
