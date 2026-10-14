package llm_lmstudio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"nexus-orchestrator/internal/adapters/outbound/sse"
	"nexus-orchestrator/internal/core/domain"
)

// maxEventBytes bounds one server-sent event line.
const maxEventBytes = 4 << 20

// ChatStream is Chat that also hands each piece of the reply to onChunk as it arrives
// (OpenAI "stream": true, server-sent events). It returns the complete reply. A stream that
// ends without [DONE] or a finish reason was cut off and is an error. A server that answers
// with a plain JSON completion instead of a stream is accepted: the reply arrives as one piece.
func (a *Adapter) ChatStream(ctx context.Context, messages []domain.Message, onChunk func(string)) (string, error) {
	reqBody, err := json.Marshal(chatCompletionRequest{
		Model:       a.activeModelOrDefault(ctx),
		Messages:    messagesToMaps(messages),
		Temperature: 0.2,
		Stream:      true,
	})
	if err != nil {
		return "", fmt.Errorf("lmstudio: marshal chat request: %w", err)
	}
	resp, err := a.do(ctx, http.MethodPost, a.baseURL+"/chat/completions", reqBody)
	if err != nil {
		return "", fmt.Errorf("lmstudio: chat request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("lmstudio: chat: unexpected status %d", resp.StatusCode)
	}
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		reply, err := decodeChat(resp.Body)
		if err == nil {
			onChunk(reply)
		}
		return reply, err
	}

	var reply strings.Builder
	finished := false
	err = sse.Read(resp.Body, maxEventBytes, func(ev sse.Event) error {
		if strings.TrimSpace(ev.Data) == "[DONE]" {
			finished = true
			return sse.ErrStop
		}
		var e struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(ev.Data), &e); err != nil {
			return fmt.Errorf("lmstudio: decode stream event: %w", err)
		}
		if e.Error != nil {
			return fmt.Errorf("lmstudio: stream error: %s", e.Error.Message)
		}
		if len(e.Choices) == 0 {
			return nil
		}
		if p := e.Choices[0].Delta.Content; p != "" {
			reply.WriteString(p)
			onChunk(p)
		}
		if e.Choices[0].FinishReason != nil {
			finished = true
		}
		return nil
	})
	if err != nil {
		return reply.String(), err
	}
	if !finished {
		return reply.String(), fmt.Errorf("lmstudio: stream ended before [DONE]")
	}
	return reply.String(), nil
}
