package embedding_client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"nexus-orchestrator/internal/core/ports"
)

var _ ports.EmbeddingProvider = (*Client)(nil)

// Client generates embeddings using an OpenAI-compatible /v1/embeddings HTTP endpoint
// such as Ollama, LM Studio, or cloud providers.
type Client struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// Config configures the embedding client.
type Config struct {
	BaseURL    string
	Model      string
	HTTPClient *http.Client
	Timeout    time.Duration
}

// NewClient returns a new embedding Client for the given configuration.
func NewClient(cfg Config) *Client {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	model := cfg.Model
	if model == "" {
		model = "nomic-embed-text"
	}
	hc := cfg.HTTPClient
	if hc == nil {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = 5 * time.Second
		}
		hc = &http.Client{Timeout: timeout}
	}
	return &Client{
		baseURL:    baseURL,
		model:      model,
		httpClient: hc,
	}
}

type embedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Embed generates an embedding vector for the provided text.
func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("embedding_client: empty text")
	}

	url := c.baseURL + "/v1/embeddings"
	payload, _ := json.Marshal(embedRequest{
		Model: c.model,
		Input: text,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("embedding_client: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding_client: do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("embedding_client: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding_client: server returned status %d: %s", resp.StatusCode, string(body))
	}

	var res embedResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("embedding_client: unmarshal response: %w", err)
	}

	if res.Error != nil && res.Error.Message != "" {
		return nil, fmt.Errorf("embedding_client: API error: %s", res.Error.Message)
	}

	if len(res.Data) == 0 || len(res.Data[0].Embedding) == 0 {
		return nil, errors.New("embedding_client: no embedding returned in response")
	}

	return res.Data[0].Embedding, nil
}
