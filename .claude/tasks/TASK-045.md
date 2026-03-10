---
id: TASK-045
title: "LM Studio adapter: fix ContextLimit() via native API + implement ActiveModel()"
role: backend
planId: PLAN-005
status: todo
dependencies: [TASK-044]
createdAt: 2026-03-10T06:00:00.000Z
---

## Context
LM Studio's OpenAI-compat `/v1/models` endpoint does NOT include `context_length` (the user confirmed
the response only has `id`, `object`, `owned_by`).  LM Studio's native REST API exposes a
`GET /api/v0/model` endpoint that returns `{"identifier":"...","contextLength":4096,...}`.
Also implement `ActiveModel()` to expose which model is currently loaded.

## Files to Read
- `internal/adapters/outbound/llm_lmstudio/adapter.go`
- `internal/core/ports/ports.go` (after TASK-044 is applied)

## Implementation Steps

### 1. Derive the native base URL from `baseURL`
The adapter is constructed with `baseURL = "http://127.0.0.1:1234/v1"`.
Strip the `/v1` suffix to get the native base:
```go
import "strings"
nativeBase := strings.TrimSuffix(baseURL, "/v1")
```
Store `nativeBase string` as a field on `Adapter`.

### 2. Add `activeModel string` + `modelOnce sync.Once` fields to `Adapter`
```go
activeModel string
modelOnce   sync.Once
```
These cache the active model name (separate Once from contextLimit).

### 3. Implement `ActiveModel() string`
```go
func (a *Adapter) ActiveModel() string {
    a.modelOnce.Do(func() {
        resp, err := a.httpClient.Get(a.nativeBase + "/api/v0/model")
        if err != nil {
            // fall back to first model from OpenAI-compat endpoint
            models, err2 := a.GetAvailableModels()
            if err2 == nil && len(models) > 0 {
                a.activeModel = models[0]
            }
            return
        }
        defer resp.Body.Close()
        var result struct {
            Identifier string `json:"identifier"`
        }
        if json.NewDecoder(resp.Body).Decode(&result) == nil {
            a.activeModel = result.Identifier
        }
    })
    return a.activeModel
}
```

### 4. Fix `ContextLimit()` to use the native endpoint
Replace the existing `limitOnce.Do` body:
```go
func (a *Adapter) ContextLimit() int {
    a.limitOnce.Do(func() {
        resp, err := a.httpClient.Get(a.nativeBase + "/api/v0/model")
        if err != nil {
            return
        }
        defer resp.Body.Close()
        var result struct {
            ContextLength int `json:"contextLength"`
        }
        if json.NewDecoder(resp.Body).Decode(&result) == nil && result.ContextLength > 0 {
            a.contextLimit = result.ContextLength
        }
    })
    return a.contextLimit
}
```

### 5. Update `NewLMStudioAdapter` to store `nativeBase`
```go
func NewLMStudioAdapter(baseURL string) *Adapter {
    return &Adapter{
        baseURL:    baseURL,
        nativeBase: strings.TrimSuffix(baseURL, "/v1"),
        httpClient: &http.Client{Timeout: 300 * time.Second},
    }
}
```

### 6. Add `"strings"` to imports if not already present.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `go build ./...` exits 0
- [ ] `ActiveModel()` is implemented and returns a non-empty string when LM Studio is reachable
- [ ] `ContextLimit()` queries `/api/v0/model` (not `/v1/models`) for context length
- [ ] `Adapter` struct has `nativeBase string`, `activeModel string`, `modelOnce sync.Once` fields
- [ ] Fallback: if native endpoint fails, `ActiveModel()` tries standard `/v1/models`

## Anti-patterns to Avoid
- NEVER use separate HTTP clients — share `a.httpClient`
- NEVER reset `sync.Once` — these fields are lifetime caches per Adapter instance
- NEVER panic on HTTP errors
