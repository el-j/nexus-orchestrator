package llm_openaicompat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"nexus-orchestrator/internal/adapters/outbound/httperr"
	"strings"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// maxEventBytes bounds one server-sent event line.
const maxEventBytes = 4 << 20

// ChatStream is Chat that also hands each piece of the reply to onChunk as it arrives
// (OpenAI "stream": true, server-sent events). It returns the complete reply. A stream that
// ends without [DONE] or a finish reason was cut off and is an error. A server that ignores
// "stream" and answers with a plain JSON completion is accepted: the reply arrives as one piece.
func (a *Adapter) ChatStream(ctx context.Context, messages []domain.Message, onChunk func(string)) (string, error) {
	resp, err := a.openStream(ctx, messages)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

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
	usage := domain.Usage{Model: a.model}
	defer func() { a.Set(usage) }()
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
		var ev streamEvent
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			return reply.String(), fmt.Errorf("%s: decode stream event: %w", a.name, err)
		}
		if ev.Error != nil {
			return reply.String(), a.streamError(ev.Error.Status, ev.Error.Message)
		}
		if ev.Usage != nil {
			usage = ev.Usage.toDomain(firstNonEmpty(ev.Model, a.model))
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

// openStream posts the streaming request and returns the response when the server accepted it; the
// caller closes the body. A refusal comes back as the typed provider error.
func (a *Adapter) openStream(ctx context.Context, messages []domain.Message) (*http.Response, error) {
	reqBody, _ := json.Marshal(chatCompletionRequest{
		Model:         a.model,
		Messages:      messagesToMaps(messages),
		Temperature:   0.2,
		Stream:        true,
		StreamOptions: &streamOptions{IncludeUsage: true},
	}) // a request of strings and numbers always encodes
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("%s: build request: %w", a.name, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	a.setAuthHeader(req)
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: request: %w", a.name, err)
	}
	if err := httperr.FromResponse(a.name, resp, time.Now()); err != nil {
		resp.Body.Close()
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, fmt.Errorf("%s: unexpected status %d%s", a.name, resp.StatusCode, errorDetail(resp.Body))
	}
	return resp, nil
}

// streamEvent is the part of a server-sent chunk the adapter reads.
type streamEvent struct {
	Model   string      `json:"model"`
	Usage   *usageBlock `json:"usage"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Status  int    `json:"status"`
	} `json:"error"`
}

// streamError turns an error event of the stream into the typed provider error. A status of 0 is read
// from the message: a quota, rate limit or 429 in it counts as 429.
func (a *Adapter) streamError(status int, message string) error {
	if status == 0 {
		low := strings.ToLower(message)
		if strings.Contains(low, "quota") || strings.Contains(low, "rate limit") || strings.Contains(low, "429") {
			status = http.StatusTooManyRequests
		}
	}
	if pe := domain.ClassifyHTTP(a.name, status, "", message, time.Now()); pe != nil {
		pe.Msg = fmt.Sprintf("%s: %s", pe.Msg, message)
		return pe
	}
	return fmt.Errorf("%s: stream error: %s", a.name, message)
}
