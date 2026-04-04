---
id: TASK-481
plan: PLAN-062
status: done
wave: 3
priority: 3
---

# TASK-481: Fix TaskDetailDrawer — no cancel for PROCESSING tasks

**Problem:** `frontend/src/components/TaskDetailDrawer.vue` renders a cancel button only when `task.status === 'QUEUED'`. A task that has been claimed and moved to `PROCESSING` cannot be cancelled from the UI, even if it is clearly stuck (e.g., LLM provider offline, task running for hours). Users must use the CLI or HTTP API directly.

**Fix:**

1. Add a separate "Interrupt" button that renders when `task.status === 'PROCESSING'`
2. The Interrupt button calls the same `cancelTask(task.id)` API as the QUEUED cancel button (the backend already handles `DELETE /api/tasks/{id}` for PROCESSING tasks)
3. Style the Interrupt button with an amber/orange color to distinguish it from the QUEUED cancel (red) and signal a more forceful action
4. Add a brief label difference: QUEUED → "Cancel" (gray/red), PROCESSING → "Interrupt" (amber) with tooltip "Force cancel a running task"
5. On successful interrupt, close the drawer and show a toast: "Task interrupted"
6. On failure (e.g., task already completed by the time the request arrives), show an inline error in the drawer rather than crashing

**Files:**

- `frontend/src/components/TaskDetailDrawer.vue`

**Acceptance criteria:**

- QUEUED tasks show "Cancel" button (existing behavior preserved)
- PROCESSING tasks show "Interrupt" button styled in amber
- Clicking Interrupt calls `cancelTask(id)` and closes the drawer on success
- COMPLETED/FAILED tasks show neither button
- `vue-tsc --noEmit` zero errors
