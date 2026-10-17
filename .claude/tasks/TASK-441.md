---
id: TASK-441
plan: PLAN-059
status: closed
closedReason: 'stale: planned before v0.10.0 and never started; the work is tracked in PLAN-073 and PLAN-074 or was superseded'
wave: 1
priority: 1
---

# TASK-441: Register LiveActivityView + HistoryView + BacklogView in App.vue

**Problem:** ProjectActivityView emits `@navigate='live-activity'` → App.vue sets `currentView = 'live-activity'` → but no `v-else-if` exists for it → black screen. HistoryView and BacklogView similarly exist but are not wired.

**Fix:**

1. Import `LiveActivityView`, `HistoryView`, `BacklogView` in App.vue
2. Add `v-else-if="currentView === 'live-activity'"` with `<LiveActivityView />`
3. Add `v-else-if="currentView === 'history'"` with `<HistoryView />`
4. Add `v-else-if="currentView === 'backlog'"` with `<BacklogView />`
5. Pass appropriate event handlers if needed

**Files:** `frontend/src/App.vue`
