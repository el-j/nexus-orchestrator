---
id: TASK-213
title: Fix frontend circular dep + incomplete components
role: backend
planId: PLAN-030
status: todo
dependencies: []
createdAt: 2025-07-25T00:00:00.000Z
---

## Context
Three critical frontend issues: (1) Circular dependency between `useProjectFilter.ts` and `useTasks.ts` — they import each other, causing runtime module initialization failures. (2) `TaskSubmitForm.vue` has an incomplete `handleSubmit()` function — the file ends mid-implementation. (3) `HistoryView.vue` has cut-off helper functions (`shortId`, `projectName`, `fileName`).

## Files to Read
- `frontend/src/composables/useProjectFilter.ts` — imports from useTasks
- `frontend/src/composables/useTasks.ts` — imports from useProjectFilter
- `frontend/src/components/TaskSubmitForm.vue` — incomplete handleSubmit at ~line 300
- `frontend/src/views/HistoryView.vue` — incomplete helper functions at end of file
- `frontend/src/types/domain.ts` — Task type definition

## Implementation Steps
1. Break the circular dependency between `useProjectFilter.ts` and `useTasks.ts`:
   - Option A: Extract shared state to a new `useTaskStore.ts` that both import
   - Option B: Move project filtering logic into `useTasks.ts` and remove `useProjectFilter.ts`
   - Choose the simpler option that preserves existing component API
2. Complete `TaskSubmitForm.vue` `handleSubmit()`: add the `submitTask()` call, error handling with toast notification, form reset on success, and loading state management
3. Complete `HistoryView.vue` helper functions: `shortId()` — return first 8 chars of task ID, `projectName()` — extract basename from project path, `fileName()` — extract filename from output path
4. Run TypeScript type check to verify no type errors introduced

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] No circular dependency warnings in frontend build
- [ ] `TaskSubmitForm.vue` successfully submits tasks end-to-end
- [ ] `HistoryView.vue` renders task history with proper helper values
- [ ] `cd frontend && npx vue-tsc --noEmit` exits 0

## Anti-patterns to Avoid
- NEVER use `any` type in TypeScript — always define proper types
- NEVER leave empty catch blocks — at minimum log the error
- NEVER mutate reactive state outside composable functions
