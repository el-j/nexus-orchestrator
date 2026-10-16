// Package llm_openaicompat implements the LLMClient port for generic
// OpenAI-compatible API endpoints.
package llm_openaicompat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"nexus-orchestrator/internal/adapters/outbound/httperr"
	"sync"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// Adapter implements ports.LLMClient for any OpenAI-compatible REST API.
// Use this for OpenAI, GitHub Copilot, Azure OpenAI, OpenRouter, etc.
type Adapter struct {
	name       string
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client

	modelsMu       sync.Mutex
	cachedModels   []string
	modelsCachedAt time.Time
	modelsTTL      time.Duration

	limitsMu  sync.Mutex
	limits    map[string]int // context length per model, as the server reports it
	limitsTry time.Time      // when /models was last asked for it, successful or not
}

// NewAdapter creates a generic OpenAI-compatible adapter.
//
//	name    — human-readable provider name (e.g. "OpenAI", "GitHub Copilot")
//	baseURL — API base without trailing slash (e.g. "https://api.openai.com/v1")
//	apiKey  — Bearer token; empty string suppresses the Authorization header
//	model   — default model to use (e.g. "gpt-4o", "gpt-4-turbo")
func NewAdapter(name, baseURL, apiKey, model string) *Adapter {
	return &Adapter{
		name:       name,
		baseURL:    baseURL,
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: 300 * time.Second},
		modelsTTL:  5 * time.Minute,
	}
}

// ProviderName returns the human-readable provider name shown in the UI and used for routing.
func (a *Adapter) ProviderName() string { return a.name }

// ActiveModel returns the model this adapter sends requests to.
func (a *Adapter) ActiveModel() string { return a.model }

// BaseURL returns the API endpoint this adapter talks to.
func (a *Adapter) BaseURL() string { return a.baseURL }

// defaultContextLimit is assumed when the server does not report a context length.
const (
	defaultContextLimit = 131072
	// limitsRetry throttles how often /models is asked for context lengths.
	limitsRetry = 30 * time.Second
)

// ContextLimit returns the context length the server reports for the configured model (the
// /models entry's context_length, context_window or max_model_len), 131072 (128K) when it
// reports none, and 0 when no model is configured.
func (a *Adapter) ContextLimit(ctx context.Context) int {
	if a.model == "" {
		return 0
	}
	a.limitsMu.Lock()
	due := time.Since(a.limitsTry) >= limitsRetry
	if due {
		a.limitsTry = time.Now()
	}
	a.limitsMu.Unlock()
	if due {
		_, _ = a.fetchModels(ctx, true) // a failure keeps the default limit
	}
	a.limitsMu.Lock()
	defer a.limitsMu.Unlock()
	if n := a.limits[a.model]; n > 0 {
		return n
	}
	return defaultContextLimit
}

