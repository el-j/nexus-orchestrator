package services_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

// failingProviderRepo wraps memProviderConfigRepo with injectable failures.
type failingProviderRepo struct {
	*memProviderConfigRepo
	getErr, deleteErr, listErr error
	nilList                    bool
}

func (f *failingProviderRepo) GetProviderConfig(ctx context.Context, id string) (domain.ProviderConfig, error) {
	if f.getErr != nil {
		return domain.ProviderConfig{}, f.getErr
	}
	return f.memProviderConfigRepo.GetProviderConfig(ctx, id)
}

func (f *failingProviderRepo) DeleteProviderConfig(ctx context.Context, id string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	return f.memProviderConfigRepo.DeleteProviderConfig(ctx, id)
}

func (f *failingProviderRepo) ListProviderConfigs(ctx context.Context) ([]domain.ProviderConfig, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.nilList {
		return nil, nil
	}
	return f.memProviderConfigRepo.ListProviderConfigs(ctx)
}

// namedFactory builds a live mock provider named after the config.
func namedFactory(cfg domain.ProviderConfig) (ports.LLMClient, error) {
	return &mockLLMClient{alive: true, name: cfg.Name}, nil
}

func newProviderOrch(t *testing.T, repo ports.ProviderConfigRepository, factory func(domain.ProviderConfig) (ports.LLMClient, error)) (*services.OrchestratorService, *services.DiscoveryService) {
	t.Helper()
	disc := services.NewDiscoveryService()
	orch := services.NewOrchestrator(disc, newMemRepo(), &noopWriter{}, nil, services.WithDisableBackgroundWorkers())
	t.Cleanup(orch.Stop)
	if repo != nil {
		orch.WithProviderConfigRepo(repo)
	}
	if factory != nil {
		orch.WithProviderFactory(factory)
	}
	return orch, disc
}

