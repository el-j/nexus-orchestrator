---
id: TASK-544
planId: PLAN-070
title: 'Regenerate App.d.ts + models.ts; add 4 Wails App brain method bindings'
role: frontend
status: todo
createdAt: 2026-04-14T02:00:00Z
---

# TASK-544 — Wails bindings: regenerate + add missing brain methods

## Context

Two critical issues with Wails-generated TypeScript files:

### Issue 1: `App.d.ts` is stale

`frontend/src/wailsjs/go/main/App.d.ts` was never regenerated after brain service was added.
Missing: `GetBrainStatus`, `IngestKnowledge`, `GetProjectContext`, `GetFocusedContext`,
`SearchKnowledge`. Still exports `WithActivityService` which was unexported in TASK-531.
File is used by Wails build tooling for type validation.

### Issue 2: `models.ts` types dates as `any`

`frontend/src/wailsjs/go/models.ts` types all `time.Time` fields as `any`. Stale codegen.

### Issue 3: 4 brain `App` methods not bound

`app.go` does NOT have Wails-bound methods for: `InitProject`, `ListKnowledge`,
`DeleteKnowledge`, `GetFileMap`. The desktop frontend has no path to manage the knowledge store.

## Work Required

### Step 1 — Add 4 Wails bindings to `app.go`

Add these methods following the existing `GetBrainStatus`/`IngestKnowledge` pattern:

```go
func (a *App) InitProject(projectPath, claudeMDPath string) (domain.BrainStatus, error)
func (a *App) ListKnowledge(projectPath, kind string) ([]domain.ProjectKnowledge, error)
func (a *App) DeleteKnowledge(id string) error
func (a *App) GetFileMap(projectPath, focusArea string) ([]string, error)
```

All must guard `if a.brainSvc == nil`.

### Step 2 — Regenerate Wails bindings

Run `wails generate module` (or the project-specific equivalent) to regenerate `App.d.ts` and
`models.ts`. If `wails` CLI is not available, manually update `App.d.ts` to:

- Add missing brain method declarations
- Remove `WithActivityService` / `WithBrainService`
- Add the 4 new methods from Step 1

For `models.ts`: update `time.Time` field types from `any` to `string` (Wails serialises
`time.Time` as ISO 8601 strings).

### Step 3 — Update `wails.ts` wrapper functions

Add wrapper functions in `frontend/src/types/wails.ts` for the 4 new Wails App methods,
following the existing `getBrainStatus` / `ingestKnowledge` pattern.

## File Targets

- `app.go`
- `frontend/src/wailsjs/go/main/App.d.ts`
- `frontend/src/wailsjs/go/models.ts`
- `frontend/src/types/wails.ts`

## Acceptance Criteria

- `CGO_ENABLED=1 go build ./...` clean
- `cd frontend && vue-tsc --noEmit` clean (no stale binding type errors)
- `App.d.ts` contains `GetBrainStatus`, `IngestKnowledge`, `InitProject`, `ListKnowledge`,
  `DeleteKnowledge`, `GetFileMap`, `GetProjectContext`, `GetFocusedContext`, `SearchKnowledge`
- `App.d.ts` does NOT contain `WithActivityService` or `WithBrainService`
- `models.ts` date fields typed as `string` not `any`
