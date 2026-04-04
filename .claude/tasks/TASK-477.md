---
id: TASK-477
plan: PLAN-062
status: done
wave: 3
priority: 3
---

# TASK-477: Remove/wire AgentDetailDrawer dead code

**Problem:** `frontend/src/components/AgentDetailDrawer.vue` is never imported by any parent component. `AgentsView.vue` does not render it, so it is completely unreachable. Inside the drawer, a "View in timeline →" button emits a `navigate` event to nobody — no parent listens. The component is either incomplete (should be wired) or abandoned (should be deleted).

**Fix:**

1. Determine intent: check git history (`git log --follow -p frontend/src/components/AgentDetailDrawer.vue`) to see if it was recently in use or always orphaned
2. **If wire path chosen:** Import `AgentDetailDrawer` in `AgentsView.vue`; add a `selectedAgent` ref; bind `:agent="selectedAgent"` and `@close="selectedAgent = null"`; trigger it from the agent row click handler; replace the `navigate` emit handler with `router.push({ path: '/projects/live-activity', query: { agent: agent.id } })`
3. **If delete path chosen:** Remove `frontend/src/components/AgentDetailDrawer.vue`; verify no other import references remain (`grep -r 'AgentDetailDrawer' frontend/src` returns empty)
4. Either way, confirm `AgentsView.vue` has a functional detail/action path for each agent row (click → drawer OR click → route)

**Files:**

- `frontend/src/components/AgentDetailDrawer.vue` (remove or wire)
- `frontend/src/views/AgentsView.vue`

**Acceptance criteria:**

- `AgentDetailDrawer.vue` is either properly imported and functional, or deleted
- `grep -r 'AgentDetailDrawer' frontend/src` returns zero results if deleted
- No orphaned `emit('navigate', ...)` calls remain without a listener
- `vue-tsc --noEmit` zero errors
