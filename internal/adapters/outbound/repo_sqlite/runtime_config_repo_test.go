package repo_sqlite_test

import (
	"context"
	"testing"
	"time"

	repo_sqlite "nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
)

func newRuntimeConfigRepo(t *testing.T) (*repo_sqlite.Repository, *repo_sqlite.RuntimeConfigRepo) {
	t.Helper()
	r := newTestRepo(t)
	return r, repo_sqlite.NewRuntimeConfigRepo(r)
}

func TestGetRuntimeConfig_Default(t *testing.T) {
	_, repo := newRuntimeConfigRepo(t)
	ctx := context.Background()

	cfg, err := repo.GetRuntimeConfig(ctx)
	if err != nil {
		t.Fatalf("expected no error on empty DB, got: %v", err)
	}
	if cfg.QueueCap < 0 {
		t.Errorf("expected non-negative QueueCap, got %d", cfg.QueueCap)
	}
}

func TestSaveRuntimeConfig_RoundTrip(t *testing.T) {
	_, repo := newRuntimeConfigRepo(t)
	ctx := context.Background()

	cfg := domain.RuntimeConfig{
		QueueCap:  42,
		APIToken:  "test-api-token",
		MCPToken:  "test-mcp-token",
		UpdatedAt: time.Now().Truncate(time.Second),
	}

	if err := repo.SaveRuntimeConfig(ctx, cfg); err != nil {
		t.Fatalf("SaveRuntimeConfig: %v", err)
	}

	got, err := repo.GetRuntimeConfig(ctx)
	if err != nil {
		t.Fatalf("GetRuntimeConfig after save: %v", err)
	}
	if got.QueueCap != 42 {
		t.Errorf("QueueCap: want 42, got %d", got.QueueCap)
	}
	if got.APIToken != "test-api-token" {
		t.Errorf("APIToken: want %q, got %q", "test-api-token", got.APIToken)
	}
	if got.MCPToken != "test-mcp-token" {
		t.Errorf("MCPToken: want %q, got %q", "test-mcp-token", got.MCPToken)
	}
}

func TestSaveRuntimeConfig_Overwrite(t *testing.T) {
	_, repo := newRuntimeConfigRepo(t)
	ctx := context.Background()

	first := domain.RuntimeConfig{QueueCap: 10, APIToken: "first"}
	if err := repo.SaveRuntimeConfig(ctx, first); err != nil {
		t.Fatalf("first save: %v", err)
	}

	second := domain.RuntimeConfig{QueueCap: 99, APIToken: "second"}
	if err := repo.SaveRuntimeConfig(ctx, second); err != nil {
		t.Fatalf("second save: %v", err)
	}

	got, err := repo.GetRuntimeConfig(ctx)
	if err != nil {
		t.Fatalf("GetRuntimeConfig: %v", err)
	}
	if got.QueueCap != 99 {
		t.Errorf("QueueCap: want 99, got %d", got.QueueCap)
	}
	if got.APIToken != "second" {
		t.Errorf("APIToken: want %q, got %q", "second", got.APIToken)
	}
}

func TestGetRuntimeConfig_CorruptRow(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	repo := repo_sqlite.NewRuntimeConfigRepo(r)
	ctx := context.Background()

	// Insert a row with corrupt JSON to test error path.
	_, err := r.DB().ExecContext(ctx,
		`INSERT OR REPLACE INTO runtime_config (id, json, updated_at) VALUES (1, '{invalid-json', 0)`)
	if err != nil {
		t.Fatalf("insert corrupt row: %v", err)
	}

	_, err = repo.GetRuntimeConfig(ctx)
	if err == nil {
		t.Fatal("expected error for corrupt JSON, got nil")
	}
}
