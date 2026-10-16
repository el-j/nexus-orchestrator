package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/core/domain"
)

type routingOrchestrator struct {
	mockOrchestrator
	preview       domain.RoutingPreview
	err           error
	project, role string
}

func (r *routingOrchestrator) PreviewRouting(_ context.Context, project, role string) (domain.RoutingPreview, error) {
	r.project, r.role = project, role
	return r.preview, r.err
}

func TestRoutingEndpoint(t *testing.T) {
	o := &routingOrchestrator{preview: domain.RoutingPreview{Role: "coder", Ladder: []string{"Ollama/*"}, QuotaThreshold: 0.9,
		Order: []domain.RoutingEntry{{Position: 1, Provider: "Ollama", Model: "qwen", Rung: 0}}}}
	ts := httptest.NewServer(httpapi.NewServer(o, nil, nil).Handler())
	defer ts.Close()

	code, body := getJSON(t, ts.URL+"/api/routing?projectPath=/p&role=coder")
	if code != http.StatusOK || o.project != "/p" || o.role != "coder" {
		t.Fatalf("%d %v project=%q role=%q", code, body, o.project, o.role)
	}
	order := body["order"].([]any)
	if len(order) != 1 || order[0].(map[string]any)["provider"] != "Ollama" || body["quotaThreshold"] != 0.9 {
		t.Errorf("body: %v", body)
	}
	if code, _ := getJSON(t, ts.URL+"/api/routing"); code != http.StatusBadRequest {
		t.Errorf("projectPath is required: %d", code)
	}
	o.err = errors.New("discovery: no active provider available")
	if code, body := getJSON(t, ts.URL+"/api/routing?projectPath=/p"); code != http.StatusServiceUnavailable || body["error"] == nil {
		t.Errorf("no provider: %d %v", code, body)
	}
	plain := httptest.NewServer(httpapi.NewServer(&mockOrchestrator{}, nil, nil).Handler())
	defer plain.Close()
	if code, _ := getJSON(t, plain.URL+"/api/routing?projectPath=/p"); code != http.StatusServiceUnavailable {
		t.Errorf("no previewer: %d", code)
	}
}
