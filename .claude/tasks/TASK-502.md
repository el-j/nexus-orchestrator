---
id: TASK-502
plan: PLAN-064
status: done
wave: 4
priority: 3
---

# TASK-502: Add MissionControlView interaction tests

## Description

`MissionControlView.spec.ts` currently contains only 2 tests verifying render and header text. The submit form, cancel action, status filter, SSE event integration, and promote action are completely untested. This view is the primary operational control surface of the application.

## Checklist

- [ ] Extend `frontend/src/views/__tests__/MissionControlView.spec.ts`
- [ ] Test submit form: fill in project path + instruction fields; click Submit; verify `submitTask()` (Wails call or composable method) invoked with correct arguments; new task appears in list
- [ ] Test cancel click: render view with one task; click Cancel button for that task; verify `cancelTask(taskId)` called; task removed from displayed list
- [ ] Test status filter: render with tasks in QUEUED, PROCESSING, COMPLETED states; select filter "PROCESSING"; verify only PROCESSING task visible
- [ ] Test SSE event updates task list: inject a `task-updated` event via the mocked global SSE bus; verify the task card reflects updated status without page reload
- [ ] Test promote action: render a DRAFT task; click Promote; verify `promoteTask(taskId)` called; task status updated in view

## Files

- `frontend/src/views/__tests__/MissionControlView.spec.ts` (extend)
- `frontend/src/views/MissionControlView.vue` (reference)

## Acceptance Criteria

- At least 5 new test cases added (submit, cancel, filter, SSE update, promote)
- Total test count in `MissionControlView.spec.ts` >= 7
- `pnpm vitest run` exits 0 with all tests green
