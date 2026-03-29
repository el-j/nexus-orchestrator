---
id: TASK-416
plan: PLAN-057
status: todo
role: frontend
wave: 3
---

# TASK-416: Merge AI Sessions + AI Agents into unified AgentsView

Create `frontend/src/views/AgentsView.vue` that combines:

1. **Registered Sessions** section (from AISessionsView) — session cards with disconnect/purge actions
2. **Discovered Agents** section (from AIAgentsView) — agent tree with parent/child nesting
3. **"Delegate All Active"** button (from AIAgentsView)

Single unified view. Use `useAISessions()` for sessions and direct discovery fetch for agents.

**Files:** `frontend/src/views/AgentsView.vue` (NEW)
**Depends on:** TASK-408 (dedup fix first)
