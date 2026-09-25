// Package llm_gemini implements the LLMClient port for the Google Gemini REST API.
package llm_gemini

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
	defaultBaseURL     = "https://generativelanguage.googleapis.com"
	defaultModel       = "gemini-2.5-pro"
	defaultHTTPTimeout = 300 * time.Second
)

// Adapter implements ports.LLMClient for the Google Gemini REST API.
type Adapter struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewAdapter creates a new Google Gemini LLM adapter.
func NewAdapter(apiKey, model string, baseURL ...string) *Adapter {
	bURL := defaultBaseURL
	if len(baseURL) > 0 && strings.TrimSpace(baseURL[0]) != "" {
		bURL = strings.TrimRight(baseURL[0], "/")
	}
	m := model
	if strings.TrimSpace(m) == "" {
		m = defaultModel
	}
	return &Adapter{
		baseURL:    bURL,
		apiKey:     apiKey,
		model:      m,
		httpClient: &http.Client{Timeout: defaultHTTPTimeout},
	}
}

// geminiContextLimits maps known Gemini model IDs to their context window sizes.
var geminiContextLimits = map[string]int{
	"gemini-2.5-pro":   2097152,
	"gemini-2.5-flash": 1048576,
	"gemini-1.5-pro":   2097152,
	"gemini-1.5-flash": 1048576,
	"gemini-1.0-pro":   32768,
}

func (a *Adapter) ProviderName() string { return "Google Gemini" }
func (a *Adapter) ActiveModel() string  { return a.model }
func (a *Adapter) BaseURL() string      { return a.baseURL }

// ContextLimit returns the maximum input token count for the configured Gemini model.
func (a *Adapter) ContextLimit() int {
	if limit, ok := geminiContextLimits[a.model]; ok {
		return limit
	}
	return 1048576
}

// Ping checks whether the Gemini endpoint is reachable.
func (a *Adapter) Ping() bool {
	req, err := a.newRequest(http.MethodGet, a.baseURL+"/v1beta/models", nil)
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

// GetAvailableModels lists models supporting generateContent from the Gemini API.
func (a *Adapter) GetAvailableModels() ([]string, error) {
	req, err := a.newRequest(http.MethodGet, a.baseURL+"/v1beta/models", nil)
	if err != nil {
		return nil, fmt.Errorf("gemini: build models request: %w", err)
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini: list models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini: list models: unexpected status %d", resp.StatusCode)
	}

	var result struct {
		Models []struct {
			Name                       string   `json:"name"`
			SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
		} `json:"models"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("gemini: decode models response: %w", err)
	}

	var modelIDs []string
	for _, m := range result.Models {
		supportsGenerate := false
		for _, method := range m.SupportedGenerationMethods {
			if method == "generateContent" {
				supportsGenerate = true
				break
			}
		}
		if supportsGenerate {
			cleanName := strings.TrimPrefix(m.Name, "models/")
			modelIDs = append(modelIDs, cleanName)
		}
	}

	if len(modelIDs) == 0 {
		return []string{"gemini-2.5-pro", "gemini-2.5-flash", "gemini-1.5-pro", "gemini-1.5-flash"}, nil
	}
	return modelIDs, nil
}

// GenerateCode sends a prompt as a single user message to Gemini.
func (a *Adapter) GenerateCode(prompt string) (string, error) {
	return a.Chat([]domain.Message{{Role: domain.RoleUser, Content: prompt}})
}

// Chat sends a multi-turn conversation to the Gemini generateContent endpoint.
func (a *Adapter) Chat(messages []domain.Message) (string, error) {
	reqBody, err := a.buildGenerateRequest(messages)
	if err != nil {
		return "", fmt.Errorf("gemini: marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", a.baseURL, a.model)
	req, err := a.newRequest(http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("gemini: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return "", fmt.Errorf("gemini: rate limited (429)")
	}
	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("gemini: unexpected status %d: %s", resp.StatusCode, string(respBytes))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
				Role string `json:"role"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&result); err != nil {
		return "", fmt.Errorf("gemini: decode response: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: empty candidate response")
	}

	var sb strings.Builder
	for _, part := range result.Candidates[0].Content.Parts {
		sb.WriteString(part.Text)
	}
	return sb.String(), nil
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiSystemInstruction struct {
	Parts []geminiPart `json:"parts"`
}

type geminiGenerateRequest struct {
	Contents          []geminiContent          `json:"contents"`
	SystemInstruction *geminiSystemInstruction `json:"systemInstruction,omitempty"`
}

func (a *Adapter) buildGenerateRequest(messages []domain.Message) ([]byte, error) {
	var systemParts []geminiPart
	var contents []geminiContent

	for _, m := range messages {
		if m.Role == domain.RoleSystem {
			systemParts = append(systemParts, geminiPart{Text: m.Content})
			continue
		}

		role := "user"
		if m.Role == domain.RoleAssistant {
			role = "model"
		}

		// Merge consecutive turns with the same role
		if len(contents) > 0 && contents[len(contents)-1].Role == role {
			lastIdx := len(contents) - 1
			contents[lastIdx].Parts = append(contents[lastIdx].Parts, geminiPart{Text: m.Content})
		} else {
			contents = append(contents, geminiContent{
				Role:  role,
				Parts: []geminiPart{{Text: m.Content}},
			})
		}
	}

	req := geminiGenerateRequest{
		Contents: contents,
	}
	if len(systemParts) > 0 {
		req.SystemInstruction = &geminiSystemInstruction{Parts: systemParts}
	}

	return json.Marshal(req)
}

func (a *Adapter) newRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if a.apiKey != "" {
		req.Header.Set("x-goog-api-key", a.apiKey)
	}
	return req, nil
}
