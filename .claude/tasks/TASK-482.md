---
id: TASK-482
plan: PLAN-062
status: done
wave: 3
priority: 3
---

# TASK-482: Replace all console.warn/console.error with structured error state

**Problem:** 16+ `console.warn` and `console.error` calls are scattered across composables and components. None of these surface errors to the user — they are invisible in production builds (DevTools closed). The affected files are: `useGlobalSSE`, `useLogs`, `useActivities`, `useAISessions`, `useDiscovery`, `useProviders`, `useTasks`, `ProviderStatus.vue`, `BacklogList.vue`, `wails.ts`, `MissionControlView.vue`, `App.vue`.

**Fix:**

1. Audit all `console.warn`/`console.error` calls across `frontend/src`: `grep -rn 'console\.warn\|console\.error' frontend/src --include='*.ts' --include='*.vue'` — catalogue into a table of file, line, error message, severity
2. For each call, classify as: (a) **transient/expected** (e.g., SSE reconnect notice) — replace with structured `error` ref or swallow silently; (b) **user-actionable error** (e.g., failed API call, auth failure) — route to a toast or `error` ref
3. Create or reuse a shared notification composable `useToast.ts` (if not already in `frontend/src/composables/`) that exposes `showError(msg)`, `showSuccess(msg)`, `showWarning(msg)` backed by a short-lived notification stack
4. For **user-actionable errors** in composables: set an `error: Ref<string | null>` on the composable and emit the message to `useToast` via `showError()`
5. For **user-actionable errors** in components: call `useToast().showError(err.message)` in catch blocks
6. For SSE parse/reconnect notices: convert to a single `console.debug` (acceptable in development, stripped by Vite in production `mode: production` build)
7. Ensure `MissionControlView.vue` promote/cancel error paths show toasts (these were noted in PLAN-059 Wave 4 but may not be fully complete)
8. After changes: `grep -rn 'console\.warn\|console\.error' frontend/src --include='*.ts' --include='*.vue'` should return zero results outside test files and `*.spec.ts`

**Files:**

- `frontend/src/composables/useGlobalSSE.ts`
- `frontend/src/composables/useLogs.ts`
- `frontend/src/composables/useActivities.ts`
- `frontend/src/composables/useAISessions.ts`
- `frontend/src/composables/useDiscovery.ts`
- `frontend/src/composables/useProviders.ts`
- `frontend/src/composables/useTasks.ts`
- `frontend/src/composables/useToast.ts` (new or extend existing)
- `frontend/src/components/ProviderStatus.vue`
- `frontend/src/components/BacklogList.vue`
- `frontend/src/wailsjs/wails.ts`
- `frontend/src/views/MissionControlView.vue`
- `frontend/src/App.vue`

**Acceptance criteria:**

- `grep -rn 'console\.warn\|console\.error' frontend/src --include='*.ts' --include='*.vue'` returns zero results outside spec/test files
- API failures in composables surface as visible toasts or inline error states
- `vue-tsc --noEmit` zero errors
- Vitest unit tests pass (update any tests that assert on `console.warn` calls)
