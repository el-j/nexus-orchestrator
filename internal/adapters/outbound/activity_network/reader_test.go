package activity_network_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/outbound/activity_network"
	"nexus-orchestrator/internal/core/domain"
)

// ---- helpers ----

func unreachableURL(ts *httptest.Server) string {
	ts.Close()
	return ts.URL // server is closed; requests will get "connection refused"
}

// ---- probeOpenAIModels via ReadActivities ----

func TestProbeOpenAIModels_ReturnsActivities(t *testing.T) {
	payload := map[string]any{
		"object": "list",
		"data": []map[string]any{
			{"id": "meta/llama-3-8b", "object": "model"},
			{"id": "mistral-7b", "object": "model"},
		},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload) //nolint:errcheck
	}))
	defer ts.Close()

	// Route only lmStudioURL to the test server; others set to closed ports.
	closed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedURL := unreachableURL(closed)

	reader := activity_network.NewNetworkProbeReaderWithURLs(ts.URL, closedURL, closedURL)
	acts, err := reader.ReadActivities(context.Background(), time.Time{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(acts) != 2 {
		t.Fatalf("expected 2 activities, got %d", len(acts))
	}

	for _, a := range acts {
		if a.AgentName != "lm-studio" {
			t.Errorf("expected AgentName lm-studio, got %q", a.AgentName)
		}
		if a.ActivityType != domain.ActivityTypeGeneration {
			t.Errorf("expected ActivityTypeGeneration, got %q", a.ActivityType)
		}
		if a.Model == "" {
			t.Error("expected non-empty Model")
		}
		if a.Metadata["provider"] != "lm-studio" {
			t.Errorf("expected metadata provider=lm-studio, got %q", a.Metadata["provider"])
		}
		if a.ID == "" {
			t.Error("expected non-empty ID")
		}
	}
}

func TestProbeOpenAIModels_NormalisesModelIDInActivityID(t *testing.T) {
	payload := map[string]any{
		"object": "list",
		"data": []map[string]any{
			{"id": "org/model:v1", "object": "model"},
		},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload) //nolint:errcheck
	}))
	defer ts.Close()

	closed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedURL := unreachableURL(closed)

	reader := activity_network.NewNetworkProbeReaderWithURLs(ts.URL, closedURL, closedURL)
	acts, err := reader.ReadActivities(context.Background(), time.Time{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(acts) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(acts))
	}
	// slashes and colons must be replaced with dashes in ID
	const wantID = "network-probe-lm-studio-org-model-v1"
	if acts[0].ID != wantID {
		t.Errorf("ID = %q, want %q", acts[0].ID, wantID)
	}
}

// ---- probeOllamaPS via ReadActivities ----

func TestProbeOllamaPS_ReturnsActivities(t *testing.T) {
	payload := map[string]any{
		"models": []map[string]any{
			{"name": "llama3:8b", "model": "llama3:8b", "size": 4_294_967_296},
			{"name": "mistral:latest", "model": "mistral:latest", "size": 4_294_967_296},
		},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/ps" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload) //nolint:errcheck
	}))
	defer ts.Close()

	closed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedURL := unreachableURL(closed)

	reader := activity_network.NewNetworkProbeReaderWithURLs(closedURL, ts.URL, closedURL)
	acts, err := reader.ReadActivities(context.Background(), time.Time{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(acts) != 2 {
		t.Fatalf("expected 2 activities, got %d", len(acts))
	}

	for _, a := range acts {
		if a.AgentName != "ollama" {
			t.Errorf("expected AgentName ollama, got %q", a.AgentName)
		}
		if a.ActivityType != domain.ActivityTypeGeneration {
			t.Errorf("expected ActivityTypeGeneration, got %q", a.ActivityType)
		}
		if a.Model == "" {
			t.Error("expected non-empty Model")
		}
		if a.Metadata["provider"] != "ollama" {
			t.Errorf("expected metadata provider=ollama, got %q", a.Metadata["provider"])
		}
	}
}

func TestProbeOllamaPS_FallsBackToNameField(t *testing.T) {
	// model field empty — should fall back to name
	payload := map[string]any{
		"models": []map[string]any{
			{"name": "phi3:mini", "model": ""},
		},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload) //nolint:errcheck
	}))
	defer ts.Close()

	closed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedURL := unreachableURL(closed)

	reader := activity_network.NewNetworkProbeReaderWithURLs(closedURL, ts.URL, closedURL)
	acts, err := reader.ReadActivities(context.Background(), time.Time{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(acts) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(acts))
	}
	if acts[0].Model != "phi3:mini" {
		t.Errorf("expected model phi3:mini, got %q", acts[0].Model)
	}
}

// ---- unreachable endpoint ----

func TestReadActivities_UnreachableEndpointsReturnEmptyNoError(t *testing.T) {
	closed1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closed2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closed3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	url1 := unreachableURL(closed1)
	url2 := unreachableURL(closed2)
	url3 := unreachableURL(closed3)

	reader := activity_network.NewNetworkProbeReaderWithURLs(url1, url2, url3)
	acts, err := reader.ReadActivities(context.Background(), time.Time{})

	if err != nil {
		t.Fatalf("unexpected error from unreachable endpoints: %v", err)
	}
	if len(acts) != 0 {
		t.Errorf("expected 0 activities from unreachable endpoints, got %d", len(acts))
	}
}

func TestReadActivities_Non200ResponseReturnsEmpty(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	reader := activity_network.NewNetworkProbeReaderWithURLs(ts.URL, ts.URL, ts.URL)
	acts, err := reader.ReadActivities(context.Background(), time.Time{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(acts) != 0 {
		t.Errorf("expected 0 activities for non-200 response, got %d", len(acts))
	}
}

// ---- SourceName ----

func TestSourceName(t *testing.T) {
	reader := activity_network.NewNetworkProbeReader()
	if got := reader.SourceName(); got != "network-probe" {
		t.Errorf("SourceName() = %q, want \"network-probe\"", got)
	}
}

// ---- interface compliance ----

func TestImplementsActivityReader(t *testing.T) {
	// Compile-time check that *NetworkProbeReader satisfies ports.ActivityReader.
	// We import domain here to avoid an unused import; the real check is the
	// ReadActivities signature parity verified by the compiler.
	var _ interface {
		ReadActivities(ctx context.Context, since time.Time) ([]domain.AIActivity, error)
		SourceName() string
	} = activity_network.NewNetworkProbeReader()
}
