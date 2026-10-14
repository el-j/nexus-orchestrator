package llm_gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"nexus-orchestrator/internal/adapters/outbound/sse"
	"nexus-orchestrator/internal/core/domain"
)

// maxEventBytes bounds one server-sent event line.
const maxEventBytes = 4 << 20

// ChatStream is Chat that also hands each piece of the reply to onChunk as it arrives
// (streamGenerateContent with alt=sse). It returns the complete reply. A stream that ends
// without a finishReason was cut off and is an error. A server that answers with a plain
// JSON reply instead of a stream is accepted: the reply arrives as one piece.
func (a *Adapter) ChatStream(ctx context.Context, messages []domain.Message, onChunk func(string)) (string, error) {
	reqBody, err := a.buildGenerateRequest(messages)
	if err != nil {
		return "", fmt.Errorf("gemini: marshal request: %w", err)
	}
	url := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse", a.baseURL, a.model)
	req, err := a.newRequest(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
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
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		reply, err := decodeCandidate(resp.Body)
		if err == nil {
			onChunk(reply)
		}
		return reply, err
	}

	var reply strings.Builder
	finished := false
	err = sse.Read(resp.Body, maxEventBytes, func(ev sse.Event) error {
		var chunk geminiCandidates
		if err := json.Unmarshal([]byte(ev.Data), &chunk); err != nil {
			return fmt.Errorf("gemini: decode stream event: %w", err)
		}
		if chunk.Error != nil {
			return fmt.Errorf("gemini: stream error: %s", chunk.Error.Message)
		}
		if len(chunk.Candidates) == 0 {
			return nil
		}
		for _, p := range chunk.Candidates[0].Content.Parts {
			if p.Text != "" {
				reply.WriteString(p.Text)
				onChunk(p.Text)
			}
		}
		if chunk.Candidates[0].FinishReason != "" {
			finished = true
		}
		return nil
	})
	if err != nil {
		return reply.String(), err
	}
	if !finished {
		return reply.String(), fmt.Errorf("gemini: stream ended before finishReason")
	}
	return reply.String(), nil
}
