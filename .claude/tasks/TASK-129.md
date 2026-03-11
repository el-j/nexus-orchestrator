---
id: TASK-129
title: Persist provider configs in SQLite
role: backend
planId: PLAN-018
status: todo
dependencies: []
createdAt: 2026-03-11T16:10:00.000Z
---

## Context
Cloud providers (OpenAI, Anthropic, GitHub) are configured only via env vars at startup. Users need to add/edit/remove providers at runtime and have those survive restarts. A `provider_configs` table in SQLite will store name, type, base URL, API key (encrypted or plaintext), and default model.

## Files to Read
- `internal/core/domain/provider.go`
- `internal/adapters/outbound/repo_sqlite/repo.go`
- `internal/core/ports/ports.go`
- `internal/core/services/orchestrator.go` (RegisterCloudProvider)

## Implementation Steps
1. Define a `ProviderConfig` domain type in `domain/provider.go` (if not already present) with fields: `ID`, `Name`, `Type` (lmstudio/ollama/openai/anthropic/openaicompat), `BaseURL`, `APIKey`, `DefaultModel`, `Enabled`, `CreatedAt`, `UpdatedAt`.
2. Add a `ProviderConfigRepository` port interface with `SaveProviderConfig`, `ListProviderConfigs`, `DeleteProviderConfig`.
3. Create a `provider_configs` table in the SQLite repo with a migration in `repo.go`'s init.
4. Implement the port in `repo_sqlite`.
5. On app startup, load saved configs and register them alongside env-var providers.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] Provider configs round-trip through SQLite (save → restart → load)
- [ ] API keys stored; consider noting in docs that local SQLite is the trust boundary

## Anti-patterns to Avoid
- NEVER import adapters from core services
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER store API keys in plain-text logs
