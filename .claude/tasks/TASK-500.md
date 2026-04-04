---
id: TASK-500
plan: PLAN-064
status: done
wave: 3
priority: 1
---

# TASK-500: Add SettingsView and BacklogView component tests

## Description

`SettingsView.vue` controls token rotation, provider config, and queue caps — mutations with immediate live impact on the running daemon. `BacklogView.vue` is the primary backlog management surface. Both have zero tests. UI regressions in these views would be invisible until manual QA.

## Checklist

- [ ] Create `frontend/src/views/__tests__/SettingsView.spec.ts`
- [ ] Test settings form submit: fill fields, click Save; verify correct Wails/API call made with form values; success toast shown
- [ ] Test token rotate: click Rotate Token button; verify API call triggered; new token displayed in field (or copied to clipboard per implementation)
- [ ] Test validation: submit with empty required field; verify error message shown and API not called
- [ ] Create `frontend/src/views/__tests__/BacklogView.spec.ts`
- [ ] Test backlog load: on mount, `fetchBacklog()` called; backlog items rendered in list
- [ ] Test promote from backlog: click Promote button on a backlog item; verify `promoteTask()` called with correct ID; item moves out of backlog list (status changes)
- [ ] Test error state: API failure on fetch -> error banner shown
- [ ] Use `mount` (not `shallowMount`) with a router stub and a `createTestingPinia` store as needed
- [ ] All tests pass with `pnpm vitest run`

## Files

- `frontend/src/views/__tests__/SettingsView.spec.ts` (create)
- `frontend/src/views/__tests__/BacklogView.spec.ts` (create)
- `frontend/src/views/SettingsView.vue` (reference)
- `frontend/src/views/BacklogView.vue` (reference)

## Acceptance Criteria

- Minimum 3 test cases for SettingsView, 3 for BacklogView
- `mount` used (not shallow) so child component rendering is validated
- `pnpm vitest run` exits 0 with all new tests green
