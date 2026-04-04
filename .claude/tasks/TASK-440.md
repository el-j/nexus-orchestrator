---
id: TASK-440
plan: PLAN-058
status: done
---

# TASK-440: Dense card layout across views

**Plan:** PLAN-058 (Wave 5)

## Problem

- Cards use `p-4` padding and `gap-4` spacing — too much whitespace for overview
- Need more information density across all views

## Solution

1. AgentsView session cards: reduce from `p-4` to `p-3`, `gap-4` to `gap-3`
2. AgentsView discovered agent cards: same density reduction
3. ProvidersView provider cards: `p-4` to `p-3`, `gap-4` to `gap-3`
4. ProvidersView AI tools cards: `p-4` to `p-3`, `gap-3` to `gap-2`
5. DiscoveredPlansView task cards: already compact, verify consistency
6. Reduce font sizes for secondary info (detection methods, timestamps) from 11px to 10px where appropriate

## Files

- `frontend/src/views/AgentsView.vue`
- `frontend/src/views/ProvidersView.vue`
- `frontend/src/views/DiscoveredPlansView.vue`

## Acceptance

- Cards are visually denser — more items visible per screen
- No cramped or overlapping text
- Consistent density across all three views
