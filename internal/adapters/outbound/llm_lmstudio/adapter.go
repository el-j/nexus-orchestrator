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

// Adapter implements ports.LLMClient for LM Studio's OpenAI-compatible REST API.
type Adapter struct {
	baseURL      string
	nativeBase   string // LM Studio-native base URL (without /v1 suffix)
	httpClient   *http.Client
	contextLimit int       // cached value; 0 = unknown
	limitOnce    sync.Once // ensures the context-limit query runs at most once
	activeModel  string    // cached active model ID
	modelOnce    sync.Once // ensures the model query runs at most once
}

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

// ActiveModel returns the identifier of the model currently loaded in LM Studio.
// It queries the native /api/v0/model endpoint; falls back to the first ID from
// the OpenAI-compat /models list if the native endpoint is not available.
// Returns empty string when LM Studio is not reachable.
func (a *Adapter) ActiveModel() string {
	a.modelOnce.Do(func() {
		// Primary: LM Studio native endpoint returns {"identifier":"...", ...}
		resp, err := a.httpClient.Get(a.nativeBase + "/api/v0/model")
		if err == nil {
			defer resp.Body.Close()
			var result struct {
				Identifier string `json:"identifier"`
			}
			if json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&result) == nil && result.Identifier != "" {
				a.activeModel = result.Identifier
				return
			}
		}
		// Fallback: first model ID from OpenAI-compat /models list
		models, err2 := a.GetAvailableModels()
		if err2 == nil && len(models) > 0 {
			a.activeModel = models[0]
		}
	})
	return a.activeModel
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
	a.limitOnce.Do(func() {
		resp, err := a.httpClient.Get(a.nativeBase + "/api/v0/model")
		if err != nil {
			return
		}
		defer resp.Body.Close()
		var result struct {
			ContextLength int `json:"contextLength"`
		}
		if json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&result) == nil && result.ContextLength > 0 {
			a.contextLimit = result.ContextLength
		}
	})
	return a.contextLimit
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

// GenerateCode sends a chat completion request to LM Studio and returns the
// generated text.
func (a *Adapter) GenerateCode(prompt string) (string, error) {
	reqBody, err := json.Marshal(map[string]interface{}{
		"model": a.activeModelOrDefault(),
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.2,
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

// Chat sends a multi-turn conversation history to LM Studio and returns the
// assistant reply. This is the preferred method for session-isolated generation.
func (a *Adapter) Chat(messages []domain.Message) (string, error) {
	msgs := make([]map[string]string, len(messages))
	for i, m := range messages {
		msgs[i] = map[string]string{"role": string(m.Role), "content": m.Content}
	}
	reqBody, err := json.Marshal(map[string]interface{}{
		"model":       a.activeModelOrDefault(),
		"messages":    msgs,
		"temperature": 0.2,
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
