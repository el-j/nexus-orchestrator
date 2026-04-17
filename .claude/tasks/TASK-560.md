---
id: TASK-560
title: Fix LLM context limits and delete dead tray stub package
role: backend
planId: PLAN-071
status: todo
dependencies: [TASK-557]
createdAt: 2026-04-15T00:00:00Z
---

## Context

Two LLM adapter correctness bugs: `llm_anthropic/adapter.go` is missing claude-4, claude-sonnet-4-5, and claude-opus-4-5 from `claudeContextLimits` so those models fall through to a zero value. `llm_openaicompat/adapter.go` returns a flat 131072 for every model regardless of what is configured, making context-budget decisions meaningless for GPT-4o, Mistral, etc. Additionally, the entire `internal/adapters/inbound/tray/` package is dead code: `Start()` no-ops, `Enabled()` always returns false, `UpdateStatus()` discards its output. It should be deleted until a real implementation is ready.

## Files to Read

- `internal/adapters/outbound/llm_anthropic/adapter.go`
- `internal/adapters/outbound/llm_openaicompat/adapter.go`
- `internal/adapters/inbound/tray/tray.go`
- `internal/adapters/inbound/tray/icon.go`
- `cmd/nexus-daemon/main.go` (check tray usage)

## Implementation Steps

1. In `llm_anthropic/adapter.go` — extend `claudeContextLimits` to include current model IDs:
   - `"claude-opus-4-5"`: 200000, `"claude-sonnet-4-5"`: 200000
   - `"claude-opus-4-6"`: 200000, `"claude-sonnet-4-6"`: 200000
   - `"claude-haiku-4-5-20251001"`: 200000
   - Add a default fallback of 200000 for any unrecognised `claude-*` prefix
2. In `llm_openaicompat/adapter.go` — replace the flat `131072` with a map of known models (gpt-4o: 128000, gpt-4o-mini: 128000, gpt-4: 8192, gpt-4-turbo: 128000, mistral-\*: 32768, etc.) with a fallback of 32768 for unknown models
3. Check `cmd/nexus-daemon/main.go` and `app.go` for tray references. Remove any wiring that instantiates `tray.TrayAdapter`.
4. Delete `internal/adapters/inbound/tray/tray.go` and `internal/adapters/inbound/tray/icon.go` (and the directory if empty)
5. If any interface or port references the tray, remove them too

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `internal/adapters/inbound/tray/` directory no longer exists
- [ ] `claudeContextLimits` includes claude-4-5 and claude-4-6 series models
- [ ] `llm_openaicompat.ContextLimit()` returns model-specific values, not flat 131072

## Anti-patterns to Avoid

- NEVER delete files that are referenced by an import — check all usages first
- NEVER add goroutines inside `internal/core/services/`
