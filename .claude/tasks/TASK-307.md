---
id: TASK-307
title: Extract buildProviders/buildProviderFromConfig to internal/bootstrap to eliminate dual-main duplication
role: backend
planId: PLAN-047
status: todo
dependencies: []
createdAt: 2026-03-25T00:00:00.000Z
---

## Context

`buildProviders()` and `buildProviderFromConfig()` are near-identical functions defined in both:

- Root `main.go` (Wails desktop app)
- `cmd/nexus-daemon/main.go`

Any change to provider construction (new provider kind, new default URL, new env var) must be made in both files. This has already caused the lmstudio `DefaultBaseURL` const to be ignored by both callers (they each hardcode the string independently).

Estimated duplication: ~60 lines in each file (~120 lines total).

## Files to Read

- `main.go` — find `buildProviders()` and `buildProviderFromConfig()` functions
- `cmd/nexus-daemon/main.go` — find the same functions, compare for differences
- `internal/adapters/outbound/llm_lmstudio/adapter.go` — DefaultBaseURL (from TASK-306)
- `internal/adapters/outbound/llm_ollama/adapter.go` — DefaultBaseURL

## Implementation Steps

1. **Create `internal/bootstrap/providers.go`**:

```go
// Package bootstrap provides shared provider construction logic used by
// all nexus entry points (Wails GUI, daemon, CLI).
package bootstrap

import (
    "nexus-orchestrator/internal/core/ports"
    // ... adapter imports
)

// BuildProviders constructs all default LLM provider clients from environment variables.
func BuildProviders() []ports.LLMClient { ... }

// BuildProviderFromConfig constructs a single LLMClient from a ProviderConfig record.
func BuildProviderFromConfig(cfg domain.ProviderConfig) (ports.LLMClient, error) { ... }
```

2. Copy the implementation from `cmd/nexus-daemon/main.go` (which is likely slightly more up-to-date). Ensure it:
   - Uses `llm_lmstudio.DefaultBaseURL` (from TASK-306) instead of hardcoded string
   - Uses `llm_ollama.DefaultBaseURL` instead of hardcoded string
   - Reads env vars for all configurable URLs

3. **In `main.go` (Wails)**: delete the local `buildProviders` and `buildProviderFromConfig` functions, import `bootstrap`, and call `bootstrap.BuildProviders()` and `bootstrap.BuildProviderFromConfig(cfg)`.

4. **In `cmd/nexus-daemon/main.go`**: same deletion + import + call.

5. If there are subtle differences between the two implementations, resolve them in the shared version (take the superset of features).

## Acceptance Criteria

- [ ] `internal/bootstrap/providers.go` exists with both functions
- [ ] `go vet ./internal/bootstrap/... ./cmd/nexus-daemon/... .` clean
- [ ] `go build ./cmd/nexus-daemon/... .` exits 0
- [ ] No `buildProviders` or `buildProviderFromConfig` functions remain in `main.go` or `cmd/nexus-daemon/main.go`
- [ ] `go test ./... -race -count=1` all pass
