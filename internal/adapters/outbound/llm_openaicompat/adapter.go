// Package llm_openaicompat implements the LLMClient port for generic
// OpenAI-compatible API endpoints.
package llm_openaicompat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func (a *Adapter) ProviderName() string { return a.name }
func (a *Adapter) ActiveModel() string  { return a.model }
func (a *Adapter) BaseURL() string      { return a.baseURL }

// openAIContextLimits maps known OpenAI model IDs to their context window sizes.
var openAIContextLimits = map[string]int{
	"gpt-4o":      128000,
	"gpt-4o-mini": 128000,
	"gpt-4":       8192,
	"gpt-4-turbo": 128000,
}

// ContextLimit returns the context window size for the configured model.
// Falls back to 32768 for unknown models.
func (a *Adapter) ContextLimit() int {
	if a.model == "" {
		return 0
	}
	if limit, ok := openAIContextLimits[a.model]; ok {
		return limit
	}
	return 32768
}

// Ping checks whether the provider is reachable by hitting the /models endpoint.
func (a *Adapter) Ping() bool {
	req, err := http.NewRequest(http.MethodGet, a.baseURL+"/models", nil)
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
func (a *Adapter) GetAvailableModels() ([]string, error) {
	a.modelsMu.Lock()
	if !a.modelsCachedAt.IsZero() && time.Since(a.modelsCachedAt) < a.modelsTTL {
		cached := a.cachedModels
		a.modelsMu.Unlock()
		return cached, nil
	}
	a.modelsMu.Unlock()

	// Fetch from API (no lock held during network request).
	req, err := http.NewRequest(http.MethodGet, a.baseURL+"/models", nil)
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
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("%s: decode models: %w", a.name, err)
	}
	ids := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		ids = append(ids, m.ID)
	}

	// Cache on success only.
	a.modelsMu.Lock()
	a.cachedModels = ids
	a.modelsCachedAt = time.Now()
	a.modelsMu.Unlock()

	return ids, nil
}

// GenerateCode sends a single prompt as a user message and returns the response.
func (a *Adapter) GenerateCode(prompt string) (string, error) {
	return a.chat([]map[string]string{{"role": "user", "content": prompt}})
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
func (a *Adapter) Chat(messages []domain.Message) (string, error) {
	return a.chat(messagesToMaps(messages))
}

// chatCompletionRequest is the request body for OpenAI-compatible /chat/completions.
type chatCompletionRequest struct {
	Model       string              `json:"model"`
	Messages    []map[string]string `json:"messages"`
	Temperature float64             `json:"temperature"`
}

func (a *Adapter) chat(messages []map[string]string) (string, error) {
	reqBody, err := json.Marshal(chatCompletionRequest{
		Model:       a.model,
		Messages:    messages,
		Temperature: 0.2,
	})
	if err != nil {
		return "", fmt.Errorf("%s: marshal request: %w", a.name, err)
	}
	req, err := http.NewRequest(http.MethodPost, a.baseURL+"/chat/completions", bytes.NewReader(reqBody))
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
	if resp.StatusCode == http.StatusTooManyRequests {
		return "", fmt.Errorf("%s: rate limited (429)", a.name)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s: unexpected status %d", a.name, resp.StatusCode)
	}
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&result); err != nil {
		return "", fmt.Errorf("%s: decode response: %w", a.name, err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("%s: no choices in response", a.name)
	}
	return result.Choices[0].Message.Content, nil
}

func (a *Adapter) setAuthHeader(req *http.Request) {
	if a.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
	}
}
