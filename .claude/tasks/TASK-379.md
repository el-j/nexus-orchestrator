# TASK-379: Fix HistoryView + LiveActivityView — pass projectPath to API

**Plan:** PLAN-055 | **Wave:** 3 | **Status:** done | **Role:** frontend

## Problem

`HistoryView.vue` fetches ALL tasks from `/api/tasks/all` and then filters to `currentProject` in computed JS. This is wasteful (loads hundreds of tasks), wrong semantics, and doesn't match the server-side filter pattern used by `useDiscoveredPlans.ts`.

`LiveActivityView.vue` uses `selectedProject` for client-side filtering but the TASK-378 fix wires it to the composable. This task focuses on `HistoryView.vue`.

## Files to Edit

- `frontend/src/views/HistoryView.vue` — change API fetch to include `?projectPath=`
- Also check `frontend/src/composables/useTasks.ts` if HistoryView uses a composable

## Implementation

### Read HistoryView.vue to find the fetch call:

Likely pattern:

```typescript
// BEFORE — fetches all tasks, filters in JS:
const res = await fetch(`${baseUrl}/api/tasks/all`);
tasks.value = (await res.json()) as Task[];

// AFTER — passes projectPath as query param:
const params = new URLSearchParams();
if (currentProject.value) {
  params.set('projectPath', currentProject.value);
}
const res = await fetch(`${baseUrl}/api/tasks/all?${params.toString()}`);
tasks.value = (await res.json()) as Task[];
```

Also add a `watch(currentProject, fetchTasks)` so that changing the project selector triggers a new server-side fetch.

### HTTP API check:

Verify `/api/tasks/all` handler in `internal/adapters/inbound/httpapi/handlers_tasks.go` supports `?projectPath=` query param. If not:

```go
// handlers_tasks.go — handleGetAllTasks
func handleGetAllTasks(orch ports.Orchestrator) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        projectPath := r.URL.Query().Get("projectPath")
        tasks, err := orch.GetAllTasks()
        if err != nil { ... }
        if projectPath != "" {
            filtered := tasks[:0]
            for _, t := range tasks {
                if t.ProjectPath == projectPath {
                    filtered = append(filtered, t)
                }
            }
            tasks = filtered
        }
        writeJSON(w, http.StatusOK, tasks)
    }
}
```

### Keep minimal client-side filter:

After the server returns project-scoped tasks, keep the `selectedFilter` (status badge) filter as pure client-side since it's already on the small result set.

## Verification

- `npx vue-tsc --noEmit` passes
- `npx vitest run` passes (new TASK-383 tests cover HistoryView)
- Selecting a project triggers a new API call with `?projectPath=...` in the URL (check Network tab)
- Switching project immediately replaces task list (not stale view)

## Status

done
