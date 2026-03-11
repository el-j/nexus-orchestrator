---
id: TASK-125
title: Configurable Ollama base URL
role: backend
planId: PLAN-018
status: todo
dependencies: []
createdAt: 2026-03-11T16:10:00.000Z
---

## Context
Ollama is hardcoded to `http://127.0.0.1:11434` but users run it in Docker containers with mapped ports or on remote hosts. The adapter must accept a configurable base URL via env var (`NEXUS_OLLAMA_URL`) and via runtime registration.

## Files to Read
- `internal/adapters/outbound/llm_ollama/adapter.go`
- `internal/core/services/discovery.go`
- `main.go` (buildProviders function)
- `cmd/nexus-daemon/main.go`

## Implementation Steps
1. Add a `baseURL` field to the Ollama adapter struct (if not already present) and a `NewWithURL(baseURL string)` constructor or an option to `New()`.
2. In `main.go` and `cmd/nexus-daemon/main.go`, read `NEXUS_OLLAMA_URL` env var. If set, use it instead of the default.
3. Ensure `Ping()`, `GetAvailableModels()`, `Chat()`, and `GenerateCode()` all use the configurable `baseURL`.
4. Update the existing constructor call sites to pass the env-var value.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] Setting `NEXUS_OLLAMA_URL=http://192.168.1.50:11434` makes the adapter ping that host
- [ ] Default behaviour unchanged when env var is unset

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/`
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
