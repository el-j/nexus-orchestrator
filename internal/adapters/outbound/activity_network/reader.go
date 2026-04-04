// Package activity_network implements ports.ActivityReader by probing local
// AI API endpoints to discover which models are currently loaded.
package activity_network

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// NetworkProbeReader polls local AI API endpoints to detect loaded/active models.
// It generates an AIActivity for each actively loaded model found.
type NetworkProbeReader struct {
	lmStudioURL    string
	ollamaURL      string
	antigravityURL string
	httpClient     *http.Client         // 2s timeout
	lastModels     map[string]time.Time // modelID -> time first seen
	mu             sync.Mutex
}

// NewNetworkProbeReader creates a NetworkProbeReader reading base URLs from
// environment variables with sensible defaults.
func NewNetworkProbeReader() *NetworkProbeReader {
	lmStudioURL := os.Getenv("NEXUS_LMSTUDIO_URL")
	if lmStudioURL == "" {
		lmStudioURL = "http://127.0.0.1:1234/v1"
	}
	ollamaURL := os.Getenv("NEXUS_OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://127.0.0.1:11434"
	}
	antigravityURL := os.Getenv("NEXUS_ANTIGRAVITY_URL")
	if antigravityURL == "" {
		antigravityURL = "http://127.0.0.1:4315/v1"
	}
	return newWithURLs(lmStudioURL, ollamaURL, antigravityURL)
}

// NewNetworkProbeReaderWithURLs creates a NetworkProbeReader with explicit
// base URLs. Intended for integration tests and custom deployments.
func NewNetworkProbeReaderWithURLs(lmStudioURL, ollamaURL, antigravityURL string) *NetworkProbeReader {
	return newWithURLs(lmStudioURL, ollamaURL, antigravityURL)
}

func newWithURLs(lmStudioURL, ollamaURL, antigravityURL string) *NetworkProbeReader {
	return &NetworkProbeReader{
		lmStudioURL:    lmStudioURL,
		ollamaURL:      ollamaURL,
		antigravityURL: antigravityURL,
		httpClient:     &http.Client{Timeout: 2 * time.Second},
		lastModels:     make(map[string]time.Time),
	}
}

// SourceName implements ports.ActivityReader.
func (r *NetworkProbeReader) SourceName() string {
	return "network-probe"
}

// ReadActivities probes all configured endpoints concurrently. Unreachable or
// erroring endpoints are silently skipped — only a nil error is returned.
func (r *NetworkProbeReader) ReadActivities(ctx context.Context, since time.Time) ([]domain.AIActivity, error) {
	type slot struct{ acts []domain.AIActivity }
	slots := make([]slot, 3)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		acts, _ := r.probeOpenAIModels(ctx, r.lmStudioURL, "lm-studio")
		slots[0].acts = acts
	}()
	go func() {
		defer wg.Done()
		acts, _ := r.probeOllamaPS(ctx, r.ollamaURL)
		slots[1].acts = acts
	}()
	go func() {
		defer wg.Done()
		acts, _ := r.probeOpenAIModels(ctx, r.antigravityURL, "antigravity")
		slots[2].acts = acts
	}()

	wg.Wait()

	var all []domain.AIActivity
	for _, s := range slots {
		all = append(all, s.acts...)
	}
	return all, nil
}

// ---- helpers ----

// openAIModelsResponse matches the /v1/models response from LM Studio and
// any OpenAI-compatible API.
type openAIModelsResponse struct {
	Object string `json:"object"`
	Data   []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// ollamaPSResponse matches the /api/ps response from Ollama.
type ollamaPSResponse struct {
	Models []struct {
		Name  string `json:"name"`
		Model string `json:"model"`
	} `json:"models"`
}

// normalizeID replaces / and : with - so model IDs are safe for use in
// composite activity ID strings.
func normalizeID(s string) string {
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, ":", "-")
	return s
}

// probeOpenAIModels calls GET <baseURL>/models and returns one AIActivity per
// model ID. Returns nil, nil on any network or parse failure.
func (r *NetworkProbeReader) probeOpenAIModels(ctx context.Context, baseURL, providerName string) ([]domain.AIActivity, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		return nil, nil //nolint:nilerr
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var parsed openAIModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, nil //nolint:nilerr
	}

	now := time.Now()
	activities := make([]domain.AIActivity, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		activities = append(activities, domain.AIActivity{
			ID:           fmt.Sprintf("network-probe-%s-%s", normalizeID(providerName), normalizeID(m.ID)),
			AgentName:    providerName,
			ActivityType: domain.ActivityTypeGeneration,
			Summary:      fmt.Sprintf("Model %s loaded", m.ID),
			Model:        m.ID,
			Timestamp:    now,
			Metadata:     map[string]string{"provider": providerName, "url": baseURL},
		})
	}
	return activities, nil
}

// probeOllamaPS calls GET <baseURL>/api/ps and returns one AIActivity per
// running model. Returns nil, nil on any network or parse failure.
func (r *NetworkProbeReader) probeOllamaPS(ctx context.Context, baseURL string) ([]domain.AIActivity, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/ps", nil)
	if err != nil {
		return nil, nil //nolint:nilerr
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var parsed ollamaPSResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, nil //nolint:nilerr
	}

	now := time.Now()
	activities := make([]domain.AIActivity, 0, len(parsed.Models))
	for _, m := range parsed.Models {
		modelID := m.Model
		if modelID == "" {
			modelID = m.Name
		}
		activities = append(activities, domain.AIActivity{
			ID:           fmt.Sprintf("network-probe-ollama-%s", normalizeID(modelID)),
			AgentName:    "ollama",
			ActivityType: domain.ActivityTypeGeneration,
			Summary:      fmt.Sprintf("Model %s loaded", modelID),
			Model:        modelID,
			Timestamp:    now,
			Metadata:     map[string]string{"provider": "ollama", "url": baseURL},
		})
	}
	return activities, nil
}
