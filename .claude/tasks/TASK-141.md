---
id: TASK-141
title: Add VS Code Extension feature to HomeView.vue
role: frontend
planId: PLAN-019
status: todo
dependencies: []
createdAt: 2026-03-11T18:00:00.000Z
---

## Context

`docs/src/views/HomeView.vue` has a `features` array with 8 items. The VS Code extension
(built in PLAN-018) is a first-class interface for the product but is not currently
listed. Adding it ensures visitors see the full picture on the home page.

## Files to Read

- `docs/src/views/HomeView.vue` (full file — check current `features` array and `stats`)

## Implementation Steps

1. Open `docs/src/views/HomeView.vue` and find the `features` array in `<script setup>`.

2. **Append a ninth feature** to the `features` array after the existing `Desktop GUI` entry:
   ```ts
   { icon: '🔌', title: 'VS Code Extension', desc: 'Submit tasks, monitor the queue, and switch providers without leaving your editor. Available as a .vsix for VS Code 1.85+.' },
   ```

3. **Update the stats bar** — change the `Interfaces` stat from `3` to `4`:
   ```ts
   { value: '4', label: 'Interfaces (HTTP/MCP/GUI/VSCode)' },
   ```

4. No other changes needed.

## Acceptance Criteria

- [ ] `features` array has 9 entries, last one is the VS Code Extension
- [ ] Stats bar shows `4` for interfaces, label updated to include VSCode
- [ ] `npm run build` in `docs/` succeeds with no TypeScript errors

## Anti-patterns to Avoid

- Do not change existing feature entries
- Do not add a new section or reorder the features grid
