---
id: TASK-040
title: "LM Studio adapter: implement ContextLimit() by querying /v1/models"
role: backend
planId: PLAN-004
status: todo
dependencies: [TASK-039]
createdAt: 2026-03-10T00:00:00.000Z
---

## Context
`LLMClient.ContextLimit()` must return the context-window size of the model currently loaded in LM Studio.  LM Studio exposes this via its `/v1/models` endpoint in the `context_length` field of each model object.

## Files to Read
- `internal/core/ports/ports.go` (for the interface contract)
- `internal/adapters/outbound/llm_lmstudio/adapter.go`

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
           resp, err := a.httpClient.Get(a.baseURL + "/models")
           if err != nil {
               return
           }
           defer resp.Body.Close()
           var result struct {
               Data []struct {
                   ContextLength int `json:"context_length"`
               } `json:"data"`
           }
           if json.NewDecoder(resp.Body).Decode(&result) != nil || len(result.Data) == 0 {
               return
           }
           a.contextLimit = result.Data[0].ContextLength
       })
       return a.contextLimit
   }
   ```

3. Ensure `sync` and `encoding/json` are in the import block (both should already be present or are needed).

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./...` exits 0
- [ ] `Adapter` struct has `contextLimit int` and `limitOnce sync.Once` fields
- [ ] `ContextLimit()` calls `/v1/models` at most once (cached via sync.Once)
- [ ] Returns 0 on any HTTP or JSON error (safe fallback — no pre-flight check when unknown)

## Anti-patterns to Avoid
- NEVER use `sync.Mutex` + bool flag when `sync.Once` is cleaner
- NEVER panic on HTTP errors — return 0 silently
- NEVER cache a stale negative result (0 on failure is safe; Once fires only once per Adapter lifetime)
