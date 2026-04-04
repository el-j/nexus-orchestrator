---
id: TASK-293
title: 'Fix domain.ts Task type — add retryCount + sync wails.ts'
role: frontend
planId: PLAN-045
status: todo
dependencies: []
createdAt: 2026-03-14T18:00:00.000Z
---

## Context

The Go backend `domain.Task` struct sends `retryCount` in JSON responses (field exists
since TASK-014, confirmed in `wailsjs/wailsjs/go/models.ts` line ~95). The hand-maintained
`frontend/src/types/domain.ts` Task interface is missing this field, so TypeScript
does not know about it when components use task data.

Additionally, `frontend/src/types/wails.ts` Window.go declaration is missing the
`ClaimTask` and `UpdateTaskStatus` methods that were added to `app.go` in earlier plans.

## Files to Edit

- `frontend/src/types/domain.ts` — add `retryCount?: number` to `Task` interface
- `frontend/src/types/wails.ts` — add `ClaimTask` and `UpdateTaskStatus` to Window.go
  declaration block; add `claimTask` and `updateTaskStatus` exported wrapper functions

## Implementation Steps

### 1. `frontend/src/types/domain.ts`

Add `retryCount?: number` to the `Task` interface. Place it after `updatedAt`:

```ts
export interface Task {
  // ...existing fields...
  updatedAt: string;
  retryCount?: number; // ← add this
  logs: string;
  // ...rest...
}
```

### 2. `frontend/src/types/wails.ts`

In the `Window.go.main.App` declaration block, add the two missing methods:

```ts
ClaimTask(taskID: string, sessionID: string): Promise<Task>
UpdateTaskStatus(taskID: string, sessionID: string, status: TaskStatus, logs: string): Promise<Task>
```

Then add the exported wrapper functions at the bottom of the file (same pattern
as existing wrappers, with HTTP fallback):

```ts
export async function claimTask(taskID: string, sessionID: string): Promise<Task> {
  if (isWails()) return window.go!.main!.App!.ClaimTask(taskID, sessionID);
  const r = await fetch(`/api/ai-sessions/${sessionID}/tasks`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ taskId: taskID }),
  });
  if (!r.ok) throw new Error(`HTTP ${r.status}`);
  return r.json() as Promise<Task>;
}

export async function updateTaskStatus(
  taskID: string,
  sessionID: string,
  status: TaskStatus,
  logs: string,
): Promise<Task> {
  if (isWails()) return window.go!.main!.App!.UpdateTaskStatus(taskID, sessionID, status, logs);
  const r = await fetch(`/api/tasks/${taskID}/status`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ sessionId: sessionID, status, logs }),
  });
  if (!r.ok) throw new Error(`HTTP ${r.status}`);
  return r.json() as Promise<Task>;
}
```

## Acceptance Criteria

- [ ] `frontend/src/types/domain.ts` `Task` interface has `retryCount?: number`
- [ ] `frontend/src/types/wails.ts` Window declaration has `ClaimTask` and `UpdateTaskStatus`
- [ ] `frontend/src/types/wails.ts` exports `claimTask()` and `updateTaskStatus()` wrapper functions
- [ ] `cd frontend && ./node_modules/.bin/vue-tsc --noEmit` exits 0
- [ ] `cd frontend && npm run build` exits 0

## Anti-patterns to Avoid

- NEVER import from `wailsjs/` auto-generated files in hand-maintained code
- NEVER make a field required if the backend may omit it from older responses
