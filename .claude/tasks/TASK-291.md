---
id: TASK-291
title: Extract shared time formatting to frontend/src/utils/time.ts
role: frontend
planId: PLAN-045
status: todo
dependencies: []
createdAt: 2026-03-14T18:00:00.000Z
---

## Context

`relativeTime`, `formatDate`, and `timeAgo` are copy-pasted into **6 different Vue files**:

| File                                                       | Function         |
| ---------------------------------------------------------- | ---------------- |
| `frontend/src/views/AIAgentsView.vue:117`                  | `relativeTime()` |
| `frontend/src/views/HistoryView.vue:227`                   | `formatDate()`   |
| `frontend/src/views/LiveActivityView.vue:167`              | `timeAgo()`      |
| `frontend/src/components/AISessionCard.vue:121`            | `relativeTime()` |
| `frontend/src/components/DiscoveredProvidersPanel.vue:171` | `timeAgo()`      |
| `frontend/src/components/TaskDetailDrawer.vue:127`         | `formatDate()`   |

All three functions are essentially the same logic with minor naming differences.

## Implementation Steps

1. **Create `frontend/src/utils/time.ts`** with a single canonical implementation:

   ```ts
   /**
    * Returns a human-readable relative-time string (e.g. "3 min ago", "just now").
    * All three previous variants (relativeTime, formatDate, timeAgo) are unified here.
    */
   export function relativeTime(iso: string | undefined): string {
     if (!iso) return '—';
     const diff = Date.now() - new Date(iso).getTime();
     if (diff < 60_000) return 'just now';
     if (diff < 3_600_000) return `${Math.floor(diff / 60_000)} min ago`;
     if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)} hr ago`;
     return new Date(iso).toLocaleDateString();
   }

   /** Alias kept for call-sites that used the name `timeAgo`. */
   export const timeAgo = relativeTime;

   /** Alias kept for call-sites that used the name `formatDate`. */
   export const formatDate = relativeTime;
   ```

2. **Update all 6 files** to remove the local function and import from utils:

   ```ts
   import { relativeTime } from '../utils/time'; // adjust path depth per file
   ```

   Remove the old local function definition after adding the import.

3. **Depth rules for relative imports:**
   - Files in `src/views/` → `'../utils/time'`
   - Files in `src/components/` → `'../utils/time'`

## Acceptance Criteria

- [ ] `frontend/src/utils/time.ts` exists and exports `relativeTime`, `timeAgo`, `formatDate`
- [ ] No local `relativeTime`, `timeAgo`, or `formatDate` function definitions remain in any `.vue` file
- [ ] `cd frontend && ./node_modules/.bin/vue-tsc --noEmit` exits 0
- [ ] `cd frontend && npm run build` exits 0

## Anti-patterns to Avoid

- NEVER change the visual output of the formatted time — this is a pure refactor
- NEVER add a new testing framework; validate with the existing `vue-tsc` + build
