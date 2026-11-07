package httpapi_client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nexus-orchestrator/internal/adapters/outbound/httpapi_client"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

func TestClient_AddProviderConfig(t *testing.T) {
	cfg := domain.ProviderConfig{ID: "prov-1", Name: "OpenAI", Kind: "openai"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/providers/config" {
			http.Error(w, "wrong path", http.StatusNotFound)
			return
		}
		assertContentType(t, r)
		var received domain.ProviderConfig
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		if received.Name != cfg.Name {
			t.Errorf("expected name %q, got %q", cfg.Name, received.Name)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(cfg) //nolint:errcheck
	}))
	defer srv.Close()
	c := httpapi_client.NewClient(srv.URL)
	created, err := c.AddProviderConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("AddProviderConfig: %v", err)
	}
	if created.ID != cfg.ID {
		t.Errorf("expected ID %q, got %q", cfg.ID, created.ID)
	}
}

func TestClient_RemoveProviderConfig(t *testing.T) {
	const id = "prov-delete-1"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/providers/config/"+id {
			http.Error(w, "wrong path", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	c := httpapi_client.NewClient(srv.URL)
	if err := c.RemoveProviderConfig(context.Background(), id); err != nil {
		t.Fatalf("RemoveProviderConfig: %v", err)
	}
}

func TestClient_GetDiscoveredProviders_OK(t *testing.T) {
	want := []domain.DiscoveredProvider{
		{ID: "dp-1", Name: "Ollama", Kind: domain.ProviderKindOllama},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/providers/discovered" {
			http.Error(w, "wrong route", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want) //nolint:errcheck
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	got, err := c.GetDiscoveredProviders()
	if err != nil {
		t.Fatalf("GetDiscoveredProviders: %v", err)
	}
	if len(got) != 1 || got[0].ID != want[0].ID {
		t.Errorf("unexpected providers: %+v", got)
	}

	// non-200 returns error
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv2.Close()
	if _, err := httpapi_client.NewClient(srv2.URL).GetDiscoveredProviders(); err == nil {
		t.Error("expected error on non-200, got nil")
	}
}

func TestClient_TriggerScan_OK(t *testing.T) {
	want := []domain.DiscoveredProvider{
		{ID: "dp-scan-1", Name: "LM Studio", Kind: domain.ProviderKindLMStudio},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/providers/discovered/scan" {
			http.Error(w, "wrong route", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want) //nolint:errcheck
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	got, err := c.TriggerScan(context.Background())
	if err != nil {
		t.Fatalf("TriggerScan: %v", err)
	}
	if len(got) != 1 || got[0].ID != want[0].ID {
		t.Errorf("unexpected providers from scan: %+v", got)
	}

	// non-200/204 returns error
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv2.Close()
	if _, err := httpapi_client.NewClient(srv2.URL).TriggerScan(context.Background()); err == nil {
		t.Error("expected error on non-200, got nil")
	}
}

func TestClient_GetProviders(t *testing.T) {
	want := []ports.ProviderInfo{
		{Name: "openai", Active: true, Models: []string{"gpt-4"}},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/providers" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want) //nolint:errcheck
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	got, err := c.GetProviders()
	if err != nil {
		t.Fatalf("GetProviders: %v", err)
	}
	if len(got) != 1 || got[0].Name != "openai" {
		t.Errorf("unexpected providers: %+v", got)
	}

	// non-200 returns error
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv2.Close()
	if _, err := httpapi_client.NewClient(srv2.URL).GetProviders(); err == nil {
		t.Error("expected error on non-200, got nil")
	}
}

func TestClient_RegisterCloudProvider(t *testing.T) {
	cfg := domain.ProviderConfig{ID: "p1", Name: "anthropic", Kind: "anthropic"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/providers" {
			http.NotFound(w, r)
			return
		}
		assertContentType(t, r)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	if err := c.RegisterCloudProvider(cfg); err != nil {
		t.Fatalf("RegisterCloudProvider: %v", err)
	}

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv2.Close()
	if err := httpapi_client.NewClient(srv2.URL).RegisterCloudProvider(cfg); err == nil {
		t.Error("expected error on non-201, got nil")
	}
}

func TestClient_RemoveProvider(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.NotFound(w, r)
			return
		}
		switch r.URL.Path {
		case "/api/providers/openai":
			w.WriteHeader(http.StatusNoContent)
		case "/api/providers/unknown":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	if err := c.RemoveProvider("openai"); err != nil {
		t.Fatalf("RemoveProvider: %v", err)
	}
	if err := c.RemoveProvider("unknown"); err == nil {
		t.Error("expected not-found error, got nil")
	}
	if err := c.RemoveProvider("other"); err == nil {
		t.Error("expected error on 500, got nil")
	}
}

func TestClient_GetProviderModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		switch r.URL.Path {
		case "/api/providers/openai/models":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]string{"gpt-4o", "gpt-4o-mini"}) //nolint:errcheck
		case "/api/providers/unknown/models":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	models, err := c.GetProviderModels("openai")
	if err != nil {
		t.Fatalf("GetProviderModels: %v", err)
	}
	if len(models) != 2 || models[0] != "gpt-4o" {
		t.Errorf("unexpected models: %+v", models)
	}
	if _, err := c.GetProviderModels("unknown"); err == nil {
		t.Error("expected not-found error, got nil")
	}
	if _, err := c.GetProviderModels("error-case"); err == nil {
		t.Error("expected error on 500, got nil")
	}
}

func TestClient_UpdateProviderConfig(t *testing.T) {
	cfg := domain.ProviderConfig{ID: "cfg-1", Name: "updated"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.NotFound(w, r)
			return
		}
		switch r.URL.Path {
		case "/api/providers/config/cfg-1":
			assertContentType(t, r)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(cfg) //nolint:errcheck
		case "/api/providers/config/notfound":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	got, err := c.UpdateProviderConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("UpdateProviderConfig: %v", err)
	}
	if got.ID != cfg.ID || got.Name != cfg.Name {
		t.Errorf("unexpected updated cfg: %+v", got)
	}

	if _, err := c.UpdateProviderConfig(context.Background(), domain.ProviderConfig{ID: "notfound"}); err == nil {
		t.Error("expected not-found error, got nil")
	}
	if _, err := c.UpdateProviderConfig(context.Background(), domain.ProviderConfig{ID: "other"}); err == nil {
		t.Error("expected error on 500, got nil")
	}
}

func TestClient_ListProviderConfigs(t *testing.T) {
	want := []domain.ProviderConfig{{ID: "c1", Name: "prov1"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/providers/config" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want) //nolint:errcheck
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	got, err := c.ListProviderConfigs(context.Background())
	if err != nil {
		t.Fatalf("ListProviderConfigs: %v", err)
	}
	if len(got) != 1 || got[0].ID != "c1" {
		t.Errorf("unexpected configs: %+v", got)
	}

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv2.Close()
	if _, err := httpapi_client.NewClient(srv2.URL).ListProviderConfigs(context.Background()); err == nil {
		t.Error("expected error on non-200, got nil")
	}
}

func TestClient_PromoteProvider(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/providers/promote/p1" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	if err := c.PromoteProvider(context.Background(), "p1"); err != nil {
		t.Fatalf("PromoteProvider: %v", err)
	}

	if err := c.PromoteProvider(context.Background(), "unknown"); err == nil {
		t.Error("expected error on 404, got nil")
	}
}
