package llm_anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"nexus-orchestrator/internal/adapters/outbound/httperr"
	"strings"
	"time"

	"nexus-orchestrator/internal/adapters/outbound/sse"
	"nexus-orchestrator/internal/core/domain"
)

// maxEventBytes bounds one server-sent event line.
const maxEventBytes = 4 << 20

// ChatStream is Chat that also hands each piece of the reply to onChunk as it arrives
// (Messages API "stream": true). It returns the complete reply. A stream that ends before
// message_stop was cut off and is an error. A server that answers with a plain JSON message
// instead of a stream is accepted: the reply arrives as one piece.
func (a *Adapter) ChatStream(ctx context.Context, messages []domain.Message, onChunk func(string)) (string, error) {
	system, turns := toAnthropicMessages(messages)
	if len(turns) == 0 {
		return "", fmt.Errorf("anthropic: no user message to send")
	}
	reqBody, err := json.Marshal(anthropicRequest{Model: a.model, MaxTokens: defaultMaxTokens, System: systemBlocks(system), Messages: turns, Stream: true})
	if err != nil {
		return "", fmt.Errorf("anthropic: marshal request: %w", err)
	}
	req, err := a.newRequest(ctx, http.MethodPost, a.baseURL+"/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("anthropic: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic: request: %w", err)
	}
	defer resp.Body.Close()
	if err := httperr.FromResponse("anthropic", resp, time.Now()); err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("anthropic: unexpected status %d%s", resp.StatusCode, errorDetail(resp.Body))
	}
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		reply, err := a.decodeMessage(resp.Body)
		if err == nil {
			onChunk(reply)
		}
		return reply, err
	}

	var reply strings.Builder
	finished := false
	usage := domain.Usage{Model: a.model}
	err = sse.Read(resp.Body, maxEventBytes, func(ev sse.Event) error {
		var e struct {
			Type    string `json:"type"`
			Message struct {
				Model string     `json:"model"`
				Usage tokenUsage `json:"usage"`
			} `json:"message"`
			Usage tokenUsage `json:"usage"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(ev.Data), &e); err != nil {
			return fmt.Errorf("anthropic: decode stream event: %w", err)
		}
		switch e.Type {
		case "message_start": // the input side; the output side arrives with message_delta
			usage = domain.Usage{Model: firstNonEmpty(e.Message.Model, a.model), InputTokens: e.Message.Usage.InputTokens,
				OutputTokens: e.Message.Usage.OutputTokens, CachedTokens: e.Message.Usage.CacheCreated + e.Message.Usage.CacheRead}
		case "message_delta":
			usage.OutputTokens = e.Usage.OutputTokens
		case "content_block_delta":
			if e.Delta.Type == "text_delta" && e.Delta.Text != "" {
				reply.WriteString(e.Delta.Text)
				onChunk(e.Delta.Text)
			}
		case "error":
			return fmt.Errorf("anthropic: stream error: %s", e.Error.Message)
		case "message_stop":
			finished = true
			return sse.ErrStop
		}
		return nil
	})
	a.Set(usage)
	if err != nil {
		return reply.String(), err
	}
	if !finished {
		return reply.String(), fmt.Errorf("anthropic: stream ended before message_stop")
	}
	return reply.String(), nil
}

// tokenUsage is the usage block of a Messages API event.
type tokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	CacheCreated int `json:"cache_creation_input_tokens"`
	CacheRead    int `json:"cache_read_input_tokens"`
}
