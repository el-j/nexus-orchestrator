package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/core/domain"
)

type policyOrchestrator struct {
	mockOrchestrator
	policy  domain.ProjectPolicy
	found   bool
	err     error
	project string
	set     *domain.ProjectPolicy
}

func (p *policyOrchestrator) GetProjectPolicy(_ context.Context, project string) (domain.ProjectPolicy, bool, error) {
	p.project = project
	return p.policy, p.found, p.err
}

func (p *policyOrchestrator) SetProjectPolicy(_ context.Context, project string, pol domain.ProjectPolicy) error {
	p.project, p.set = project, &pol
	return p.err
}

func putPolicy(t *testing.T, url, body string) (int, map[string]any) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPut, url, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func TestPolicyEndpoint_GetAndPut(t *testing.T) {
	o := &policyOrchestrator{found: true, policy: domain.ProjectPolicy{QuotaThreshold: 0.8, Budgets: map[string]domain.ProviderBudget{"Claude": {DailyTokens: 5}}}}
	ts := httptest.NewServer(httpapi.NewServer(o, nil, nil).Handler())
	defer ts.Close()

	code, body := getJSON(t, ts.URL+"/api/policy?projectPath=/p")
	if code != http.StatusOK || o.project != "/p" || body["found"] != true {
		t.Fatalf("get: %d %v", code, body)
	}
	pol := body["policy"].(map[string]any)
	if pol["quotaThreshold"] != 0.8 {
		t.Errorf("policy: %v", pol)
	}
	o.found = false
	if _, body := getJSON(t, ts.URL+"/api/policy?projectPath=/p"); body["found"] != false {
		t.Errorf("no file yet: %v", body)
	}
	if code, _ := getJSON(t, ts.URL+"/api/policy"); code != http.StatusBadRequest {
		t.Errorf("projectPath is required: %d", code)
	}

	code, _ = putPolicy(t, ts.URL+"/api/policy?projectPath=/p", `{"quotaThreshold":0.7,"ladder":{"coder":["Ollama/*"]}}`)
	if code != http.StatusOK || o.set == nil || o.set.QuotaThreshold != 0.7 || o.set.Ladder["coder"][0] != "Ollama/*" {
		t.Fatalf("put: %d %+v", code, o.set)
	}
	for name, tc := range map[string]struct{ url, body string }{
		"no project":    {"/api/policy", `{}`},
		"not json":      {"/api/policy?projectPath=/p", `nope`},
		"unknown field": {"/api/policy?projectPath=/p", `{"nonsense":1}`},
	} {
		if code, out := putPolicy(t, ts.URL+tc.url, tc.body); code != http.StatusBadRequest || out["error"] == nil {
			t.Errorf("%s: %d %v", name, code, out)
		}
	}
	o.err = errors.New("policy: quotaThreshold must be between 0 and 1")
	if code, out := putPolicy(t, ts.URL+"/api/policy?projectPath=/p", `{}`); code != http.StatusBadRequest || !strings.Contains(out["error"].(string), "quotaThreshold") {
		t.Errorf("invalid: %d %v", code, out)
	}
	if code, body := getJSON(t, ts.URL+"/api/policy?projectPath=/p"); code != http.StatusBadRequest || body["error"] == nil {
		t.Errorf("a broken file is reported: %d %v", code, body)
	}

	plain := httptest.NewServer(httpapi.NewServer(&mockOrchestrator{}, nil, nil).Handler())
	defer plain.Close()
	if code, _ := getJSON(t, plain.URL+"/api/policy?projectPath=/p"); code != http.StatusServiceUnavailable {
		t.Errorf("no policy manager: %d", code)
	}
	if code, _ := putPolicy(t, plain.URL+"/api/policy?projectPath=/p", `{}`); code != http.StatusServiceUnavailable {
		t.Errorf("no policy manager: %d", code)
	}
}
