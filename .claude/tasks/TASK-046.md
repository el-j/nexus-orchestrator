---
id: TASK-046
title: "Ollama adapter: implement ActiveModel()"
role: backend
planId: PLAN-005
status: todo
dependencies: [TASK-044]
createdAt: 2026-03-10T06:00:00.000Z
---

## Context
`LLMClient.ActiveModel() string` was added to the port in TASK-044. Ollama is configured
with a specific model at construction time (`a.model`), so `ActiveModel()` is trivial.
Also ensure `GetAvailableModels()` results are cached for routing use.

## Files to Read
- `internal/adapters/outbound/llm_ollama/adapter.go`
- `internal/core/ports/ports.go` (after TASK-044)

## Implementation Steps

### 1. Implement `ActiveModel() string`
```go
func (a *Adapter) ActiveModel() string { return a.model }
```
The `model` field is set in `NewOllamaAdapter` and never changes.

### 2. Add model cache fields for routing
```go
availableModels   []string
availableOnce     sync.Once
```
These allow `DiscoveryService.FindForModel()` to check model availability without repeated network calls.

### 3. Refactor `GetAvailableModels()` to use the cache
```go
func (a *Adapter) GetAvailableModels() ([]string, error) {
    var getErr error
    a.availableOnce.Do(func() {
        resp, err := a.httpClient.Get(a.baseURL + "/api/tags")
        if err != nil {
            getErr = fmt.Errorf("ollama: list models: %w", err)
            return
        }
        defer resp.Body.Close()
        var result struct {
            Models []struct {
                Name string `json:"name"`
            } `json:"models"`
        }
        if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
            getErr = fmt.Errorf("ollama: decode models: %w", err)
            return
        }
        names := make([]string, 0, len(result.Models))
        for _, m := range result.Models {
            names = append(names, m.Name)
        }
        a.availableModels = names
    })
    if getErr != nil {
        return nil, getErr
    }
    return a.availableModels, nil
}
```

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `go build ./...` exits 0
- [ ] `ActiveModel()` returns the configured model name
- [ ] `GetAvailableModels()` caches results via `sync.Once`
- [ ] `Adapter` struct has `availableModels []string` and `availableOnce sync.Once`

## Anti-patterns to Avoid
- NEVER use goroutines inside adapter methods
- NEVER ignore the error from `sync.Once` — capture it via closure variable
