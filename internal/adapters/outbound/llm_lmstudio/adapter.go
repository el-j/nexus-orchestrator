package llm_lmstudio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Adapter implements ports.LLMClient for LM Studio's OpenAI-compatible REST API.
type Adapter struct {
	baseURL    string
	httpClient *http.Client
}

// NewLMStudioAdapter creates an Adapter pointing at the given LM Studio base URL
// (e.g. "http://127.0.0.1:1234/v1").
func NewLMStudioAdapter(baseURL string) *Adapter {
	return &Adapter{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ProviderName identifies this adapter.
func (a *Adapter) ProviderName() string { return "LM Studio" }

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

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
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
		"model": "local-model",
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

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("lmstudio: decode response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("lmstudio: no choices in response")
	}
	return result.Choices[0].Message.Content, nil
}
