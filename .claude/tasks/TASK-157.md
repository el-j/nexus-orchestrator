---
id: TASK-157
title: Entry point wiring — AISessionRepo in daemon and app.go
role: devops
planId: PLAN-022
status: todo
dependencies: [TASK-152, TASK-153, TASK-154]
priority: high
estimated_effort: S
createdAt: 2026-03-12T11:00:00.000Z
---

## Goal
Wire the new `AISessionRepo` and the updated `OrchestratorService.RegisterAISession/ListAISessions/DeregisterAISession` methods into all three entry points: `cmd/nexus-daemon/main.go`, `main.go` (Wails desktop), and `cmd/nexus-cli/main.go` (if it creates a daemon locally — it doesn't, it's a remote client only, so skip).

## Context
The project has three entry points:

| Binary | File | Role |
|--------|------|------|
| Desktop GUI | `main.go` + `app.go` | Wails window + embedded HTTP + MCP on `:9998` |
| Headless daemon | `cmd/nexus-daemon/main.go` | HTTP `:9999` + MCP `:9998` |
| CLI client | `cmd/nexus-cli/main.go` | Thin HTTP client, no local services — **no changes needed** |

Every time a new outbound adapter is added, ALL non-CLI entry points must be updated. The wiring order is always: construct repo → construct service → pass to HTTP server + MCP server.

The `OrchestratorService` currently is constructed with `NewOrchestrator(discovery, repo, writer, sessionRepo)`. After TASK-153, a setter method `SetAISessionRepo(r ports.AISessionRepository)` (or equivalent mechanism — follow whatever pattern was chosen in TASK-153) exists to inject the `AISessionRepo`.

## Scope

### Files to modify
- `cmd/nexus-daemon/main.go`
- `main.go` (Wails desktop entry point) — and/or `app.go` if service construction is there

## Implementation Steps

### 1. Read both entry points fully before making any changes
- Read `cmd/nexus-daemon/main.go` (full file)
- Read `main.go` lines 1–100 and `app.go` lines 1–100 to understand where `NewOrchestrator` is called and what repos are already wired

### 2. cmd/nexus-daemon/main.go
After the line where `repo_sqlite.NewSessionRepo(r)` or `repo_sqlite.NewProviderConfigRepo(r)` is constructed:
1. Construct `aiSessionRepo := repo_sqlite.NewAISessionRepo(r)`
2. Call `orch.SetAISessionRepo(aiSessionRepo)` (or pass to constructor — follow how it was implemented in TASK-153)

### 3. main.go / app.go (Wails desktop)
Same pattern: where other repos are constructed, also construct `repo_sqlite.NewAISessionRepo(r)` and inject via `SetAISessionRepo`.

### 4. Verify the Orchestrator interface contract
After wiring, verify that `*OrchestratorService` still satisfies `ports.Orchestrator`:
- Add or verify a compile-time assertion `var _ ports.Orchestrator = (*services.OrchestratorService)(nil)` exists in either the service file or a test file
- If this assertion already exists in a test file, ensure it still compiles

### 5. Smoke test
Run these commands and confirm they exit 0:
```sh
CGO_ENABLED=1 go vet ./...
CGO_ENABLED=1 go build ./cmd/nexus-cli/...
CGO_ENABLED=1 go build ./cmd/nexus-daemon/...
CGO_ENABLED=1 go test -race -count=1 ./...
```

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `cmd/nexus-daemon/main.go` constructs `AISessionRepo` and injects it into the orchestrator
- [ ] `main.go`/`app.go` (Wails) does the same
- [ ] `cmd/nexus-cli/main.go` is NOT modified (remote-only client)
- [ ] `POST /api/ai-sessions` returns 201 (not 500) when the daemon is started after this wiring

## Anti-patterns to Avoid
- NEVER create a goroutine in main.go for session monitoring — the HTTP adapter's inbound requests handle this
- NEVER duplicate the `AISessionRepo` construction — construct once, share the `*sql.DB`
- NEVER change the `NewOrchestrator` constructor signature if a setter pattern was chosen in TASK-153 (be consistent)
- NEVER modify `cmd/nexus-cli/main.go` — it is a pure remote HTTP client
