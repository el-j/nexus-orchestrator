// Package gateway implements an OpenAI-compatible HTTP reverse proxy and LLM gateway.
// It routes chat completions through Nexus's model ladder, provider cooldowns,
// rate-limit handlers, and token budgets.
package gateway

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ChatCompletionRequest is the incoming payload for /v1/chat/completions.
type ChatCompletionRequest struct {
	Model         string        `json:"model"`
	Messages      []ChatMessage `json:"messages"`
	Stream        bool          `json:"stream,omitempty"`
	Temperature   *float64      `json:"temperature,omitempty"`
	TopP          *float64      `json:"top_p,omitempty"`
	MaxTokens     *int          `json:"max_tokens,omitempty"`
	User          string        `json:"user,omitempty"`
	HistoryBudget int           `json:"history_budget,omitempty"`
}

// UnmarshalJSON reads a message whose content is a string, null or a list of typed parts, as OpenAI
// clients send it; the text parts are joined with newlines and other parts (images) are ignored.
func (m *ChatMessage) UnmarshalJSON(data []byte) error {
	var raw struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
		Name    string          `json:"name"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	m.Role, m.Name = raw.Role, raw.Name
	var text string
	if err := json.Unmarshal(raw.Content, &text); err == nil || len(raw.Content) == 0 || string(raw.Content) == "null" {
		m.Content = text
		return nil
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw.Content, &parts); err != nil {
		return fmt.Errorf("message content must be a string or a list of parts: %w", err)
	}
	var texts []string
	for _, p := range parts {
		if p.Type == "text" {
			texts = append(texts, p.Text)
		}
	}
	m.Content = strings.Join(texts, "\n")
	return nil
}

// ChatMessage represents a single turn in a conversation.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// ChatCompletionResponse is the standard OpenAI non-streaming JSON response.
type ChatCompletionResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []ChatChoice `json:"choices"`
	Usage   *UsageInfo   `json:"usage,omitempty"`
}

// ChatChoice is one completion choice in a non-streaming response.
type ChatChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// ChatCompletionChunk is an SSE chunk emitted when stream is true.
type ChatCompletionChunk struct {
	ID      string            `json:"id"`
	Object  string            `json:"object"`
	Created int64             `json:"created"`
	Model   string            `json:"model"`
	Choices []ChatChunkChoice `json:"choices"`
}

// ChatChunkChoice is one choice inside an SSE chunk.
type ChatChunkChoice struct {
	Index        int            `json:"index"`
	Delta        ChatChunkDelta `json:"delta"`
	FinishReason *string        `json:"finish_reason"`
}

// ChatChunkDelta carries the streamed delta chunk.
type ChatChunkDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// UsageInfo reports token usage metrics.
type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ModelList reports available models for GET /v1/models.
type ModelList struct {
	Object string      `json:"object"`
	Data   []ModelItem `json:"data"`
}

// ModelItem represents a model descriptor in /v1/models.
type ModelItem struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// APIError represents an OpenAI-style error payload.
type APIError struct {
	Error ErrorDetails `json:"error"`
}

// ErrorDetails carries the error message, type, and code.
type ErrorDetails struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}
