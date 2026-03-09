---
id: TASK-018
title: Wails binding — extend app.go with GetAllTasks / GetSession / ClearSession / GetStats
role: backend
planId: PLAN-002
status: todo
dependencies: [TASK-015]
createdAt: 2026-03-09T12:00:00.000Z
---

## Context

`app.go` exposes Wails bindings to the JavaScript frontend. Currently it has 6 methods: `Greet`, `SubmitTask`, `GetTask`, `GetQueue`, `GetProviders`, `CancelTask`. The GUI views (TASK-019 through TASK-022) need history access, session management, and dashboard stats. These methods must be added to `app.go` before the frontend views are implemented.

There is a duplicate `internal/adapters/inbound/wailsbind/bind.go` that is never registered in any entry point — it should be deleted in this task to avoid confusion.

## Files to Read

- `app.go` — full file (existing struct definition + all 6 current methods)
- `internal/core/ports/ports.go` — all port interfaces (TaskRepository, SessionRepository, Orchestrator)
- `internal/core/domain/task.go` — Task struct
- `internal/core/domain/session.go` — Session + Message types
- `internal/adapters/inbound/wailsbind/bind.go` — confirm it is unused before deleting

## Implementation Steps

1. **Verify `wailsbind/bind.go` is unused**: search for any import of `nexus-ai/internal/adapters/inbound/wailsbind` in all Go files. If confirmed unused, delete the file.

2. **Add `sessionRepo ports.SessionRepository` field** to the `App` struct in `app.go`:
   ```go
   type App struct {
       ctx         context.Context
       orch        ports.Orchestrator
       sessionRepo ports.SessionRepository
       taskRepo    ports.TaskRepository
   }
   ```
   Update `NewApp(orch, sessionRepo, taskRepo)` constructor accordingly. Wire it in `main.go`.

3. **Add `GetAllTasks(status string) []domain.Task`**:
   ```go
   func (a *App) GetAllTasks(status string) []domain.Task {
       tasks, err := a.taskRepo.GetAll(status)
       if err != nil {
           log.Printf("app: GetAllTasks: %v", err)
           return nil
       }
       return tasks
   }
   ```

4. **Add `GetSession(projectPath string) []domain.Message`**:
   ```go
   func (a *App) GetSession(projectPath string) []domain.Message {
       sess, err := a.sessionRepo.GetOrCreate(filepath.Clean(projectPath))
       if err != nil {
           log.Printf("app: GetSession: %v", err)
           return nil
       }
       return sess.Messages
   }
   ```
   Returns empty slice (not nil) when no messages exist — important for JSON serialisation to `[]` not `null`.

5. **Add `ClearSession(projectPath string) error`**:
   ```go
   func (a *App) ClearSession(projectPath string) error {
       if err := a.sessionRepo.Delete(filepath.Clean(projectPath)); err != nil {
           return fmt.Errorf("app: ClearSession: %w", err)
       }
       return nil
   }
   ```
   Uses the `Delete` method added to `SessionRepository` port in TASK-015.

6. **Define `StatsResponse` type** (in `app.go` or a new `internal/core/domain/stats.go`):
   ```go
   type StatsResponse struct {
       QueueDepth    int    `json:"queueDepth"`
       ActiveTask    string `json:"activeTask"`    // task ID or ""
       ProviderCount int    `json:"providerCount"`
   }
   ```

7. **Add `GetStats() StatsResponse`**:
   ```go
   func (a *App) GetStats() StatsResponse {
       queue := a.orch.GetQueue()
       providers, _ := a.orch.GetProviders()   // DiscoveryService.GetProviders (already on Orchestrator interface)
       return StatsResponse{
           QueueDepth:    len(queue),
           ProviderCount: len(providers),
       }
   }
   ```
   Note: If `ports.Orchestrator` does not expose `GetProviders()`, add it or use a separate `DiscoveryService` field on `App`.

8. **Update `main.go`** to pass the new dependencies to `NewApp(...)`.

9. **Update `frontend/src/wailsjs/go/main/App.d.ts`** (Vue 3 project TypeScript declarations — uses typed imports from `domain.ts` rather than `any`):
   ```typescript
   export function SubmitTask(projectPath: string, prompt: string): Promise<string>
   export function GetTask(id: string): Promise<import('../../../types/domain').Task>
   export function GetQueue(): Promise<import('../../../types/domain').Task[]>
   export function GetAllTasks(status: string): Promise<import('../../../types/domain').Task[]>
   export function GetProviders(): Promise<import('../../../types/domain').Provider[]>
   export function CancelTask(id: string): Promise<void>
   export function GetSession(projectPath: string): Promise<import('../../../types/domain').Message[]>
   export function ClearSession(projectPath: string): Promise<void>
   export function GetStats(): Promise<import('../../../types/domain').Stats>
   ```
   Note: The actual hand-written declaration file is created in TASK-017 step 12. This step exists to confirm the Go method signatures are in sync with the Vue 3 frontend types in `src/types/domain.ts`.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build .` (Wails binary) exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `app.go` has `GetAllTasks`, `GetSession`, `ClearSession`, `GetStats` methods
- [ ] `wailsbind/bind.go` is deleted (confirmed unused)
- [ ] `StatsResponse` is JSON-serializable with camelCase keys
- [ ] `GetSession` returns `[]domain.Message{}` (not nil) when no messages exist

## Anti-patterns to Avoid

- NEVER import `internal/adapters/outbound/...` directly from `app.go` — accept ports interfaces only
- NEVER call `os.Exit` in Wails binding methods — return errors or zero values
- NEVER add auth or session cookie logic to the Wails binding
- NEVER return Go error types directly to JS — Wails serialises errors as strings; return nil or log internally