// Ping checks whether the provider is reachable by hitting the /models endpoint.
func (a *Adapter) Ping(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/models", nil)
	if err != nil {
		return false
	}
	a.setAuthHeader(req)
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// GetAvailableModels returns model IDs from the provider's /models list.
// Results are cached for modelsTTL (default 5 minutes). A failed call is never
// cached — the next call will retry the API.
func (a *Adapter) GetAvailableModels(ctx context.Context) ([]string, error) {
	a.modelsMu.Lock()
	if !a.modelsCachedAt.IsZero() && time.Since(a.modelsCachedAt) < a.modelsTTL {
		cached := a.cachedModels
		a.modelsMu.Unlock()
		return cached, nil
	}
	a.modelsMu.Unlock()
	return a.fetchModels(ctx, false)
}

// fetchModels reads /models (no lock held during the request), caches the IDs on success and
// remembers the context length of every model that reports one.
func (a *Adapter) fetchModels(ctx context.Context, quick bool) ([]string, error) {
	if quick { // asked on the way to a prompt: do not stall it for long
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("%s: build models request: %w", a.name, err)
	}
	a.setAuthHeader(req)
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: list models: %w", a.name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: list models: unexpected status %d", a.name, resp.StatusCode)
	}
	var result struct {
		Data []struct {
			ID            string `json:"id"`
			ContextLength int    `json:"context_length"`
			ContextWindow int    `json:"context_window"`
			MaxModelLen   int    `json:"max_model_len"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("%s: decode models: %w", a.name, err)
	}
	ids := make([]string, 0, len(result.Data))
	limits := make(map[string]int, len(result.Data))
	for _, m := range result.Data {
		ids = append(ids, m.ID)
		for _, n := range []int{m.ContextLength, m.ContextWindow, m.MaxModelLen} {
			if n > 0 {
				limits[m.ID] = n
				break
			}
		}
	}

	// Cache on success only.
	a.modelsMu.Lock()
	a.cachedModels = ids
	a.modelsCachedAt = time.Now()
	a.modelsMu.Unlock()
	a.limitsMu.Lock()
	a.limits = limits
	a.limitsMu.Unlock()

	return ids, nil
}

// GenerateCode sends a single prompt as a user message and returns the response.
func (a *Adapter) GenerateCode(ctx context.Context, prompt string) (string, error) {
	return a.chat(ctx, []map[string]string{{"role": "user", "content": prompt}})
}

// messagesToMaps converts a slice of domain.Message to the map representation
// expected by OpenAI-compatible chat completion APIs.
func messagesToMaps(msgs []domain.Message) []map[string]string {
	out := make([]map[string]string, len(msgs))
	for i, m := range msgs {
		out[i] = map[string]string{"role": string(m.Role), "content": m.Content}
	}
	return out
}

// Chat converts the multi-turn conversation history into an OpenAI chat completion
// request and returns the assistant reply.
func (a *Adapter) Chat(ctx context.Context, messages []domain.Message) (string, error) {
	return a.chat(ctx, messagesToMaps(messages))
}

// chatCompletionRequest is the request body for OpenAI-compatible /chat/completions.
type chatCompletionRequest struct {
	Model       string              `json:"model"`
	Messages    []map[string]string `json:"messages"`
	Temperature float64             `json:"temperature"`
	Stream      bool                `json:"stream,omitempty"`
}

func (a *Adapter) chat(ctx context.Context, messages []map[string]string) (string, error) {
	reqBody, err := json.Marshal(chatCompletionRequest{
		Model:       a.model,
		Messages:    messages,
		Temperature: 0.2,
	})
	if err != nil {
		return "", fmt.Errorf("%s: marshal request: %w", a.name, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("%s: build request: %w", a.name, err)
	}
	req.Header.Set("Content-Type", "application/json")
	a.setAuthHeader(req)
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s: request: %w", a.name, err)
	}
	defer resp.Body.Close()
	if err := httperr.FromResponse(a.name, resp, time.Now()); err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s: unexpected status %d%s", a.name, resp.StatusCode, errorDetail(resp.Body))
	}
	return a.decodeCompletion(resp.Body)
}

// decodeCompletion reads a non-streamed chat completion and returns the first reply.
func (a *Adapter) decodeCompletion(body io.Reader) (string, error) {
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(io.LimitReader(body, 10<<20)).Decode(&result); err != nil {
		return "", fmt.Errorf("%s: decode response: %w", a.name, err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("%s: no choices in response", a.name)
	}
	return result.Choices[0].Message.Content, nil
}

// errorDetail extracts ": <message>" from an OpenAI-style error body
// ({"error":{"message":"..."}}), or "" when the body carries none.
func errorDetail(body io.Reader) string {
	var e struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(body, 4096)).Decode(&e); err != nil || e.Error.Message == "" {
		return ""
	}
	return ": " + e.Error.Message
}

func (a *Adapter) setAuthHeader(req *http.Request) {
	if a.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
	}
}
