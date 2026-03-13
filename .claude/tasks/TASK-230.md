---
id: TASK-230
title: Initialize nil slices in SQLite repo methods to prevent null JSON
role: backend
planId: PLAN-032
status: todo
dependencies: []
createdAt: 2026-03-13T00:30:00.000Z
---

## Context
Go's `encoding/json` marshals nil slices (`var x []T`) as `null`, not `[]`. When Wails sends these nil-marshaled results to the frontend, Vue refs get set to `null` and subsequent `.map()` / `.filter()` calls crash with `null is not an object`. Initializing slices with `make()` or literal `[]T{}` ensures they marshal to `[]`.

## Files to Read
- `internal/adapters/outbound/repo_sqlite/repo.go` (lines around 166, 228 — `GetPending`, `GetAllTasks`)
- `internal/adapters/outbound/repo_sqlite/ai_session_repo.go` (line 96 — `ListAISessions`)
- `internal/adapters/outbound/repo_sqlite/provider_config_repo.go` (line 77 — `ListProviderConfigs`)

## Implementation Steps
1. In `repo.go` ~ line 166: change `var tasks []domain.Task` to `tasks := []domain.Task{}`
2. In `repo.go` ~ line 228: change `var tasks []domain.Task` to `tasks := []domain.Task{}`
3. In `ai_session_repo.go` ~ line 96: change `var sessions []domain.AISession` to `sessions := []domain.AISession{}`
4. In `provider_config_repo.go` ~ line 77: change `var configs []domain.ProviderConfig` to `configs := []domain.ProviderConfig{}`
5. Run `go vet ./...` and `CGO_ENABLED=1 go test -race -count=1 ./internal/adapters/outbound/repo_sqlite/...`

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `GetPending()` returns `[]` (empty JSON array) not `null` when no tasks exist
- [ ] `ListAISessions()` returns `[]` not `null` when no sessions exist
- [ ] `ListProviderConfigs()` returns `[]` not `null` when no configs exist

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
