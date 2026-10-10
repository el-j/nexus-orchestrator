// Package llm_ollama implements the LLMClient port for Ollama's REST API.
// The default base URL is DefaultBaseURL; override via NEXUS_OLLAMA_URL.
package llm_ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// DefaultBaseURL is the default Ollama endpoint used when NEXUS_OLLAMA_URL is not set.
const DefaultBaseURL = "http://127.0.0.1:11434"

// Adapter implements ports.LLMClient for Ollama's REST API.
type Adapter struct {
	baseURL    string
	model      string
	httpClient *http.Client

	infoMu       sync.Mutex
	contextLimit int       // last successfully read value; 0 = unknown
	infoAt       time.Time // when contextLimit was last read successfully
	infoTried    time.Time // when the last attempt (successful or not) started
}

// Package-level so tests can shorten them; production code never reassigns them.
var (
	// infoTTL is how long a successfully read context limit is trusted. Models can
	// be swapped or re-created with another num_ctx, so it must not live forever.
	infoTTL = 5 * time.Minute
	// infoRetry throttles re-queries while Ollama is unreachable or the model unknown.
	infoRetry = 10 * time.Second
)

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

// ActiveModel returns the configured default model name for this Ollama instance.
func (a *Adapter) ActiveModel() string { return a.model }

// BaseURL returns the configured endpoint URL for this adapter.
func (a *Adapter) BaseURL() string { return a.baseURL }

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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama: list models: unexpected status %d", resp.StatusCode)
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("ollama: decode models: %w", err)
	}

	names := make([]string, 0, len(result.Models))
	for _, m := range result.Models {
		names = append(names, m.Name)
	}
	return names, nil
}

// generateRequest is the request body for the Ollama /api/generate endpoint.
type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// ollamaChatRequest is the request body for the Ollama /api/chat endpoint.
type ollamaChatRequest struct {
	Model    string              `json:"model"`
	Messages []map[string]string `json:"messages"`
	Stream   bool                `json:"stream"`
}

// GenerateCode sends a chat completion request to Ollama and returns the generated text.
func (a *Adapter) GenerateCode(prompt string) (string, error) {
	reqBody, err := json.Marshal(generateRequest{
		Model:  a.model,
		Prompt: prompt,
		Stream: false,
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

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama: generate: unexpected status %d", resp.StatusCode)
	}

	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&result); err != nil {
		return "", fmt.Errorf("ollama: decode response: %w", err)
	}
	if result.Response == "" {
		return "", fmt.Errorf("ollama: empty response from model %q", a.model)
	}
	return result.Response, nil
}

// ContextLimit queries Ollama's /api/show endpoint for the model's context
// window size. Returns 0 when it cannot be determined (safe fallback — the
// caller skips its pre-flight check). A successful value is cached for infoTTL;
// failures are retried at most every infoRetry, so a transient outage at the
// first call does not disable the check for the life of the process.
func (a *Adapter) ContextLimit() int {
	a.infoMu.Lock()
	defer a.infoMu.Unlock()
	now := time.Now()
	if a.contextLimit > 0 && now.Sub(a.infoAt) < infoTTL {
		return a.contextLimit
	}
	if now.Sub(a.infoTried) < infoRetry {
		return a.contextLimit
	}
	a.infoTried = now
	if n := a.fetchContextLimit(); n > 0 {
		a.contextLimit = n
		a.infoAt = now
	}
	return a.contextLimit
}

// fetchContextLimit reads the context length from /api/show. Ollama names the
// key after the model architecture ("llama.context_length",
// "qwen2.context_length", "gemma3.context_length", ...), so the architecture
// from "general.architecture" is tried first, then any "*.context_length" key.
func (a *Adapter) fetchContextLimit() int {
	body, err := json.Marshal(map[string]string{"name": a.model})
	if err != nil {
		return 0
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(a.baseURL+"/api/show", "application/json", bytes.NewReader(body))
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0
	}
	var result struct {
		ModelInfo map[string]any `json:"model_info"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&result) != nil {
		return 0
	}
	return contextLengthFromModelInfo(result.ModelInfo)
}

// contextLengthFromModelInfo extracts the context length from /api/show's model_info.
func contextLengthFromModelInfo(info map[string]any) int {
	asInt := func(v any) int {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
		return 0
	}
	if arch, ok := info["general.architecture"].(string); ok && arch != "" {
		if n := asInt(info[arch+".context_length"]); n > 0 {
			return n
		}
	}
	keys := make([]string, 0, len(info))
	for k := range info {
		if strings.HasSuffix(k, ".context_length") {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys) // deterministic when several architectures are reported
	for _, k := range keys {
		if n := asInt(info[k]); n > 0 {
			return n
		}
	}
	return 0
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

// Chat sends a multi-turn conversation history to Ollama using the /api/chat
// endpoint and returns the assistant reply.
func (a *Adapter) Chat(messages []domain.Message) (string, error) {
	reqBody, err := json.Marshal(ollamaChatRequest{
		Model:    a.model,
		Messages: messagesToMaps(messages),
		Stream:   false,
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

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama: chat: unexpected status %d", resp.StatusCode)
	}

	var result struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&result); err != nil {
		return "", fmt.Errorf("ollama: decode chat response: %w", err)
	}
	return result.Message.Content, nil
}
