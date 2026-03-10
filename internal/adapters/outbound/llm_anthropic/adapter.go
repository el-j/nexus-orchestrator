package llm_anthropic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

const (
	defaultBaseURL   = "https://api.anthropic.com"
	anthropicVersion = "2023-06-01"
	defaultMaxTokens = 4096
)

// Adapter implements ports.LLMClient for the Anthropic Claude API.
// This uses the native Anthropic Messages API (/v1/messages), NOT an
// OpenAI-compatible endpoint.
type Adapter struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewAdapter creates an Anthropic Claude adapter.
//
//	apiKey — Anthropic API key (required)
//	model  — model ID to use (e.g. "claude-sonnet-4-5", "claude-opus-4-5")
func NewAdapter(apiKey, model string) *Adapter {
	return &Adapter{
		baseURL: defaultBaseURL,
		apiKey:  apiKey,
		model:   model,
		httpClient: &http.Client{Timeout: 300 * time.Second},
	}
}

func (a *Adapter) ProviderName() string { return "Anthropic" }
func (a *Adapter) ActiveModel() string  { return a.model }
func (a *Adapter) ContextLimit() int    { return 0 }

// Ping checks Anthropic API reachability via the /v1/models endpoint.
func (a *Adapter) Ping() bool {
	req, err := a.newRequest(http.MethodGet, a.baseURL+"/v1/models", nil)
	if err != nil {
		return false
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// GetAvailableModels queries Anthropic's /v1/models endpoint for available models.
func (a *Adapter) GetAvailableModels() ([]string, error) {
	req, err := a.newRequest(http.MethodGet, a.baseURL+"/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("anthropic: build models request: %w", err)
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("anthropic: list models: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic: list models: unexpected status %d", resp.StatusCode)
	}
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("anthropic: decode models: %w", err)
	}
	ids := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		ids = append(ids, m.ID)
	}
	return ids, nil
}

// GenerateCode sends a single prompt as a user message to Claude.
func (a *Adapter) GenerateCode(prompt string) (string, error) {
	return a.sendMessages([]anthropicMessage{{Role: "user", Content: prompt}})
}

// Chat converts the multi-turn conversation history into Anthropic format.
// Consecutive same-role messages are merged (Anthropic requires alternating turns).
func (a *Adapter) Chat(messages []domain.Message) (string, error) {
	return a.sendMessages(toAnthropicMessages(messages))
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// toAnthropicMessages converts domain messages, filtering non-user/assistant roles
// and merging consecutive same-role messages.
func toAnthropicMessages(msgs []domain.Message) []anthropicMessage {
	out := make([]anthropicMessage, 0, len(msgs))
	for _, m := range msgs {
		if m.Role != domain.RoleUser && m.Role != domain.RoleAssistant {
			continue
		}
		if len(out) > 0 && out[len(out)-1].Role == string(m.Role) {
			out[len(out)-1].Content += "\n" + m.Content
		} else {
			out = append(out, anthropicMessage{Role: string(m.Role), Content: m.Content})
		}
	}
	return out
}

func (a *Adapter) sendMessages(messages []anthropicMessage) (string, error) {
	reqBody, err := json.Marshal(map[string]interface{}{
		"model":      a.model,
		"max_tokens": defaultMaxTokens,
		"messages":   messages,
	})
	if err != nil {
		return "", fmt.Errorf("anthropic: marshal request: %w", err)
	}
	req, err := a.newRequest(http.MethodPost, a.baseURL+"/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("anthropic: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic: request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		return "", fmt.Errorf("anthropic: rate limited (429)")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("anthropic: unexpected status %d", resp.StatusCode)
	}
	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&result); err != nil {
		return "", fmt.Errorf("anthropic: decode response: %w", err)
	}
	for _, block := range result.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}
	return "", fmt.Errorf("anthropic: no text content in response")
}

func (a *Adapter) newRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	return req, nil
}
























































































































































