---
id: TASK-474
plan: PLAN-062
status: done
wave: 2
priority: 2
---

# TASK-474: Fix BacklogView error handling — no try/catch on refresh

**Problem:** `frontend/src/views/BacklogView.vue` calls `getBacklog()` inside a `refresh()` function with no try/catch. Any thrown error (network failure, non-2xx response, JSON parse error) causes the loading spinner to activate and never deactivate — the view is permanently frozen. There is also no error state rendered in the template.

**Fix:**

1. Wrap the `getBacklog()` call in `refresh()` with `try { ... } catch (err) { error.value = err instanceof Error ? err.message : 'Failed to load backlog' } finally { loading.value = false }`
2. Add `const error = ref<string | null>(null)` to the component state
3. Reset `error.value = null` at the start of each `refresh()` call before the try block
4. In the template, add an error state section: when `error` is non-null, display a red alert banner with the error message and a "Retry" button that calls `refresh()`
5. Ensure `loading.value` is set to `false` in the `finally` block (not only on success), so the spinner never gets stuck
6. Add the same try/finally guard to any other async calls in `BacklogView.vue` (e.g., promote, delete actions)

**Files:**

- `frontend/src/views/BacklogView.vue`

**Acceptance criteria:**

- With daemon offline: spinner disappears after the request times out/fails, error banner appears with retry button
- Clicking "Retry" clears the error and re-fetches
- `vue-tsc --noEmit` zero errors
- `loading` is always `false` after `refresh()` resolves or rejects
