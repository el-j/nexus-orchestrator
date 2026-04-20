package repo_sqlite_test

import (
	"errors"
	"testing"
	"time"

	repo_sqlite "nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
)

func newModelCapabilityRepo(t *testing.T) *repo_sqlite.ModelCapabilityRepo {
	t.Helper()
	r := newTestRepo(t)
	t.Cleanup(func() { _ = r.Close() })
	return repo_sqlite.NewModelCapabilityRepo(r)
}

func TestSaveModelCapability_RoundTrip(t *testing.T) {
	repo := newModelCapabilityRepo(t)

	now := time.Now().Truncate(time.Second).UTC()
	p := domain.ModelCapabilityProfile{
		ModelID:              "test-model-1",
		ContextWindow:        65536,
		RecommendedMaxOutput: 4096,
		Notes:                "test notes",
		BuiltIn:              true,
		CreatedAt:            now,
	}

	if err := repo.Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetByModelID("test-model-1")
	if err != nil {
		t.Fatalf("GetByModelID: %v", err)
	}

	if got.ModelID != p.ModelID {
		t.Errorf("ModelID: got %q, want %q", got.ModelID, p.ModelID)
	}
	if got.ContextWindow != p.ContextWindow {
		t.Errorf("ContextWindow: got %d, want %d", got.ContextWindow, p.ContextWindow)
	}
	if got.RecommendedMaxOutput != p.RecommendedMaxOutput {
		t.Errorf("RecommendedMaxOutput: got %d, want %d", got.RecommendedMaxOutput, p.RecommendedMaxOutput)
	}
	if got.Notes != p.Notes {
		t.Errorf("Notes: got %q, want %q", got.Notes, p.Notes)
	}
	if got.BuiltIn != p.BuiltIn {
		t.Errorf("BuiltIn: got %v, want %v", got.BuiltIn, p.BuiltIn)
	}
}

func TestGetByModelID_NotFound(t *testing.T) {
	repo := newModelCapabilityRepo(t)

	_, err := repo.GetByModelID("nonexistent-model")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected domain.ErrNotFound, got %v", err)
	}
}

func TestGetAll_Empty(t *testing.T) {
	repo := newModelCapabilityRepo(t)

	got, err := repo.GetAll()
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if got == nil {
		t.Fatal("GetAll: expected non-nil slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("GetAll: expected 0 profiles, got %d", len(got))
	}
}

func TestGetAll_MultipleModels(t *testing.T) {
	repo := newModelCapabilityRepo(t)

	profiles := []domain.ModelCapabilityProfile{
		{ModelID: "model-alpha", ContextWindow: 8192, RecommendedMaxOutput: 1024, Notes: "alpha", BuiltIn: false},
		{ModelID: "model-beta", ContextWindow: 32768, RecommendedMaxOutput: 4096, Notes: "beta", BuiltIn: true},
		{ModelID: "model-gamma", ContextWindow: 131072, RecommendedMaxOutput: 8192, Notes: "gamma", BuiltIn: false},
	}

	for _, p := range profiles {
		if err := repo.Save(p); err != nil {
			t.Fatalf("Save %q: %v", p.ModelID, err)
		}
	}

	got, err := repo.GetAll()
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(got) != len(profiles) {
		t.Fatalf("GetAll: expected %d profiles, got %d", len(profiles), len(got))
	}

	// GetAll is ordered by model_id; our names sort alpha < beta < gamma
	wantOrder := []string{"model-alpha", "model-beta", "model-gamma"}
	for i, want := range wantOrder {
		if got[i].ModelID != want {
			t.Errorf("got[%d].ModelID: want %q, got %q", i, want, got[i].ModelID)
		}
	}
}

func TestDeleteModelCapability(t *testing.T) {
	repo := newModelCapabilityRepo(t)

	p := domain.ModelCapabilityProfile{
		ModelID:       "delete-me",
		ContextWindow: 4096,
	}
	if err := repo.Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := repo.Delete("delete-me"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	all, err := repo.GetAll()
	if err != nil {
		t.Fatalf("GetAll after delete: %v", err)
	}
	for _, got := range all {
		if got.ModelID == "delete-me" {
			t.Error("deleted model still returned by GetAll")
		}
	}

	_, err = repo.GetByModelID("delete-me")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetByModelID after delete: expected domain.ErrNotFound, got %v", err)
	}
}

func TestDeleteModelCapability_NotFound(t *testing.T) {
	repo := newModelCapabilityRepo(t)

	err := repo.Delete("ghost-model")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected domain.ErrNotFound, got %v", err)
	}
}

func TestSaveModelCapability_Update(t *testing.T) {
	repo := newModelCapabilityRepo(t)

	original := domain.ModelCapabilityProfile{
		ModelID:              "upsert-model",
		ContextWindow:        8192,
		RecommendedMaxOutput: 1024,
		Notes:                "original notes",
		BuiltIn:              false,
	}
	if err := repo.Save(original); err != nil {
		t.Fatalf("first Save: %v", err)
	}

	updated := domain.ModelCapabilityProfile{
		ModelID:              "upsert-model",
		ContextWindow:        32768,
		RecommendedMaxOutput: 4096,
		Notes:                "updated notes",
		BuiltIn:              true,
	}
	if err := repo.Save(updated); err != nil {
		t.Fatalf("second Save (upsert): %v", err)
	}

	got, err := repo.GetByModelID("upsert-model")
	if err != nil {
		t.Fatalf("GetByModelID: %v", err)
	}
	if got.ContextWindow != updated.ContextWindow {
		t.Errorf("ContextWindow: got %d, want %d", got.ContextWindow, updated.ContextWindow)
	}
	if got.RecommendedMaxOutput != updated.RecommendedMaxOutput {
		t.Errorf("RecommendedMaxOutput: got %d, want %d", got.RecommendedMaxOutput, updated.RecommendedMaxOutput)
	}
	if got.Notes != updated.Notes {
		t.Errorf("Notes: got %q, want %q", got.Notes, updated.Notes)
	}
	if got.BuiltIn != updated.BuiltIn {
		t.Errorf("BuiltIn: got %v, want %v", got.BuiltIn, updated.BuiltIn)
	}

	// Ensure only one record exists (upsert, not insert)
	all, err := repo.GetAll()
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("expected 1 profile after upsert, got %d", len(all))
	}
}
