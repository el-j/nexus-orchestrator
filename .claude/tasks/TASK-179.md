---
id: TASK-179
title: Frontend — TypeScript types, useProjectFilter, enhanced useTasks
role: frontend
planId: PLAN-024
status: closed
closedReason: 'stale: planned before v0.10.0 and never started; the work is tracked in PLAN-073 and PLAN-074 or was superseded'
dependencies: [TASK-176]
createdAt: 2026-03-11T22:00:00.000Z
---

## Context

The frontend TypeScript types must reflect the new domain fields, and two composables are needed: `useProjectFilter` for project-scoped views and `useTasks` enhancements for filtering by project and status.

## Files to Read

- `frontend/src/types/domain.ts` — existing Task, TaskStatus types
- `frontend/src/types/wails.ts` — Wails binding wrappers
- `frontend/src/composables/useTasks.ts` — existing composable
- `frontend/src/composables/useProviders.ts` — pattern reference

## Implementation Steps

1. Update `frontend/src/types/domain.ts`:
   - Add `'DRAFT' | 'BACKLOG'` to `TaskStatus` union
   - Add fields to `Task`: `providerName?: string`, `priority?: number`, `tags?: string[]`
   - Add `TaskInput` type with optional fields for form submissions
   - Add `TaskUpdate` type for partial updates

2. Update `frontend/src/types/wails.ts`:
   - Add wrapper functions: `createDraft(task)`, `getBacklog(projectPath)`, `promoteTask(id)`, `updateTask(id, updates)`

3. Create `frontend/src/composables/useProjectFilter.ts`:
   - `currentProject: Ref<string | null>` — null = all projects
   - `projectList: ComputedRef<string[]>` — derived from unique `Task.projectPath` values
   - `setProject(path: string | null): void`
   - Persist to `localStorage('nexus-project-filter')`
   - Provide via Vue `provide/inject` so all views share one instance

4. Enhance `frontend/src/composables/useTasks.ts`:
   - Add `backlogTasks: ComputedRef<Task[]>` — filter by DRAFT+BACKLOG + current project
   - Add `queuedTasks: ComputedRef<Task[]>` — filter by QUEUED+PROCESSING + current project
   - Add `filteredTasks: ComputedRef<Task[]>` — filter by current project (all statuses)
   - Consume `useProjectFilter()` for filtering
   - Add methods: `createDraft(task)`, `promoteTask(id)`, `updateTask(id, updates)`
   - SSE: handle new event types `task_promoted`, `task_updated`

## Acceptance Criteria

- [ ] TypeScript types match Go domain struct (all new fields present)
- [ ] `useProjectFilter` persists selection to localStorage
- [ ] `backlogTasks` computed ref filters correctly by project + status
- [ ] `createDraft`, `promoteTask`, `updateTask` call correct HTTP/Wails endpoints
- [ ] Project filter is shared across all views via provide/inject

## Anti-patterns to Avoid

- NEVER hardcode API base URL — use existing pattern
- NEVER poll for backlog separately — reuse existing SSE/polling with status filter
- NEVER duplicate project list logic — derive from tasks, don't maintain separately
