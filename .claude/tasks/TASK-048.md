---
id: TASK-048
title: "New llm_anthropic adapter: Claude (Anthropic native API)"
role: backend
planId: PLAN-005
status: todo
dependencies: [TASK-044]
createdAt: 2026-03-10T06:00:00.000Z
---

## Context
Anthropic's Claude uses a native API (`/v1/messages`) that is NOT OpenAI-compatible.
Request format, headers, and response shape are all different.  This adapter wraps the
Anthropic Messages API so Claude models can participate in the routing system.

## Files to Read
- `internal/core/ports/ports.go` (full interface after TASK-044)
- `internal/core/domain/session.go` (for domain.Message)

## Implementation Steps

Create `internal/adapters/outbound/llm_anthropic/adapter.go`:

```go
package llm_anthropic

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "nexus-orchestrator/internal/core/domain"
)

const (
    defaultBaseURL        = "https://api.anthropic.com"
    anthropicVersion      = "2023-06-01"
    defaultMaxTokens      = 4096
)

// Adapter implements ports.LLMClient for the Anthropic Claude API.
type Adapter struct {
    baseURL    string
    apiKey     string
    model      string // e.g. "claude-sonnet-4-5", "claude-3-5-haiku-latest"
    httpClient *http.Client
}

// NewAdapter creates an Anthropic Claude adapter.
//   apiKey — Anthropic API key (required)
//   model  — model ID (e.g. "claude-opus-4-5", "claude-sonnet-4-5")
func NewAdapter(apiKey, model string) *Adapter {
    return &Adapter{
        baseURL: defaultBaseURL,
        apiKey:  apiKey,
        model:   model,
        httpClient: &http.Client{
            Timeout: 300 * time.Second,
        },
    }
}

func (a *Adapter) ProviderName() string { return "Anthropic" }
func (a *Adapter) ActiveModel() string  { return a.model }
func (a *Adapter) ContextLimit() int    { return 0 } // not exposed via API; user-known

// Ping checks reachability by calling the models endpoint.
func (a *Adapter) Ping() bool {
    req, err := a.newRequest(http.MethodGet, a.baseURL+"/v1/models", nil)
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

// GetAvailableModels returns known Claude model IDs.
func (a *Adapter) GetAvailableModels() ([]string, error) {
    req, err := a.newRequest(http.MethodGet, a.baseURL+"/v1/models", nil)
    if err != nil {
        return nil, fmt.Errorf("anthropic: build models request: %w", err)
    }
    resp, err := a.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("anthropic: list models: %w", err)
    }
    defer resp.Body.Close()

    var result struct {
        Data []struct {
            ID string `json:"id"`
        } `json:"data"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("anthropic: decode models: %w", err)
    }
    ids := make([]string, 0, len(result.Data))
    for _, m := range result.Data {
        ids = append(ids, m.ID)
    }
    return ids, nil
}

// GenerateCode sends a single prompt request to Claude.
func (a *Adapter) GenerateCode(prompt string) (string, error) {
    return a.sendMessages([]anthropicMessage{{Role: "user", Content: prompt}})
}

// Chat converts the multi-turn history into Anthropic message format.
// Anthropic requires alternating user/assistant turns; consecutive same-role
// messages are merged with a newline separator.
func (a *Adapter) Chat(messages []domain.Message) (string, error) {
    anthropicMsgs := toAnthropicMessages(messages)
    return a.sendMessages(anthropicMsgs)
}

type anthropicMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

// toAnthropicMessages converts domain messages, merging consecutive same-role turns.
func toAnthropicMessages(msgs []domain.Message) []anthropicMessage {
    if len(msgs) == 0 {
        return nil
    }
    out := make([]anthropicMessage, 0, len(msgs))
    for _, m := range msgs {
        role := m.Role
        if role != "user" && role != "assistant" {
            continue // skip system/other roles (handle separately if needed)
        }
        if len(out) > 0 && out[len(out)-1].Role == role {
            out[len(out)-1].Content += "\n" + m.Content
        } else {
            out = append(out, anthropicMessage{Role: role, Content: m.Content})
        }
    }
    return out
}

func (a *Adapter) sendMessages(messages []anthropicMessage) (string, error) {
    reqBody, err := json.Marshal(map[string]interface{}{
        "model":      a.model,
        "max_tokens": defaultMaxTokens,
        "messages":   messages,
    })
    if err != nil {
        return "", fmt.Errorf("anthropic: marshal request: %w", err)
    }

    req, err := a.newRequest(http.MethodPost, a.baseURL+"/v1/messages", bytes.NewReader(reqBody))
    if err != nil {
        return "", fmt.Errorf("anthropic: build request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := a.httpClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("anthropic: request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusTooManyRequests {
        return "", fmt.Errorf("anthropic: rate limited (429)")
    }
    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("anthropic: unexpected status %d", resp.StatusCode)
    }

    var result struct {
        Content []struct {
            Type string `json:"type"`
            Text string `json:"text"`
        } `json:"content"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", fmt.Errorf("anthropic: decode response: %w", err)
    }
    for _, block := range result.Content {
        if block.Type == "text" {
            return block.Text, nil
        }
    }
    return "", fmt.Errorf("anthropic: no text content in response")
}

func (a *Adapter) newRequest(method, url string, body interface{ Read([]byte) (int, error) }) (*http.Request, error) {
    var req *http.Request
    var err error
    if body != nil {
        req, err = http.NewRequest(method, url, body.(interface {
            Read(p []byte) (n int, err error)
        }))
    } else {
        req, err = http.NewRequest(method, url, nil)
    }
    if err != nil {
        return nil, err
    }
    req.Header.Set("x-api-key", a.apiKey)
    req.Header.Set("anthropic-version", anthropicVersion)
    return req, nil
}
```

**Note on `newRequest`**: The `body` parameter should be `io.Reader` not a custom interface.
Simplify to:
```go
import "io"
func (a *Adapter) newRequest(method, url string, body io.Reader) (*http.Request, error) {
    req, err := http.NewRequest(method, url, body)
    if err != nil {
        return nil, err
    }
    req.Header.Set("x-api-key", a.apiKey)
    req.Header.Set("anthropic-version", anthropicVersion)
    return req, nil
}
```
And call as `a.newRequest(http.MethodPost, url, bytes.NewReader(reqBody))` and `a.newRequest(http.MethodGet, url, nil)`.

## Acceptance Criteria
- [ ] File `internal/adapters/outbound/llm_anthropic/adapter.go` created
- [ ] `go vet ./...` exits 0
- [ ] `go build ./...` exits 0
- [ ] `Adapter` implements full `ports.LLMClient` interface
- [ ] HTTP 429 returns error containing "rate limited"
- [ ] Auth uses `x-api-key` + `anthropic-version` headers (NOT `Authorization: Bearer`)
- [ ] `Chat()` merges consecutive same-role messages (Anthropic requires alternating turns)
- [ ] `io.Reader` used for request body, not custom interface

## Anti-patterns to Avoid
- NEVER log the API key
- NEVER use `Authorization: Bearer` for Anthropic (they use `x-api-key`)
- NEVER assume OpenAI-compatible response shape
