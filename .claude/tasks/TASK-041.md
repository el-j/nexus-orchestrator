---
id: TASK-041
title: "Ollama adapter: implement ContextLimit() by querying /api/show"
role: backend
planId: PLAN-004
status: todo
dependencies: [TASK-039]
createdAt: 2026-03-10T00:00:00.000Z
---

## Context
`LLMClient.ContextLimit()` must return the context-window size of the model loaded in Ollama.  Ollama exposes model metadata via `POST /api/show` with `{"name": "<model>"}`.  The response contains `model_info` with a `"llama.context_length"` key (int), or alternatively `parameters` as a string with `num_ctx <N>` lines.

## Files to Read
- `internal/core/ports/ports.go` (for the interface contract)
- `internal/adapters/outbound/llm_ollama/adapter.go`

## Implementation Steps
1. Add two fields to the `Adapter` struct:
   ```go
   contextLimit int        // cached value; 0 = unknown
   limitOnce    sync.Once  // ensures the network query runs at most once
   ```
   Add `"sync"` to the import block if not already present.

2. Implement `ContextLimit() int`:
   ```go
   func (a *Adapter) ContextLimit() int {
       a.limitOnce.Do(func() {
           body, err := json.Marshal(map[string]string{"name": a.model})
           if err != nil {
               return
           }
           resp, err := a.httpClient.Post(a.baseURL+"/api/show", "application/json", bytes.NewReader(body))
           if err != nil {
               return
           }
           defer resp.Body.Close()
           var result struct {
               ModelInfo map[string]interface{} `json:"model_info"`
           }
           if json.NewDecoder(resp.Body).Decode(&result) != nil {
               return
           }
           if v, ok := result.ModelInfo["llama.context_length"]; ok {
               switch n := v.(type) {
               case float64:
                   a.contextLimit = int(n)
               case int:
                   a.contextLimit = n
               }
           }
       })
       return a.contextLimit
   }
   ```
   `bytes` and `encoding/json` are already imported.

3. Ensure `"sync"` is in the import block.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./...` exits 0
- [ ] `Adapter` struct has `contextLimit int` and `limitOnce sync.Once` fields
- [ ] `ContextLimit()` calls `/api/show` at most once (cached via sync.Once)
- [ ] Reads `model_info["llama.context_length"]` (float64 from JSON unmarshalled to int)
- [ ] Returns 0 on any HTTP or JSON error (safe fallback)

## Anti-patterns to Avoid
- NEVER panic on HTTP/JSON errors — return 0 silently
- NEVER use `o.mu.Lock()` inside `limitOnce.Do()` — `sync.Once` is already concurrency-safe
