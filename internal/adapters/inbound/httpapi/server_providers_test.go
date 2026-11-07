package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/core/domain"
)

// TestHTTP_GetDiscoveredProviders_Returns200 verifies that GET /api/providers/discovered
// returns 200 with application/json content type.
func TestHTTP_GetDiscoveredProviders_Returns200(t *testing.T) {
	mock := &mockOrchestrator{
		discoveredProvidersResult: []domain.DiscoveredProvider{},
	}
	srv := httpapi.NewServer(mock, nil, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/providers/discovered")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %q", ct)
	}
}

// TestHTTP_TriggerScan_Returns200 verifies that POST /api/providers/discovered/scan
// returns 200 when the scan succeeds.
func TestHTTP_TriggerScan_Returns200(t *testing.T) {
	mock := &mockOrchestrator{}
	srv := httpapi.NewServer(mock, nil, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/api/providers/discovered/scan", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// TestHTTP_PromoteProvider_NotFound_Returns404 verifies that
// POST /api/providers/promote/{id} returns 404 when the orchestrator
// returns domain.ErrNotFound for the given provider ID.
func TestHTTP_PromoteProvider_NotFound_Returns404(t *testing.T) {
	mock := &mockOrchestrator{promoteProviderErr: domain.ErrNotFound}
	srv := httpapi.NewServer(mock, nil, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/api/providers/promote/bad-id", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}
