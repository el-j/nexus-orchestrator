---
id: TASK-434
plan: PLAN-058
status: done
---

# TASK-434: AgentsView sorting (active first + toggle)

**Plan:** PLAN-058 (Wave 2)

## Problem

- Sessions and discovered agents render in API response order
- Active/running items are mixed randomly with inactive/disconnected ones

## Solution

1. Add `sortedSessions` computed that sorts: active > idle > disconnected, then by lastActivity desc
2. Add `sortedAgentRoots` computed that sorts: isRunning first, then by lastSeen desc
3. Add sort dropdown with options: "Active first" (default), "Recent first", "Name A-Z"
4. Use sorted computeds in template instead of raw `sessions` / `agentTree.roots`

## Files

- `frontend/src/views/AgentsView.vue`

## Acceptance

- Active sessions appear before idle, idle before disconnected
- Running agents appear before stopped ones
- Sort toggle changes ordering
