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
	name            string
	baseURL         string
	apiKey          string
	model           string
	httpClient      *http.Client
	availableModels []string
	availableOnce   sync.Once
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
	}
}

func (a *Adapter) ProviderName() string { return a.name }
func (a *Adapter) ActiveModel() string  { return a.model }
func (a *Adapter) BaseURL() string      { return a.baseURL }
func (a *Adapter) ContextLimit() int    { return 0 }

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
// Results are cached for the lifetime of the adapter instance.
func (a *Adapter) GetAvailableModels() ([]string, error) {
	var getErr error
	a.availableOnce.Do(func() {
		req, err := http.NewRequest(http.MethodGet, a.baseURL+"/models", nil)
		if err != nil {
			getErr = fmt.Errorf("%s: build models request: %w", a.name, err)
			return
		}
		a.setAuthHeader(req)
		resp, err := a.httpClient.Do(req)
		if err != nil {
			getErr = fmt.Errorf("%s: list models: %w", a.name, err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			getErr = fmt.Errorf("%s: list models: unexpected status %d", a.name, resp.StatusCode)
			return
		}
		var result struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&result); err != nil {
			getErr = fmt.Errorf("%s: decode models: %w", a.name, err)
			return
		}
		ids := make([]string, 0, len(result.Data))
		for _, m := range result.Data {
			ids = append(ids, m.ID)
		}
		a.availableModels = ids
	})
	return a.availableModels, getErr
}

// GenerateCode sends a single prompt as a user message and returns the response.
func (a *Adapter) GenerateCode(prompt string) (string, error) {
	return a.chat([]map[string]string{{"role": "user", "content": prompt}})
}

// Chat converts the multi-turn conversation history into an OpenAI chat completion
// request and returns the assistant reply.
func (a *Adapter) Chat(messages []domain.Message) (string, error) {
	msgs := make([]map[string]string, len(messages))
	for i, m := range messages {
		msgs[i] = map[string]string{"role": string(m.Role), "content": m.Content}
	}
	return a.chat(msgs)
}

func (a *Adapter) chat(messages []map[string]string) (string, error) {
	reqBody, err := json.Marshal(map[string]interface{}{
		"model":       a.model,
		"messages":    messages,
		"temperature": 0.2,
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
