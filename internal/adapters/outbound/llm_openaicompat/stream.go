package llm_openaicompat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"nexus-orchestrator/internal/core/domain"
)

// maxEventBytes bounds one server-sent event line.
const maxEventBytes = 4 << 20

// ChatStream is Chat that also hands each piece of the reply to onChunk as it arrives
// (OpenAI "stream": true, server-sent events). It returns the complete reply. A stream that
// ends without [DONE] or a finish reason was cut off and is an error. A server that ignores
// "stream" and answers with a plain JSON completion is accepted: the reply arrives as one piece.
func (a *Adapter) ChatStream(ctx context.Context, messages []domain.Message, onChunk func(string)) (string, error) {
	reqBody, err := json.Marshal(chatCompletionRequest{
		Model:       a.model,
		Messages:    messagesToMaps(messages),
		Temperature: 0.2,
		Stream:      true,
	})
	if err != nil {
		return "", fmt.Errorf("%s: marshal request: %w", a.name, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("%s: build request: %w", a.name, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	a.setAuthHeader(req)
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s: request: %w", a.name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		return "", fmt.Errorf("%s: rate limited (429)", a.name)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s: unexpected status %d%s", a.name, resp.StatusCode, errorDetail(resp.Body))
	}

	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		// The server ignored "stream": true and sent a normal completion: deliver it as one piece.
		reply, err := a.decodeCompletion(resp.Body)
		if err == nil {
			onChunk(reply)
		}
		return reply, err
	}

	var reply strings.Builder
	finished := false
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64<<10), maxEventBytes)
	for scanner.Scan() {
		data, ok := strings.CutPrefix(scanner.Text(), "data:")
		if !ok { // blank separators and ": keep-alive" comments
			continue
		}
		data = strings.TrimSpace(data)
		if data == "[DONE]" {
			return reply.String(), nil
		}
		var ev struct {
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
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			return reply.String(), fmt.Errorf("%s: decode stream event: %w", a.name, err)
		}
		if ev.Error != nil {
			return reply.String(), fmt.Errorf("%s: stream error: %s", a.name, ev.Error.Message)
		}
		if len(ev.Choices) == 0 {
			continue
		}
		if piece := ev.Choices[0].Delta.Content; piece != "" {
			reply.WriteString(piece)
			onChunk(piece)
		}
		if ev.Choices[0].FinishReason != nil {
			finished = true
		}
	}
	if err := scanner.Err(); err != nil {
		return reply.String(), fmt.Errorf("%s: read stream: %w", a.name, err)
	}
	if !finished {
		return reply.String(), fmt.Errorf("%s: stream ended before [DONE]", a.name)
	}
	return reply.String(), nil
}
