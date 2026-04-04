---
id: TASK-435
plan: PLAN-058
status: done
---

# TASK-435: ProvidersView sorting (active first + toggle)

**Plan:** PLAN-058 (Wave 2)

## Problem

- Active providers grid renders in API order — unreachable providers mixed with active ones
- AI Coding Tools section also renders unsorted

## Solution

1. Add `sortedProviders` computed: active first, then alphabetical
2. Add `sortedAgents` computed for AI tools: isRunning first, then alphabetical
3. Add sort toggle matching AgentsView pattern

## Files

- `frontend/src/views/ProvidersView.vue`

## Acceptance

- Active providers always appear before unreachable ones
- Running AI tools appear before inactive ones
