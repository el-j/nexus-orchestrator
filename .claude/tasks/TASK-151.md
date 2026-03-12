---
id: TASK-151
title: Ports — AISessionRepository + AISessionMonitor interfaces
role: architecture
planId: PLAN-022
status: todo
dependencies: [TASK-150]
priority: critical
estimated_effort: S
createdAt: 2026-03-12T11:00:00.000Z
---

## Goal
Add two new port interfaces — `AISessionRepository` (outbound persistence) and `AISessionMonitor` (inbound discovery) — to `internal/core/ports/ports.go`, keeping the hexagonal dependency rule intact.

## Context
`internal/core/ports/ports.go` is the single file containing all port interface contracts. The two new interfaces follow the exact same pattern as the existing `SessionRepository` and `TaskRepository`:
- Pure Go interfaces, no concrete types, no framework imports
- `AISessionRepository` is outbound (SQLite will implement it)
- `AISessionMonitor` is inbound (VS Code extension / HTTP adapter will drive it via registration, not a persistent subscription — the monitor is a stateful adapter that the orchestrator calls to get the current snapshot)

> **Architecture decision (from design doc):** The "monitor" pattern here is a pull model — `ListActive()` returns the current live slice — rather than a callback subscription. This keeps core services free of goroutines and concurrent channel management. The inbound HTTP adapter writes sessions to `AISessionRepository` directly via the `Orchestrator` port methods (added in TASK-153). `AISessionMonitor` port is thus for **future** push-based adapters only; stub it now but leave implementation optional.

## Scope

### Files to modify
- `internal/core/ports/ports.go` — append two new interface blocks after `ProviderConfigRepository`

## Implementation Steps
1. Read the full current `internal/core/ports/ports.go` to understand existing style.
2. Append after the `ProviderConfigRepository` interface (at end of file):

   **`AISessionRepository`** — outbound port for persisting AISession entities:
   - `SaveAISession(ctx context.Context, s domain.AISession) error`
   - `GetAISessionByID(ctx context.Context, id string) (domain.AISession, error)` — returns `domain.ErrNotFound` wrapped error when missing
   - `ListAISessions(ctx context.Context) ([]domain.AISession, error)`
   - `UpdateAISessionStatus(ctx context.Context, id string, status domain.AISessionStatus, lastActivity time.Time) error`
   - `DeleteAISession(ctx context.Context, id string) error`

   **`AISessionMonitor`** — optional inbound port; adapters that can push external session events implement this:
   - `RegisterSession(s domain.AISession) error` — called by an external adapter when a new session is detected
   - `ListActive() ([]domain.AISession, error)` — returns currently known non-disconnected sessions

3. Ensure `context.Context` is imported (already used by `ProviderConfigRepository`).
4. Add a brief `// AISessionRepository ...` doc comment above each interface, matching the style of existing comments.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `AISessionRepository` interface is exported and has all 5 methods listed above
- [ ] `AISessionMonitor` interface is exported and has 2 methods
- [ ] No concrete types, no adapter imports, no framework imports in ports.go
- [ ] Existing interfaces are not modified

## Anti-patterns to Avoid
- NEVER add methods to the `Orchestrator` inbound port here — that is TASK-153's scope
- NEVER create a constructor or struct in ports.go — interfaces only
- NEVER use `interface{}` or untyped parameters
- NEVER import anything from `internal/adapters/`