func TestUpdateProviderConfig_RenamesAdapterAndKeepsCreatedAt(t *testing.T) {
	repo := newMemProviderConfigRepo()
	orch, disc := newProviderOrch(t, repo, namedFactory)

	added, err := orch.AddProviderConfig(bg, domain.ProviderConfig{Name: "old", Kind: domain.ProviderKindOllama, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := disc.GetClientByName("old"); !ok {
		t.Fatal("adapter for 'old' should be registered")
	}

	updated, err := orch.UpdateProviderConfig(bg, domain.ProviderConfig{ID: added.ID, Name: "new", Kind: domain.ProviderKindOllama, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if !updated.CreatedAt.Equal(added.CreatedAt) || !updated.UpdatedAt.After(added.UpdatedAt) && !updated.UpdatedAt.Equal(added.UpdatedAt) {
		t.Errorf("timestamps: created %v -> %v, updated %v -> %v", added.CreatedAt, updated.CreatedAt, added.UpdatedAt, updated.UpdatedAt)
	}
	if _, ok := disc.GetClientByName("old"); ok {
		t.Error("old adapter must be deregistered after rename")
	}
	if _, ok := disc.GetClientByName("new"); !ok {
		t.Error("renamed adapter must be registered")
	}

	// Disabling removes the adapter and does not rebuild it.
	if _, err := orch.UpdateProviderConfig(bg, domain.ProviderConfig{ID: added.ID, Name: "new", Enabled: false}); err != nil {
		t.Fatal(err)
	}
	if _, ok := disc.GetClientByName("new"); ok {
		t.Error("disabled provider must not stay registered")
	}
}

func TestUpdateProviderConfig_Errors(t *testing.T) {
	// No repo.
	orch, _ := newProviderOrch(t, nil, nil)
	if _, err := orch.UpdateProviderConfig(bg, domain.ProviderConfig{ID: "x"}); err == nil || !strings.Contains(err.Error(), "no config repo") {
		t.Errorf("no repo: %v", err)
	}

	repo := &failingProviderRepo{memProviderConfigRepo: newMemProviderConfigRepo()}
	orch, _ = newProviderOrch(t, repo, namedFactory)
	if _, err := orch.UpdateProviderConfig(bg, domain.ProviderConfig{ID: "ghost"}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("unknown id: %v", err)
	}

	added, _ := orch.AddProviderConfig(bg, domain.ProviderConfig{Name: "p", Enabled: true})
	repo.saveErr = errInjected
	if _, err := orch.UpdateProviderConfig(bg, domain.ProviderConfig{ID: added.ID, Name: "p"}); !errors.Is(err, errInjected) {
		t.Errorf("save failure: %v", err)
	}
	repo.saveErr = nil

	orch.WithProviderFactory(func(domain.ProviderConfig) (ports.LLMClient, error) { return nil, errInjected })
	if _, err := orch.UpdateProviderConfig(bg, domain.ProviderConfig{ID: added.ID, Name: "p", Enabled: true}); !errors.Is(err, errInjected) {
		t.Errorf("factory failure: %v", err)
	}
}

func TestAddProviderConfig_Errors(t *testing.T) {
	repo := newMemProviderConfigRepo()
	orch, _ := newProviderOrch(t, repo, func(domain.ProviderConfig) (ports.LLMClient, error) { return nil, errInjected })
	if _, err := orch.AddProviderConfig(bg, domain.ProviderConfig{Name: "p", Enabled: true}); !errors.Is(err, errInjected) {
		t.Errorf("factory failure: %v", err)
	}
	repo.saveErr = errInjected
	if _, err := orch.AddProviderConfig(bg, domain.ProviderConfig{Name: "p"}); !errors.Is(err, errInjected) {
		t.Errorf("save failure: %v", err)
	}
}

func TestRemoveProviderConfig_Branches(t *testing.T) {
	orch, _ := newProviderOrch(t, nil, nil)
	if err := orch.RemoveProviderConfig(bg, "x"); err == nil || !strings.Contains(err.Error(), "no config repo") {
		t.Errorf("no repo: %v", err)
	}

	repo := &failingProviderRepo{memProviderConfigRepo: newMemProviderConfigRepo()}
	orch, disc := newProviderOrch(t, repo, namedFactory)
	added, _ := orch.AddProviderConfig(bg, domain.ProviderConfig{Name: "p", Enabled: true})

	if err := orch.RemoveProviderConfig(bg, "ghost"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("unknown id: %v", err)
	}
	repo.deleteErr = errInjected
	if err := orch.RemoveProviderConfig(bg, added.ID); !errors.Is(err, errInjected) {
		t.Errorf("delete failure: %v", err)
	}
	repo.deleteErr = nil
	if err := orch.RemoveProviderConfig(bg, added.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := disc.GetClientByName("p"); ok {
		t.Error("adapter must be deregistered with its config")
	}
}

func TestListProviderConfigs_Branches(t *testing.T) {
	orch, _ := newProviderOrch(t, nil, nil)
	if got, err := orch.ListProviderConfigs(bg); err != nil || got == nil || len(got) != 0 {
		t.Errorf("no repo: %v %v", got, err)
	}

	repo := &failingProviderRepo{memProviderConfigRepo: newMemProviderConfigRepo(), nilList: true}
	orch, _ = newProviderOrch(t, repo, nil)
	if got, err := orch.ListProviderConfigs(bg); err != nil || got == nil || len(got) != 0 {
		t.Errorf("nil from repo must become empty slice: %v %v", got, err)
	}
	repo.nilList = false
	repo.listErr = errInjected
	if _, err := orch.ListProviderConfigs(bg); !errors.Is(err, errInjected) {
		t.Errorf("list failure: %v", err)
	}
	repo.listErr = nil
	_ = repo.SaveProviderConfig(bg, domain.ProviderConfig{ID: "a", Name: "a"})
	if got, err := orch.ListProviderConfigs(bg); err != nil || len(got) != 1 {
		t.Errorf("populated: %v %v", got, err)
	}
}

// erroringModelsLLM returns an error from GetAvailableModels.
type erroringModelsLLM struct{ mockLLMClient }

func (e *erroringModelsLLM) GetAvailableModels() ([]string, error) { return nil, errInjected }

func TestGetProviderModels_PropagatesAdapterError(t *testing.T) {
	disc := services.NewDiscoveryService(&erroringModelsLLM{mockLLMClient{alive: true, name: "broken"}})
	orch := services.NewOrchestrator(disc, newMemRepo(), &noopWriter{}, nil, services.WithDisableBackgroundWorkers())
	defer orch.Stop()
	if _, err := orch.GetProviderModels("broken"); !errors.Is(err, errInjected) {
		t.Errorf("got %v", err)
	}
}

func TestGetDiscoveredProviders_ReturnsCopyOfLastScan(t *testing.T) {
	orch, _ := newProviderOrch(t, nil, nil)
	orch.WithSystemScanner(&mockScanner{results: []domain.DiscoveredProvider{{ID: "d1", Name: "found", Status: domain.DiscoveryStatusReachable}}})
	if _, err := orch.TriggerScan(bg); err != nil {
		t.Fatal(err)
	}
	got, err := orch.GetDiscoveredProviders()
	if err != nil || len(got) != 1 || got[0].ID != "d1" {
		t.Fatalf("got %v %v", got, err)
	}
	got[0].ID = "mutated"
	again, _ := orch.GetDiscoveredProviders()
	if again[0].ID != "d1" {
		t.Error("GetDiscoveredProviders must return a defensive copy")
	}
}
