---
id: TASK-557
title: Fix App architecture — ActivityService concrete→port interface, constructor panics, httpapi_client context
role: backend
planId: PLAN-071
status: todo
dependencies: [TASK-554, TASK-555]
createdAt: 2026-04-15T00:00:00Z
---

## Context

Three architectural violations exist. (1) `app.go` holds `*services.ActivityService` (concrete type), importing the core services package from the Wails entry point and breaking the hexagonal port boundary. (2) Three `panic()` calls in `orchestrator.go`'s constructor are in-domain panics that belong in the wiring layer. (3) `httpapi_client/client.go` uses `context.Background()` internally for 14+ port methods, silently dropping caller cancellation signals.

## Files to Read

- `app.go`
- `internal/core/ports/ports.go`
- `internal/core/services/orchestrator.go` (constructor section)
- `internal/adapters/outbound/httpapi_client/client.go`
- `cmd/nexus-daemon/main.go`

## Implementation Steps

1. In `internal/core/ports/ports.go`: add an `ActivityReader` port interface with the methods that `app.go` calls on `activitySvc` (e.g., `GetRecentActivities`, `GetTimeline`)
2. In `app.go`: change `activitySvc *services.ActivityService` field and `withActivityService(*services.ActivityService)` to use the new `ports.ActivityReader` interface
3. In `cmd/nexus-daemon/main.go` (and any other wiring file): ensure the concrete `*services.ActivityService` is passed as `ports.ActivityReader` at construction — no change to service code needed
4. In `internal/core/services/orchestrator.go` constructor: replace the three `panic()` calls with early-return errors: change the function signature to return `(*OrchestratorService, error)` and return `fmt.Errorf("orchestrator: new: %w", ErrMissingDependency)` for nil deps
5. Update all callers of the constructor in `cmd/nexus-daemon/main.go` and entry points to handle the returned error
6. In `httpapi_client/client.go`: for each of the 14+ non-context methods that call `context.Background()`, check if the port interface method accepts a `context.Context`. For those that don't (older port methods), add the context to the method signature in the port interface and propagate it through — OR, if changing the port is too invasive, at minimum add a `// TODO: port interface needs ctx` comment and accept the `context.Background()` fallback for now. Choose the less invasive approach that doesn't break existing callers.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/... ./cmd/nexus-mcp-stdio/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `app.go` no longer imports `nexus-orchestrator/internal/core/services`
- [ ] `app.go` `activitySvc` field is typed as `ports.ActivityReader`
- [ ] `OrchestratorService` constructor returns `error` instead of panicking
- [ ] All constructor callers in `cmd/` handle the error return

## Anti-patterns to Avoid

- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/`
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
