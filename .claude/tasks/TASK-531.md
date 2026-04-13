---
id: TASK-531
planId: PLAN-069
title: 'Fix app.go GetFocusedContext bug and unexport With*Service setters'
role: backend
status: todo
createdAt: 2026-04-13T16:00:00Z
---

# TASK-531 — Fix app.go Wails binding bugs

## Context

`app.go` (root, Wails desktop app bindings) has two confirmed issues:

### Bug 1: GetFocusedContext calls wrong method

```go
// GetFocusedContext searches the brain context relative to a specific reasoning query.
func (a *App) GetFocusedContext(projectPath, question string, maxTokens int) (domain.ContextResponse, error) {
    if a.brainSvc == nil { ... }
    return a.brainSvc.GetContext(context.Background(), domain.ContextQuery{  // ← WRONG: should be GetFocusedContext
```

The frontend binding calls `GetFocusedContext` on the App, but internally it delegates to
`brainSvc.GetContext` (the macro context method) instead of `brainSvc.GetFocusedContext`.
This means the `Question` field is never used — focused context always returns the full macro
budget response regardless of the search question.

### Bug 2: Exported With\*Service setters exposed to Wails

`WithBrainService` and `WithActivityService` are exported methods on `*App`, meaning they appear
in Wails-generated `App.d.ts` TypeScript bindings and are callable from the frontend. These are
internal setup/DI methods and should be unexported.

## Work Required

1. `app.go` line ~298: Change `a.brainSvc.GetContext(...)` → `a.brainSvc.GetFocusedContext(...)`.
   Ensure the `ContextQuery{Question: question}` field is passed correctly.

2. Rename `WithBrainService` → `withBrainService` and `WithActivityService` → `withActivityService`.
   Update the single call site in `main.go` accordingly.

## File Targets

- `app.go`
- `main.go`

## Acceptance Criteria

- `GetFocusedContext` correctly delegates to `brainSvc.GetFocusedContext`
- `WithBrainService` and `WithActivityService` are unexported (lowercase `w`)
- `go vet ./...` and `CGO_ENABLED=1 go build ./...` clean
