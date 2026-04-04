---
id: TASK-306
title: LLM adapter dedup — export DefaultBaseURL, merge lmstudio Once guards, extract message-conversion helper
role: backend
planId: PLAN-047
status: todo
dependencies: []
createdAt: 2026-03-25T00:00:00.000Z
---

## Context

Three duplication issues found in the LLM adapter packages:

**Issue 1 — lmstudio defaultBaseURL unused const**
`const defaultBaseURL = "http://127.0.0.1:1234/v1"` is package-private and never read by callers — both `main.go` and `cmd/nexus-daemon/main.go` hardcode the string. Ollama already exports `DefaultBaseURL`; lmstudio should do the same.

**Issue 2 — double HTTP call in lmstudio ActiveModel + ContextLimit**
`ActiveModel()` and `ContextLimit()` each have their own `sync.Once` guard making separate GET requests to `/api/v0/model`. The response contains both `identifier` and `contextLength` — only one request is needed.

**Issue 3 — message-conversion loop duplicated in 3 adapters**
The `domain.Message → map[string]string` conversion is copy-pasted in:

- `llm_lmstudio/adapter.go`
- `llm_ollama/adapter.go`
- `llm_openaicompat/adapter.go`

## Files to Read

- `internal/adapters/outbound/llm_lmstudio/adapter.go` — full file
- `internal/adapters/outbound/llm_ollama/adapter.go` — Chat() method
- `internal/adapters/outbound/llm_openaicompat/adapter.go` — Chat() method
- `internal/adapters/outbound/llm_ollama/adapter.go` line ~18 — how Ollama exports DefaultBaseURL

## Implementation Steps

### Fix 1 — Export DefaultBaseURL

In `llm_lmstudio/adapter.go`:

```go
// DefaultBaseURL is the default LM Studio API endpoint.
const DefaultBaseURL = "http://127.0.0.1:1234/v1"
```

Remove or keep the old `defaultBaseURL` const — if it was used anywhere internally, replace those usages.

Do NOT update callers (main.go files) in this task — that is handled by TASK-307 which also refactors the provider builder.

### Fix 2 — Merge Once guards

In `llm_lmstudio/adapter.go`:

1. Remove `modelOnce sync.Once` and `limitOnce sync.Once` fields from the adapter struct.
2. Add a single `infoOnce sync.Once` field.
3. Add private fields `activeModel string` and `contextLimit int` if not already present.
4. Create a single private method `fetchModelInfo()` that calls `/api/v0/model` once and populates both `a.activeModel` and `a.contextLimit`.
5. Have both `ActiveModel()` and `ContextLimit()` call `a.infoOnce.Do(a.fetchModelInfo)` before returning their respective field.

### Fix 3 — Extract message-conversion helper

In a shared location within each adapter package (or as a package-level function in a new `internal/adapters/outbound/llmutil/` package):

Option A (simpler, preferred): Add a package-level helper in `llm_openaicompat/adapter.go` (already has a helper pattern) and import from others — but this creates an import dependency.

Option B (cleaner): Add to each adapter package separately as a private helper `messagesToMaps`:

```go
func messagesToMaps(msgs []domain.Message) []map[string]string {
    out := make([]map[string]string, len(msgs))
    for i, m := range msgs {
        out[i] = map[string]string{"role": string(m.Role), "content": m.Content}
    }
    return out
}
```

Add this function to llm_lmstudio, llm_ollama, and llm_openaicompat, then replace the inline loops in each `Chat()` with a call to `messagesToMaps(messages)`.

Use Option B to avoid cross-package dependencies between sibling adapters.

## Acceptance Criteria

- [ ] `go vet ./internal/adapters/outbound/llm_lmstudio/... ./internal/adapters/outbound/llm_ollama/... ./internal/adapters/outbound/llm_openaicompat/...` clean
- [ ] `go test ./internal/adapters/outbound/... -race -count=1` all pass
- [ ] `llm_lmstudio.DefaultBaseURL` is exported (capital D)
- [ ] `lmstudio.ActiveModel()` and `lmstudio.ContextLimit()` share a single `sync.Once`
- [ ] No inline `msgs[i] = map[string]string{...}` loops remain in any Chat() method
