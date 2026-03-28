# TASK-378: Fix useActivities.ts — wire projectFilter option to API query param

**Plan:** PLAN-055 | **Wave:** 3 | **Status:** done | **Role:** frontend

## Problem

`useActivities()` in `frontend/src/composables/useActivities.ts` accepts a `projectFilter?: string` option but the `/api/activities/timeline` fetch call never includes it as a query parameter. The `LiveActivityView.vue` dropdown for project selection therefore does nothing.

## Files to Edit

- `frontend/src/composables/useActivities.ts` — add projectPath to URLSearchParams
- `frontend/src/views/LiveActivityView.vue` — pass `selectedProject` to `useActivities()`

## Implementation

### useActivities.ts fix:

Read the current file first, then find the fetch call and add projectPath:

```typescript
// In the fetch function inside useActivities:
const params = new URLSearchParams({
  limit: String(options?.limit ?? 100),
  since: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
});
if (options?.projectFilter) {
  params.set('projectPath', options.projectFilter);
}
const res = await fetch(`${base}/api/activities/timeline?${params.toString()}`);
```

If `projectFilter` is a `Ref<string>`, unref it:

```typescript
const pf = isRef(options?.projectFilter)
  ? (options.projectFilter as Ref<string>).value
  : options?.projectFilter;
if (pf) params.set('projectPath', pf);
```

Also: when `projectFilter` is a reactive ref, the composable should `watch` it and refetch:

```typescript
// If projectFilter is a ref, watch and refetch
if (isRef(options?.projectFilter)) {
  watch(options.projectFilter as Ref<string>, () => fetchActivities());
}
```

### LiveActivityView.vue fix:

The composable call currently ignores `selectedProject`:

```typescript
// BEFORE:
const { activities, filtered, loading, error, refresh } = useActivities();

// AFTER:
const { activities, filtered, loading, error, refresh } = useActivities({
  projectFilter: selectedProject, // reactive ref passed in
});
```

This way, when the dropdown changes `selectedProject`, `useActivities` watches it and refetches with `?projectPath=...`.

## Backend verification

Check that `/api/activities/timeline` actually accepts and uses `?projectPath=` — read `internal/adapters/inbound/httpapi/handlers_activity.go`. If the handler doesn't filter by projectPath, add that filter.

## Verification

- `npx vitest run` passes (new tests added in TASK-382 will cover this)
- Changing project dropdown in LiveActivityView triggers a new fetch with correct query param
- `vue-tsc --noEmit` passes

## Status

done
