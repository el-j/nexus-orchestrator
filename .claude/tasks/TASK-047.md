---
id: TASK-047
title: "New llm_openaicompat adapter: generic OpenAI-compatible provider"
role: backend
planId: PLAN-005
status: todo
dependencies: [TASK-044]
createdAt: 2026-03-10T06:00:00.000Z
---

## Context
Support any OpenAI-compatible cloud provider: OpenAI, GitHub Copilot, Azure OpenAI, OpenRouter, etc.
A single generic adapter configured with (name, baseURL, apiKey, model) covers all of these.

## Files to Read
- `internal/adapters/outbound/llm_lmstudio/adapter.go` (reference implementation)
- `internal/core/ports/ports.go` (full interface after TASK-044)
- `internal/core/domain/session.go` (for domain.Message)

## Implementation Steps

Create `internal/adapters/outbound/llm_openaicompat/adapter.go`:

```go
package llm_openaicompat

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "sync"
    "time"

    "nexus-orchestrator/internal/core/domain"
)

// Adapter implements ports.LLMClient for any OpenAI-compatible REST API.
// Use this for OpenAI, GitHub Copilot, Azure OpenAI, OpenRouter, etc.
type Adapter struct {
    name       string
    baseURL    string
    apiKey     string
    model      string
    httpClient *http.Client

    availableModels []string
    availableOnce   sync.Once
}

// NewAdapter creates a generic OpenAI-compatible adapter.
//   name    — human-readable provider name (e.g. "OpenAI", "GitHub Copilot")
//   baseURL — API base without trailing slash (e.g. "https://api.openai.com/v1")
//   apiKey  — Bearer token (empty string = no auth header sent)
//   model   — default model to use (e.g. "gpt-4o", "gpt-4-turbo")
func NewAdapter(name, baseURL, apiKey, model string) *Adapter {
    return &Adapter{
        name:    name,
        baseURL: baseURL,
        apiKey:  apiKey,
        model:   model,
        httpClient: &http.Client{
            Timeout: 300 * time.Second,
        },
    }
}

func (a *Adapter) ProviderName() string { return a.name }
func (a *Adapter) ActiveModel() string  { return a.model }
func (a *Adapter) ContextLimit() int    { return 0 } // unknown for generic providers

func (a *Adapter) Ping() bool {
    req, err := http.NewRequest(http.MethodGet, a.baseURL+"/models", nil)
    if err != nil {
        return false
    }
    a.setAuthHeader(req)
    resp, err := a.httpClient.Do(req)
    if err != nil {
        return false
    }
    defer resp.Body.Close()
    return resp.StatusCode == http.StatusOK
}

func (a *Adapter) GetAvailableModels() ([]string, error) {
    var getErr error
    a.availableOnce.Do(func() {
        req, err := http.NewRequest(http.MethodGet, a.baseURL+"/models", nil)
        if err != nil {
            getErr = fmt.Errorf("%s: build models request: %w", a.name, err)
            return
        }
        a.setAuthHeader(req)
        resp, err := a.httpClient.Do(req)
        if err != nil {
            getErr = fmt.Errorf("%s: list models: %w", a.name, err)
            return
        }
        defer resp.Body.Close()
        var result struct {
            Data []struct {
                ID string `json:"id"`
            } `json:"data"`
        }
        if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
            getErr = fmt.Errorf("%s: decode models: %w", a.name, err)
            return
        }
        ids := make([]string, 0, len(result.Data))
        for _, m := range result.Data {
            ids = append(ids, m.ID)
        }
        a.availableModels = ids
    })
    return a.availableModels, getErr
}

func (a *Adapter) GenerateCode(prompt string) (string, error) {
    return a.chat([]map[string]string{{"role": "user", "content": prompt}})
}

func (a *Adapter) Chat(messages []domain.Message) (string, error) {
    msgs := make([]map[string]string, len(messages))
    for i, m := range messages {
        msgs[i] = map[string]string{"role": m.Role, "content": m.Content}
    }
    return a.chat(msgs)
}

func (a *Adapter) chat(messages []map[string]string) (string, error) {
    reqBody, err := json.Marshal(map[string]interface{}{
        "model":       a.model,
        "messages":    messages,
        "temperature": 0.2,
    })
    if err != nil {
        return "", fmt.Errorf("%s: marshal request: %w", a.name, err)
    }
    req, err := http.NewRequest(http.MethodPost, a.baseURL+"/chat/completions", bytes.NewReader(reqBody))
    if err != nil {
        return "", fmt.Errorf("%s: build request: %w", a.name, err)
    }
    req.Header.Set("Content-Type", "application/json")
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
        return "", fmt.Errorf("%s: unexpected status %d", a.name, resp.StatusCode)
    }

    var result struct {
        Choices []struct {
            Message struct {
                Content string `json:"content"`
            } `json:"message"`
        } `json:"choices"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", fmt.Errorf("%s: decode response: %w", a.name, err)
    }
    if len(result.Choices) == 0 {
        return "", fmt.Errorf("%s: no choices in response", a.name)
    }
    return result.Choices[0].Message.Content, nil
}

func (a *Adapter) setAuthHeader(req *http.Request) {
    if a.apiKey != "" {
        req.Header.Set("Authorization", "Bearer "+a.apiKey)
    }
}
```

## Acceptance Criteria
- [ ] File `internal/adapters/outbound/llm_openaicompat/adapter.go` created
- [ ] `go vet ./...` exits 0
- [ ] `go build ./...` exits 0
- [ ] `Adapter` implements full `ports.LLMClient` interface (`Ping`, `ProviderName`, `ActiveModel`, `GetAvailableModels`, `ContextLimit`, `GenerateCode`, `Chat`)
- [ ] HTTP 429 response returns a typed error containing "rate limited"
- [ ] API key is sent as `Authorization: Bearer <key>`; empty key skips the header
- [ ] `GetAvailableModels()` is cached via `sync.Once`

## Anti-patterns to Avoid
- NEVER store the API key in logs
- NEVER hardcode model names in the adapter; always use `a.model`
- NEVER use `sync.Mutex` + bool when `sync.Once` is cleaner
