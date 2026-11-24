package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

type autoOrch struct {
	*mockOrchestrator
	run      domain.AutoModeRun
	runs     []domain.AutoModeRun
	startErr error
	getErr   error
	stopErr  error
}

func (a *autoOrch) StartAutoMode(_ context.Context, req domain.AutoModeRequest) (domain.AutoModeRun, error) {
	if a.startErr != nil {
		return domain.AutoModeRun{}, a.startErr
	}
	r := a.run
	r.ProjectPath = req.ProjectPath
	r.Goal = req.Goal
	return r, nil
}
func (a *autoOrch) GetAutoModeRun(string) (domain.AutoModeRun, error) {
	return a.run, a.getErr
}
func (a *autoOrch) ListAutoModeRuns(string) []domain.AutoModeRun { return a.runs }
func (a *autoOrch) StopAutoMode(string) error                    { return a.stopErr }

func TestAutoModeHandlers(t *testing.T) {
	// Unavailable when orch does not implement AutoModeManager
	srvNoAuto := httpapi.NewServer(&mockOrchestrator{}, nil, nil)
	tsNo := httptest.NewServer(srvNoAuto.Handler())
	defer tsNo.Close()
	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, "/api/auto/start"},
		{http.MethodGet, "/api/auto"},
		{http.MethodGet, "/api/auto/r1"},
		{http.MethodPost, "/api/auto/r1/stop"},
	} {
		req, _ := http.NewRequest(tc.method, tsNo.URL+tc.path, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("%s %s: status=%d, want 503", tc.method, tc.path, resp.StatusCode)
		}
	}

	orch := &autoOrch{
		mockOrchestrator: &mockOrchestrator{},
		run:              domain.AutoModeRun{ID: "run-1", Phase: domain.AutoPhaseExecuting},
		runs:             nil,
	}
	srv := httpapi.NewServer(orch, nil, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Invalid JSON
	resp, err := http.Post(ts.URL+"/api/auto/start", "application/json", bytes.NewBufferString("{bad"))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad json: got %d", resp.StatusCode)
	}

	// Start success
	resp, err = http.Post(ts.URL+"/api/auto/start", "application/json", bytes.NewBufferString(`{"projectPath":"/p","goal":"g"}`))
	if err != nil {
		t.Fatal(err)
	}
	var started domain.AutoModeRun
	_ = json.NewDecoder(resp.Body).Decode(&started)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated || started.ID != "run-1" {
		t.Fatalf("start: status=%d run=%+v", resp.StatusCode, started)
	}

	// Start conflict & bad request
	orch.startErr = services.ErrAutoModeActive
	resp, err = http.Post(ts.URL+"/api/auto/start", "application/json", bytes.NewBufferString(`{"projectPath":"/p"}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("conflict: got %d", resp.StatusCode)
	}

	orch.startErr = errors.New("missing path")
	resp, err = http.Post(ts.URL+"/api/auto/start", "application/json", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad req: got %d", resp.StatusCode)
	}
	orch.startErr = nil

	// List (nil -> empty array)
	resp, err = http.Get(ts.URL + "/api/auto?projectPath=/p")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: got %d", resp.StatusCode)
	}

	// Get OK & NotFound
	resp, err = http.Get(ts.URL + "/api/auto/run-1")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get: got %d", resp.StatusCode)
	}
	orch.getErr = domain.ErrNotFound
	resp, err = http.Get(ts.URL + "/api/auto/missing")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("get missing: got %d", resp.StatusCode)
	}

	// Stop OK & NotFound
	resp, err = http.Post(ts.URL+"/api/auto/run-1/stop", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stop: got %d", resp.StatusCode)
	}
	orch.stopErr = domain.ErrNotFound
	resp, err = http.Post(ts.URL+"/api/auto/missing/stop", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("stop missing: got %d", resp.StatusCode)
	}
}
