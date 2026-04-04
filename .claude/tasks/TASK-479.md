---
id: TASK-479
plan: PLAN-062
status: done
wave: 3
priority: 3
---

# TASK-479: Fix useProjectFilter — misses completed projects

**Problem:** `frontend/src/composables/useProjectFilter.ts` builds its `projectList` exclusively from `useTasks()` which only exposes tasks in non-terminal statuses (QUEUED, PROCESSING, DRAFT). Projects where all tasks are COMPLETED, FAILED, or exist only as DRAFT are therefore absent from the sidebar project selector. Users working with history or reviewing completed work cannot filter to those projects.

**Fix:**

1. Identify the current `projectList` derivation in `useProjectFilter.ts` — confirm it sources only from `useTasks().tasks`
2. Add a separate history fetch: call `GET /api/tasks/all` (or the appropriate all-tasks endpoint) once on mount and store results in an `allTasks` ref
3. Derive `projectList` as the union of `useTasks().tasks` and `allTasks`, deduplicating by `projectPath` using a `Set` or `Map`
4. If `GET /api/tasks/all` does not exist, check if `GET /api/tasks?status=COMPLETED` + `GET /api/tasks?status=FAILED` endpoints exist and union those; alternatively use `GET /api/tasks` if it supports a `?all=true` query param — pick the approach that matches the existing HTTP API
5. Refresh `allTasks` on a 60 s interval (history does not change frequently; no need for SSE)
6. Ensure `projectList` is sorted alphabetically or by most recent task activity
7. Clear the interval in `onUnmounted`

**Files:**

- `frontend/src/composables/useProjectFilter.ts`

**Acceptance criteria:**

- Projects with only COMPLETED or FAILED tasks appear in the sidebar project selector
- Projects with only DRAFT tasks appear in the selector
- No duplicate project entries
- `vue-tsc --noEmit` zero errors
