package httpapi_client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nexus-orchestrator/internal/adapters/outbound/httpapi_client"
	"nexus-orchestrator/internal/core/domain"
)

func TestClient_AutoMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/auto/start":
			_ = json.NewEncoder(w).Encode(domain.AutoModeRun{ID: "r1", Phase: domain.AutoPhaseSyncing})
		case r.Method == http.MethodGet && r.URL.Path == "/api/auto/r1":
			_ = json.NewEncoder(w).Encode(domain.AutoModeRun{ID: "r1", Phase: domain.AutoPhaseCompleted})
		case r.Method == http.MethodGet && r.URL.Path == "/api/auto":
			if r.URL.Query().Get("projectPath") == "/err" {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode([]domain.AutoModeRun{{ID: "r1", Phase: domain.AutoPhaseCompleted}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/auto/r1/stop":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	run, err := c.StartAutoMode(context.Background(), domain.AutoModeRequest{ProjectPath: "/p", Goal: "g"})
	if err != nil || run.ID != "r1" {
		t.Fatalf("StartAutoMode: %+v %v", run, err)
	}
	got, err := c.GetAutoModeRun("r1")
	if err != nil || got.Phase != domain.AutoPhaseCompleted {
		t.Fatalf("GetAutoModeRun: %+v %v", got, err)
	}
	runs := c.ListAutoModeRuns("/p")
	if len(runs) != 1 || runs[0].ID != "r1" {
		t.Fatalf("ListAutoModeRuns: %+v", runs)
	}
	if empty := c.ListAutoModeRuns("/err"); len(empty) != 0 {
		t.Fatalf("expected empty on error, got %+v", empty)
	}
	if err := c.StopAutoMode("r1"); err != nil {
		t.Fatalf("StopAutoMode: %v", err)
	}
}
