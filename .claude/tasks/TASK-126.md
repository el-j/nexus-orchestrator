---
id: TASK-126
title: Configurable LM Studio base URL
role: backend
planId: PLAN-018
status: todo
dependencies: []
createdAt: 2026-03-11T16:10:00.000Z
---

## Context
LM Studio is hardcoded to `http://127.0.0.1:1234/v1`. Users may run it on a different port or machine. The adapter must accept a configurable base URL via `NEXUS_LMSTUDIO_URL`.

## Files to Read
- `internal/adapters/outbound/llm_lmstudio/adapter.go`
- `main.go` (buildProviders function)
- `cmd/nexus-daemon/main.go`

## Implementation Steps
1. Add configurable `baseURL` to the LM Studio adapter (similar pattern to Ollama).
2. Read `NEXUS_LMSTUDIO_URL` env var in both entry points.
3. Pass the URL to the adapter constructor; fall back to `http://127.0.0.1:1234/v1` when unset.
4. Verify all methods (`Ping`, `GetAvailableModels`, `Chat`, `GenerateCode`) use the field.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] Setting `NEXUS_LMSTUDIO_URL=http://10.0.0.5:1234/v1` makes the adapter use that URL
- [ ] Default behaviour unchanged when env var is unset

## Anti-patterns to Avoid
- NEVER import adapters from core services
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
