---
id: TASK-429
plan: PLAN-057
status: todo
role: testing
wave: 7
---

# TASK-429: Vitest — MissionControlView renders all 5 panels

Write `frontend/src/test/MissionControlView.spec.ts`:

- Status bar renders agent count, task count, provider count
- Unified task list shows DRAFT+QUEUED+PROCESSING tasks with instruction text
- Agents panel shows active sessions
- Providers panel shows provider health with context limit
- Recent completions panel shows terminal tasks
- Submit form toggles collapse

Mock `useTasks`, `useAISessions`, `useProviders`, `useActivities`.

**Files:** `frontend/src/test/MissionControlView.spec.ts` (NEW)
**Depends on:** TASK-410..414
