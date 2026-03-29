---
id: TASK-436
plan: PLAN-058
status: done
---

# TASK-436: AgentsView filtering (search + status)

**Plan:** PLAN-058 (Wave 3)

## Problem

- No way to filter agents by status or search by name
- With 15+ agents, finding specific ones is difficult

## Solution

1. Add a search input + status filter chips bar below the tab bar
2. Search filters sessions by agentName, externalId, projectPath
3. Search filters discovered agents by name, kind, workingDir
4. Status chips: All, Active, Idle, Disconnected (for sessions); All, Running, Stopped (for agents)
5. Apply filters to both sortedSessions and sortedAgentRoots

## Files

- `frontend/src/views/AgentsView.vue`

## Acceptance

- Typing in search box filters visible cards in real-time
- Status filter chips highlight and filter
- Filters work together (search + status)
