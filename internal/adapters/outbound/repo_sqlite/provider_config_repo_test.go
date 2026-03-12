package repo_sqlite_test

import (
	"context"
	"errors"
	"testing"

	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
)

func TestProviderConfigRepo_Save_NewConfig(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	pr := repo_sqlite.NewProviderConfigRepo(r)
	ctx := context.Background()
	cfg := domain.ProviderConfig{
		Name:    "local-lm",
		Kind:    domain.ProviderKindLMStudio,
		BaseURL: "http://127.0.0.1:1234",
		Enabled: true,
	}
	if err := pr.SaveProviderConfig(ctx, cfg); err != nil {
		t.Fatalf("SaveProviderConfig: %v", err)
	}
	list, err := pr.ListProviderConfigs(ctx)
	if err != nil {
		t.Fatalf("ListProviderConfigs: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 config, got %d", len(list))
	}
	if list[0].ID == "" {
		t.Error("expected ID to be assigned, got empty string")
	}
	if list[0].Name != cfg.Name {
		t.Errorf("Name: got %q, want %q", list[0].Name, cfg.Name)
	}
}

func TestProviderConfigRepo_Save_UpdateExisting(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	pr := repo_sqlite.NewProviderConfigRepo(r)
	ctx := context.Background()
	cfg := domain.ProviderConfig{
		ID:      "fixed-id-001",
		Name:    "original-name",
		Kind:    domain.ProviderKindOllama,
		Enabled: true,
	}
	if err := pr.SaveProviderConfig(ctx, cfg); err != nil {
		t.Fatalf("SaveProviderConfig (initial): %v", err)
	}
	cfg.Name = "updated-name"
	cfg.Enabled = false
	if err := pr.SaveProviderConfig(ctx, cfg); err != nil {
		t.Fatalf("SaveProviderConfig (update): %v", err)
	}
	got, err := pr.GetProviderConfig(ctx, "fixed-id-001")
	if err != nil {
		t.Fatalf("GetProviderConfig after update: %v", err)
	}
	if got.Name != "updated-name" {
		t.Errorf("Name after update: got %q, want %q", got.Name, "updated-name")
	}
	if got.Enabled {
		t.Error("Enabled after update: got true, want false")
	}
	list, err := pr.ListProviderConfigs(ctx)
	if err != nil {
		t.Fatalf("ListProviderConfigs: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 record after upsert, got %d", len(list))
	}
}

func TestProviderConfigRepo_List_Empty(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	pr := repo_sqlite.NewProviderConfigRepo(r)
	ctx := context.Background()
	list, err := pr.ListProviderConfigs(ctx)
	if err != nil {
		t.Fatalf("ListProviderConfigs on empty DB: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 configs, got %d", len(list))
	}
}

func TestProviderConfigRepo_List_Multiple(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	pr := repo_sqlite.NewProviderConfigRepo(r)
	ctx := context.Background()
	configs := []domain.ProviderConfig{
		{ID: "cfg-1", Name: "provider-a", Kind: domain.ProviderKindLMStudio, Enabled: true},
		{ID: "cfg-2", Name: "provider-b", Kind: domain.ProviderKindOllama, Enabled: false},
		{ID: "cfg-3", Name: "provider-c", Kind: domain.ProviderKindOpenAICompat, Enabled: true},
	}
	for _, cfg := range configs {
		if err := pr.SaveProviderConfig(ctx, cfg); err != nil {
			t.Fatalf("SaveProviderConfig %q: %v", cfg.ID, err)
		}
	}
	list, err := pr.ListProviderConfigs(ctx)
	if err != nil {
		t.Fatalf("ListProviderConfigs: %v", err)
	}
	if len(list) != len(configs) {
		t.Fatalf("expected %d configs, got %d", len(configs), len(list))
	}
	ids := make(map[string]bool, len(list))
	for _, c := range list {
		ids[c.ID] = true
	}
	for _, cfg := range configs {
		if !ids[cfg.ID] {
			t.Errorf("ID %q not found in list", cfg.ID)
		}
	}
}

func TestProviderConfigRepo_Get_Found(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	pr := repo_sqlite.NewProviderConfigRepo(r)
	ctx := context.Background()
	cfg := domain.ProviderConfig{
		ID:      "get-test-1",
		Name:    "my-provider",
		Kind:    domain.ProviderKindAnthropic,
		APIKey:  "sk-secret",
		Enabled: true,
	}
	if err := pr.SaveProviderConfig(ctx, cfg); err != nil {
		t.Fatalf("SaveProviderConfig: %v", err)
	}
	got, err := pr.GetProviderConfig(ctx, "get-test-1")
	if err != nil {
		t.Fatalf("GetProviderConfig: %v", err)
	}
	if got.ID != cfg.ID {
		t.Errorf("ID: got %q, want %q", got.ID, cfg.ID)
	}
	if got.Name != cfg.Name {
		t.Errorf("Name: got %q, want %q", got.Name, cfg.Name)
	}
	if got.Kind != cfg.Kind {
		t.Errorf("Kind: got %q, want %q", got.Kind, cfg.Kind)
	}
	if got.APIKey != cfg.APIKey {
		t.Errorf("APIKey: got %q, want %q", got.APIKey, cfg.APIKey)
	}
	if !got.Enabled {
		t.Error("Enabled: got false, want true")
	}
}

func TestProviderConfigRepo_Get_NotFound(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	pr := repo_sqlite.NewProviderConfigRepo(r)
	ctx := context.Background()
	_, err := pr.GetProviderConfig(ctx, "does-not-exist")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected domain.ErrNotFound, got %v", err)
	}
}

func TestProviderConfigRepo_Delete_Existing(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	pr := repo_sqlite.NewProviderConfigRepo(r)
	ctx := context.Background()
	cfg := domain.ProviderConfig{
		ID:   "del-test-1",
		Name: "delete-me",
		Kind: domain.ProviderKindLMStudio,
	}
	if err := pr.SaveProviderConfig(ctx, cfg); err != nil {
		t.Fatalf("SaveProviderConfig: %v", err)
	}
	if err := pr.DeleteProviderConfig(ctx, "del-test-1"); err != nil {
		t.Fatalf("DeleteProviderConfig: %v", err)
	}
	_, err := pr.GetProviderConfig(ctx, "del-test-1")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound after deletion, got %v", err)
	}
}

func TestProviderConfigRepo_Delete_NotFound(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	pr := repo_sqlite.NewProviderConfigRepo(r)
	ctx := context.Background()
	err := pr.DeleteProviderConfig(ctx, "non-existent-id")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected domain.ErrNotFound, got %v", err)
	}
}
