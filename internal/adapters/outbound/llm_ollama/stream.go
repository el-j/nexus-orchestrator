package llm_ollama

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"nexus-orchestrator/internal/core/domain"
)

// maxLineBytes bounds one line of Ollama's newline-delimited JSON stream.
const maxLineBytes = 4 << 20

// ChatStream is Chat that also hands each piece of the reply to onChunk as it arrives
// (/api/chat with "stream": true). It returns the complete reply. A stream that ends before the
// "done" line was cut off and is an error.
func (a *Adapter) ChatStream(ctx context.Context, messages []domain.Message, onChunk func(string)) (string, error) {
	reqBody, err := json.Marshal(ollamaChatRequest{Model: a.model, Messages: messagesToMaps(messages), Stream: true})
	if err != nil {
		return "", fmt.Errorf("ollama: marshal chat request: %w", err)
	}
	resp, err := a.do(ctx, a.httpClient, http.MethodPost, "/api/chat", reqBody)
	if err != nil {
		return "", fmt.Errorf("ollama: chat request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama: chat: unexpected status %d", resp.StatusCode)
	}

	var reply strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64<<10), maxLineBytes)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "" {
			continue
		}
		var ev struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Done  bool   `json:"done"`
			Error string `json:"error"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			return reply.String(), fmt.Errorf("ollama: decode stream line: %w", err)
		}
		if ev.Error != "" {
			return reply.String(), fmt.Errorf("ollama: stream error: %s", ev.Error)
		}
		if ev.Message.Content != "" {
			reply.WriteString(ev.Message.Content)
			onChunk(ev.Message.Content)
		}
		if ev.Done {
			return reply.String(), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return reply.String(), fmt.Errorf("ollama: read stream: %w", err)
	}
	return reply.String(), fmt.Errorf("ollama: stream ended before done")
}
