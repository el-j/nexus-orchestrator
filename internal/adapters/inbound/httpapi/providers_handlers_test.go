package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// serveWithOrch creates a Server backed by orch (no brain, no hub) and serves
// the request directly into the recorder — no real TCP server needed.
func serveWithOrch(orch *mockOrchestrator, req *http.Request, rec *httptest.ResponseRecorder) {
	srv := httpapi.NewServer(orch, nil, nil)
	srv.Handler().ServeHTTP(rec, req)
}

// ---------------------------------------------------------------------------
// GET /api/providers
// ---------------------------------------------------------------------------

func TestHandleGetProviders_OK(t *testing.T) {
	providers := []ports.ProviderInfo{
		{Name: "ollama", Active: true, ActiveModel: "llama3"},
		{Name: "lmstudio", Active: false},
	}
	mock := &mockOrchestrator{getProvidersResult: providers}
	req := httptest.NewRequest(http.MethodGet, "/api/providers", nil)
	rec := httptest.NewRecorder()

	serveWithOrch(mock, req, rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
	var got []ports.ProviderInfo
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(got))
	}
	if got[0].Name != "ollama" {
		t.Errorf("expected first provider %q, got %q", "ollama", got[0].Name)
	}
}

func TestHandleGetProviders_Empty(t *testing.T) {
	mock := &mockOrchestrator{getProvidersResult: nil}
	req := httptest.NewRequest(http.MethodGet, "/api/providers", nil)
	rec := httptest.NewRecorder()

	serveWithOrch(mock, req, rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	// nil slice should be normalised to an empty JSON array
	body := rec.Body.String()
	if body == "null\n" || body == "null" {
		t.Errorf("expected JSON array, got %q", body)
	}
}

// ---------------------------------------------------------------------------
// POST /api/providers  (register provider)
// ---------------------------------------------------------------------------

func TestHandleRegisterProvider_OK(t *testing.T) {
	mock := &mockOrchestrator{}
	payload, _ := json.Marshal(domain.ProviderConfig{
		Name: "my-ollama",
		Kind: domain.ProviderKindOllama,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/providers", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	serveWithOrch(mock, req, rec)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["name"] != "my-ollama" {
		t.Errorf("expected name %q, got %q", "my-ollama", resp["name"])
	}
	if resp["kind"] != string(domain.ProviderKindOllama) {
		t.Errorf("expected kind %q, got %q", domain.ProviderKindOllama, resp["kind"])
	}
}

func TestHandleRegisterProvider_MissingFields(t *testing.T) {
	mock := &mockOrchestrator{}
	// name is present but kind is missing
	payload, _ := json.Marshal(map[string]string{"name": "only-name"})
	req := httptest.NewRequest(http.MethodPost, "/api/providers", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	serveWithOrch(mock, req, rec)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// DELETE /api/providers/{name}
// ---------------------------------------------------------------------------

func TestHandleRemoveProvider_OK(t *testing.T) {
	mock := &mockOrchestrator{}
	req := httptest.NewRequest(http.MethodDelete, "/api/providers/ollama", nil)
	rec := httptest.NewRecorder()

	serveWithOrch(mock, req, rec)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleRemoveProvider_NotFound(t *testing.T) {
	mock := &mockOrchestrator{removeProviderErr: domain.ErrNotFound}
	req := httptest.NewRequest(http.MethodDelete, "/api/providers/ghost", nil)
	rec := httptest.NewRecorder()

	serveWithOrch(mock, req, rec)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// GET /api/providers/{name}/models
// ---------------------------------------------------------------------------

func TestHandleGetProviderModels_OK(t *testing.T) {
	models := []string{"llama3", "mistral"}
	mock := &mockOrchestrator{getProviderModelsResult: models}
	req := httptest.NewRequest(http.MethodGet, "/api/providers/ollama/models", nil)
	rec := httptest.NewRecorder()

	serveWithOrch(mock, req, rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
	var got []string
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 models, got %d", len(got))
	}
}

func TestHandleGetProviderModels_NotFound(t *testing.T) {
	mock := &mockOrchestrator{getProviderModelsErr: domain.ErrNotFound}
	req := httptest.NewRequest(http.MethodGet, "/api/providers/ghost/models", nil)
	rec := httptest.NewRecorder()

	serveWithOrch(mock, req, rec)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// POST /api/providers/config  (add provider config)
// ---------------------------------------------------------------------------

func TestHandleAddProviderConfig_OK(t *testing.T) {
	mock := &mockOrchestrator{}
	payload, _ := json.Marshal(domain.ProviderConfig{
		Name:   "cloud-anthropic",
		Kind:   domain.ProviderKindAnthropic,
		APIKey: "sk-testtokenXYZ",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/providers/config", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	serveWithOrch(mock, req, rec)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rec.Code, rec.Body.String())
	}
	var got domain.ProviderConfig
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Name != "cloud-anthropic" {
		t.Errorf("expected name %q, got %q", "cloud-anthropic", got.Name)
	}
	// API key must be masked in the response
	if got.APIKey == "sk-testtokenXYZ" {
		t.Errorf("APIKey should be masked, got plain-text value")
	}
}

func TestHandleAddProviderConfig_MissingFields(t *testing.T) {
	mock := &mockOrchestrator{}
	payload := []byte(`{"name":"no-kind"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/providers/config", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	serveWithOrch(mock, req, rec)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// GET /api/providers/config  (list provider configs)
// ---------------------------------------------------------------------------

func TestHandleListProviderConfigs_OK(t *testing.T) {
	mock := &mockOrchestrator{}
	req := httptest.NewRequest(http.MethodGet, "/api/providers/config", nil)
	rec := httptest.NewRecorder()

	serveWithOrch(mock, req, rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
	// The mock returns nil which the handler converts to an empty masked slice.
	var got []domain.ProviderConfig
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

// ---------------------------------------------------------------------------
// PUT /api/providers/config/{id}  (update provider config)
// ---------------------------------------------------------------------------

func TestHandleUpdateProviderConfig_OK(t *testing.T) {
	mock := &mockOrchestrator{}
	payload, _ := json.Marshal(domain.ProviderConfig{
		Name: "updated-name",
		Kind: domain.ProviderKindOllama,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/providers/config/cfg-001", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	serveWithOrch(mock, req, rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
	var got domain.ProviderConfig
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	// The ID in the URL should be set on the returned config
	if got.ID != "cfg-001" {
		t.Errorf("expected ID %q, got %q", "cfg-001", got.ID)
	}
}

func TestHandleUpdateProviderConfig_InvalidJSON(t *testing.T) {
	mock := &mockOrchestrator{}
	req := httptest.NewRequest(http.MethodPut, "/api/providers/config/cfg-001", bytes.NewBufferString(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	serveWithOrch(mock, req, rec)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// DELETE /api/providers/config/{id}  (remove provider config)
// ---------------------------------------------------------------------------

func TestHandleRemoveProviderConfig_OK(t *testing.T) {
	mock := &mockOrchestrator{}
	req := httptest.NewRequest(http.MethodDelete, "/api/providers/config/cfg-001", nil)
	rec := httptest.NewRecorder()

	serveWithOrch(mock, req, rec)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleRemoveProviderConfig_NotFound(t *testing.T) {
	mock := &mockOrchestrator{removeProviderConfigErr: domain.ErrNotFound}
	req := httptest.NewRequest(http.MethodDelete, "/api/providers/config/ghost-cfg", nil)
	rec := httptest.NewRecorder()

	serveWithOrch(mock, req, rec)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// GET /api/providers/discovered
// ---------------------------------------------------------------------------

func TestHandleGetDiscoveredProviders_OK(t *testing.T) {
	discovered := []domain.DiscoveredProvider{
		{ID: "d1", Name: "Ollama", Kind: domain.ProviderKindOllama},
	}
	mock := &mockOrchestrator{discoveredProvidersResult: discovered}
	req := httptest.NewRequest(http.MethodGet, "/api/providers/discovered", nil)
	rec := httptest.NewRecorder()

	serveWithOrch(mock, req, rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
	var got []domain.DiscoveredProvider
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 discovered provider, got %d", len(got))
	}
}

// ---------------------------------------------------------------------------
// POST /api/providers/discovered/scan  (trigger scan)
// ---------------------------------------------------------------------------

func TestHandleTriggerScan_OK(t *testing.T) {
	mock := &mockOrchestrator{}
	req := httptest.NewRequest(http.MethodPost, "/api/providers/discovered/scan", nil)
	rec := httptest.NewRecorder()

	serveWithOrch(mock, req, rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// POST /api/providers/promote/{id}  (promote provider)
// ---------------------------------------------------------------------------

func TestHandlePromoteProvider_OK(t *testing.T) {
	mock := &mockOrchestrator{}
	req := httptest.NewRequest(http.MethodPost, "/api/providers/promote/d1", nil)
	rec := httptest.NewRecorder()

	serveWithOrch(mock, req, rec)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlePromoteProvider_NotFound(t *testing.T) {
	mock := &mockOrchestrator{promoteProviderErr: domain.ErrNotFound}
	req := httptest.NewRequest(http.MethodPost, "/api/providers/promote/ghost", nil)
	rec := httptest.NewRecorder()

	serveWithOrch(mock, req, rec)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
