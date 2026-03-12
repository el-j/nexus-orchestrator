---
id: TASK-221
title: Add Vue global error boundary + rejection handler
role: backend
planId: PLAN-030
status: todo
dependencies: [TASK-213]
createdAt: 2025-07-25T00:00:00.000Z
---

## Context
The Vue frontend has no global error handler. If any view component throws during setup (e.g., composable initialization), the entire UI crashes without recovery. Additionally, unhandled promise rejections (common with async composables) go completely silent. Both `app.config.errorHandler` and `window.addEventListener('unhandledrejection')` are missing.

## Files to Read
- `frontend/src/main.ts` — app initialization, plugin registration
- `frontend/src/App.vue` — root component
- `frontend/src/composables/useTasks.ts` — async operations that could reject
- `frontend/src/components/TaskDetailDrawer.vue` — `formatDate` silently returns empty on error

## Implementation Steps
1. In `frontend/src/main.ts`: add `app.config.errorHandler` that logs to console.error and optionally shows a toast notification via PrimeVue
2. Add `window.addEventListener('unhandledrejection', handler)` to catch unhandled promise rejections — log them and show user-friendly notification
3. In `App.vue`: wrap router-view with Vue `<Suspense>` + error boundary component, or add `onErrorCaptured` hook to display fallback UI instead of blank screen
4. In `TaskDetailDrawer.vue` `formatDate()`: return `'Invalid date'` instead of silent empty string on error, so users see something meaningful
5. Create a simple `ErrorFallback.vue` component that shows "Something went wrong" with a retry button

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `app.config.errorHandler` is set in `main.ts`
- [ ] Unhandled promise rejections are caught and logged
- [ ] Component errors show fallback UI instead of blank screen
- [ ] `cd frontend && npx vue-tsc --noEmit` exits 0

## Anti-patterns to Avoid
- NEVER let the entire app crash from a single component error
- NEVER swallow errors silently — at minimum log them
- NEVER show raw error objects to end users — show friendly messages
