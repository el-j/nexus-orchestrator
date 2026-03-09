package llm_ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"nexus-ai/internal/core/domain"
)

// Adapter implements ports.LLMClient for Ollama's REST API.
type Adapter struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewOllamaAdapter creates an Adapter pointing at the given Ollama base URL
// (e.g. "http://127.0.0.1:11434") with the specified default model.
func NewOllamaAdapter(baseURL, model string) *Adapter {
	return &Adapter{
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// ProviderName identifies this adapter.
func (a *Adapter) ProviderName() string { return "Ollama" }

// Ping checks whether Ollama is reachable.
func (a *Adapter) Ping() bool {
	resp, err := a.httpClient.Get(a.baseURL + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// GetAvailableModels returns the list of model names pulled into Ollama.
func (a *Adapter) GetAvailableModels() ([]string, error) {
	resp, err := a.httpClient.Get(a.baseURL + "/api/tags")
	if err != nil {
		return nil, fmt.Errorf("ollama: list models: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ollama: decode models: %w", err)
	}

	names := make([]string, 0, len(result.Models))
	for _, m := range result.Models {
		names = append(names, m.Name)
	}
	return names, nil
}

// GenerateCode sends a chat completion request to Ollama and returns the generated text.
func (a *Adapter) GenerateCode(prompt string) (string, error) {
	reqBody, err := json.Marshal(map[string]interface{}{
		"model":  a.model,
		"prompt": prompt,
		"stream": false,
	})
	if err != nil {
		return "", fmt.Errorf("ollama: marshal request: %w", err)
	}

	resp, err := a.httpClient.Post(
		a.baseURL+"/api/generate",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return "", fmt.Errorf("ollama: request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("ollama: decode response: %w", err)
	}
	return result.Response, nil
}

// Chat sends a multi-turn conversation history to Ollama using the /api/chat
// endpoint and returns the assistant reply.
func (a *Adapter) Chat(messages []domain.Message) (string, error) {
	msgs := make([]map[string]string, len(messages))
	for i, m := range messages {
		msgs[i] = map[string]string{"role": m.Role, "content": m.Content}
	}
	reqBody, err := json.Marshal(map[string]interface{}{
		"model":    a.model,
		"messages": msgs,
		"stream":   false,
	})
	if err != nil {
		return "", fmt.Errorf("ollama: marshal chat request: %w", err)
	}

	resp, err := a.httpClient.Post(
		a.baseURL+"/api/chat",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return "", fmt.Errorf("ollama: chat request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("ollama: decode chat response: %w", err)
	}
	return result.Message.Content, nil
}
