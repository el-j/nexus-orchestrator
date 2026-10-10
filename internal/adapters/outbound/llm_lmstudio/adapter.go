// Package llm_lmstudio implements the LLMClient port for LM Studio's
// OpenAI-compatible API at 127.0.0.1:1234.
package llm_lmstudio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// DefaultBaseURL is the default LM Studio API endpoint. Override via NEXUS_LMSTUDIO_URL.
const DefaultBaseURL = "http://127.0.0.1:1234/v1"

// Adapter implements ports.LLMClient for LM Studio's OpenAI-compatible REST API.
type Adapter struct {
	baseURL    string
	nativeBase string // LM Studio-native base URL (without /v1 suffix)
	httpClient *http.Client

	infoMu       sync.Mutex
	contextLimit int       // last read value; 0 = unknown
	activeModel  string    // last read active model ID
	infoAt       time.Time // when the model info was last read successfully
	infoTried    time.Time // when the last attempt (successful or not) started
}

// Package-level so tests can shorten them; production code never reassigns them.
var (
	// infoTTL bounds how long the loaded model's identity and context length are
	// trusted: users swap models in LM Studio while the daemon keeps running.
	infoTTL = 30 * time.Second
	// infoRetry throttles re-queries while LM Studio is unreachable.
	infoRetry = 10 * time.Second
)

// NewLMStudioAdapter creates an Adapter pointing at the given LM Studio base URL
// (e.g. "http://127.0.0.1:1234/v1").
func NewLMStudioAdapter(baseURL string) *Adapter {
	return &Adapter{
		baseURL:    baseURL,
		nativeBase: strings.TrimSuffix(baseURL, "/v1"),
		httpClient: &http.Client{
			Timeout: 300 * time.Second, // large models can take 2-3 min for complex prompts
		},
	}
}

// ProviderName identifies this adapter.
func (a *Adapter) ProviderName() string { return "LM Studio" }

// BaseURL returns the configured endpoint URL for this adapter.
func (a *Adapter) BaseURL() string { return a.baseURL }

// modelInfo returns the loaded model's identifier and context length, reading
// them from LM Studio at most once per infoTTL (and, while LM Studio cannot be
// reached, retrying at most once per infoRetry). Both are zero values when
// unknown; a failed refresh keeps the last known values.
func (a *Adapter) modelInfo() (model string, contextLimit int) {
	a.infoMu.Lock()
	defer a.infoMu.Unlock()
	now := time.Now()
	fresh := a.activeModel != "" && now.Sub(a.infoAt) < infoTTL
	if !fresh && now.Sub(a.infoTried) >= infoRetry {
		a.infoTried = now
		if m, limit, ok := a.fetchModelInfo(); ok {
			a.activeModel, a.contextLimit, a.infoAt = m, limit, now
		}
	}
	return a.activeModel, a.contextLimit
}

// fetchModelInfo queries the native /api/v0/model endpoint for the loaded model
// and its context length. It falls back to the first ID of the OpenAI-compatible
// /models list when the native endpoint is unavailable. ok is false when no
// model identifier could be determined.
func (a *Adapter) fetchModelInfo() (model string, contextLimit int, ok bool) {
	resp, err := a.httpClient.Get(a.nativeBase + "/api/v0/model")
	if err == nil {
		defer resp.Body.Close()
		var result struct {
			Identifier    string `json:"identifier"`
			ContextLength int    `json:"contextLength"`
		}
		if resp.StatusCode == http.StatusOK && json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&result) == nil {
			if result.ContextLength > 0 {
				contextLimit = result.ContextLength
			}
			if result.Identifier != "" {
				return result.Identifier, contextLimit, true
			}
		}
	}
	models, err2 := a.GetAvailableModels()
	if err2 == nil && len(models) > 0 {
		return models[0], contextLimit, true
	}
	return "", 0, false
}

// ActiveModel returns the identifier of the model currently loaded in LM Studio.
// It queries the native /api/v0/model endpoint; falls back to the first ID from
// the OpenAI-compat /models list if the native endpoint is not available.
// Returns empty string when LM Studio is not reachable.
func (a *Adapter) ActiveModel() string {
	m, _ := a.modelInfo()
	return m
}

// activeModelOrDefault returns the currently active model ID, or "local-model" as
// the safe LM Studio default when the model ID cannot be determined.
func (a *Adapter) activeModelOrDefault() string {
	if m := a.ActiveModel(); m != "" {
		return m
	}
	return "local-model"
}

// ContextLimit returns the context-window size of the currently loaded model.
// It queries the native LM Studio /api/v0/model endpoint which includes
// contextLength; falls back to 0 when unavailable.
func (a *Adapter) ContextLimit() int {
	_, limit := a.modelInfo()
	return limit
}

// Ping checks whether LM Studio is reachable by hitting the /models endpoint.
func (a *Adapter) Ping() bool {
	resp, err := a.httpClient.Get(a.baseURL + "/models")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// GetAvailableModels returns the list of model IDs loaded in LM Studio.
func (a *Adapter) GetAvailableModels() ([]string, error) {
	resp, err := a.httpClient.Get(a.baseURL + "/models")
	if err != nil {
		return nil, fmt.Errorf("lmstudio: list models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lmstudio: list models: unexpected status %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("lmstudio: decode models: %w", err)
	}

	ids := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		ids = append(ids, m.ID)
	}
	return ids, nil
}

// chatCompletionRequest is the request body for the LM Studio /chat/completions endpoint.
type chatCompletionRequest struct {
	Model       string              `json:"model"`
	Messages    []map[string]string `json:"messages"`
	Temperature float64             `json:"temperature"`
}

// GenerateCode sends a chat completion request to LM Studio and returns the
// generated text.
func (a *Adapter) GenerateCode(prompt string) (string, error) {
	reqBody, err := json.Marshal(chatCompletionRequest{
		Model:       a.activeModelOrDefault(),
		Messages:    []map[string]string{{"role": "user", "content": prompt}},
		Temperature: 0.2,
	})
	if err != nil {
		return "", fmt.Errorf("lmstudio: marshal request: %w", err)
	}

	resp, err := a.httpClient.Post(
		a.baseURL+"/chat/completions",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return "", fmt.Errorf("lmstudio: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("lmstudio: generate: unexpected status %d", resp.StatusCode)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&result); err != nil {
		return "", fmt.Errorf("lmstudio: decode response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("lmstudio: no choices in response")
	}
	return result.Choices[0].Message.Content, nil
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

// Chat sends a multi-turn conversation history to LM Studio and returns the
// assistant reply. This is the preferred method for session-isolated generation.
func (a *Adapter) Chat(messages []domain.Message) (string, error) {
	reqBody, err := json.Marshal(chatCompletionRequest{
		Model:       a.activeModelOrDefault(),
		Messages:    messagesToMaps(messages),
		Temperature: 0.2,
	})
	if err != nil {
		return "", fmt.Errorf("lmstudio: marshal chat request: %w", err)
	}

	resp, err := a.httpClient.Post(
		a.baseURL+"/chat/completions",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return "", fmt.Errorf("lmstudio: chat request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("lmstudio: chat: unexpected status %d", resp.StatusCode)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&result); err != nil {
		return "", fmt.Errorf("lmstudio: decode chat response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("lmstudio: no choices in chat response")
	}
	return result.Choices[0].Message.Content, nil
}
