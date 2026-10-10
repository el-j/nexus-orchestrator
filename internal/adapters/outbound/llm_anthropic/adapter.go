// Package llm_anthropic implements the LLMClient port for the Anthropic Messages API.
package llm_anthropic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
		baseURL:    defaultBaseURL,
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: 300 * time.Second},
	}
}

// claudeContextLimits maps known Claude model IDs to their context window sizes.
var claudeContextLimits = map[string]int{
	"claude-3-opus-20240229":     200000,
	"claude-3-sonnet-20240229":   200000,
	"claude-3-haiku-20240307":    200000,
	"claude-3-5-sonnet-20241022": 200000,
	"claude-3-5-sonnet-20240620": 200000,
	"claude-3-5-haiku-20241022":  200000,
}

func (a *Adapter) ProviderName() string { return "Anthropic" }
func (a *Adapter) ActiveModel() string  { return a.model }
func (a *Adapter) BaseURL() string      { return a.baseURL }

// ContextLimit returns the context window size for the configured Claude model.
// Defaults to 200000 for unknown models (all modern Claude models support 200K).
func (a *Adapter) ContextLimit() int {
	if limit, ok := claudeContextLimits[a.model]; ok {
		return limit
	}
	return 200000
}

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
	return a.sendMessages("", []anthropicMessage{{Role: "user", Content: prompt}})
}

// Chat converts the multi-turn conversation history into Anthropic format.
// Consecutive same-role messages are merged (Anthropic requires alternating turns).
func (a *Adapter) Chat(messages []domain.Message) (string, error) {
	system, turns := toAnthropicMessages(messages)
	return a.sendMessages(system, turns)
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// toAnthropicMessages converts domain messages to the Messages API shape:
//
//   - system messages are joined into the top-level "system" prompt (the API has
//     no system role inside "messages"; dropping them would lose instructions),
//   - empty messages are skipped (the API rejects empty text blocks),
//   - consecutive same-role messages are merged (turns must alternate),
//   - leading assistant turns are dropped (the first turn must be a user turn).
func toAnthropicMessages(msgs []domain.Message) (system string, out []anthropicMessage) {
	var systemParts []string
	for _, m := range msgs {
		if strings.TrimSpace(m.Content) == "" {
			continue
		}
		switch m.Role {
		case domain.RoleSystem:
			systemParts = append(systemParts, m.Content)
		case domain.RoleUser, domain.RoleAssistant:
			if len(out) > 0 && out[len(out)-1].Role == string(m.Role) {
				out[len(out)-1].Content += "\n" + m.Content
			} else {
				out = append(out, anthropicMessage{Role: string(m.Role), Content: m.Content})
			}
		}
	}
	for len(out) > 0 && out[0].Role != "user" {
		out = out[1:]
	}
	return strings.Join(systemParts, "\n\n"), out
}

// anthropicRequest is the request body for the Anthropic /v1/messages endpoint.
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
}

func (a *Adapter) sendMessages(system string, messages []anthropicMessage) (string, error) {
	if len(messages) == 0 {
		return "", fmt.Errorf("anthropic: no user message to send")
	}
	reqBody, err := json.Marshal(anthropicRequest{
		Model:     a.model,
		MaxTokens: defaultMaxTokens,
		System:    system,
		Messages:  messages,
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
		return "", fmt.Errorf("anthropic: unexpected status %d%s", resp.StatusCode, errorDetail(resp.Body))
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

// errorDetail extracts ": <message>" from an Anthropic error body
// ({"type":"error","error":{"type":"...","message":"..."}}), or "" when the body
// carries none. The message tells the user *why* (for example "prompt is too
// long" or "invalid x-api-key"); it never contains the request's API key.
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

func (a *Adapter) newRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	return req, nil
}
