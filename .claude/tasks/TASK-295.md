---
id: TASK-295
title: 'Mark TASK-014/021/022 done in orchestrator.json'
role: orchestration
planId: PLAN-045
status: todo
dependencies: []
createdAt: 2026-03-14T18:00:00.000Z
---

## Context

Three PLAN-002 tasks have been showing `status: "todo"` since 2026-03-09 despite
being fully implemented (confirmed by deep audit on 2026-03-14):

- **TASK-014**: Retry limits + exponential backoff + queue cap + ProjectPath normalization
  - `domain.Task.RetryCount` exists; `requeueForRetry()` exists; `WithQueueCap()` exists;
    `filepath.Clean` in SubmitTask; `TestRetryLimit`, `TestQueueCap`, `TestPathNormalization`
    all pass in `orchestrator_hardening_test.go`
- **TASK-021**: GUI Task History page + TaskDetail Drawer
  - `HistoryView.vue` (239 lines) exists; `TaskDetailDrawer.vue` (134 lines) exists;
    frontend builds and type-checks clean
- **TASK-022**: GUI Settings page
  - `SettingsView.vue` (180 lines) exists with provider connections, queue limits, env var
    display, and about section; frontend builds clean

## Steps

Run Python script to update all three to `status: "done"` with `completedAt` timestamp.

## Acceptance Criteria

- [ ] `TASK-014.status` is `"done"` in orchestrator.json
- [ ] `TASK-021.status` is `"done"` in orchestrator.json
- [ ] `TASK-022.status` is `"done"` in orchestrator.json
- [ ] All 3 task `.md` files updated to `status: done`
