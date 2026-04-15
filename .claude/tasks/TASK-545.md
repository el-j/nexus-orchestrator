---
id: TASK-545
planId: PLAN-070
title: 'Create useBrain composable + /brain route + wire dead context/search UI code'
role: frontend
status: todo
createdAt: 2026-04-14T02:00:00Z
---

# TASK-545 — Brain frontend: composable, view, wire dead code

## Context

Three brain-related gaps in the frontend:

1. **No `useBrain` composable**: Brain state is scattered inside `ProjectBrainCard.vue`. Every
   other feature domain has a composable (`useTasks`, `useAISessions`, etc.).

2. **Dead UI code**: `getProjectContext`, `getFocusedContext`, `searchKnowledge` are exported from
   `wails.ts` but called by ZERO Vue files or composables. The context and search half of the
   brain feature is entirely disconnected from the UI.

3. **No brain route**: `routes.ts` has no `/brain` entry. All brain UI is inside
   `DiscoveredPlansView` — a planning view — making brain features undiscoverable.

4. **`KnowledgeResult` type mismatch**: `domain.ts` declares `KnowledgeResult` with `relevance`
   etc. but the backend returns `ContextSection[]`. Types need reconciliation.

## Work Required

### `frontend/src/composables/useBrain.ts` (new file)

Create a `useBrain(projectPath: Ref<string>)` composable exposing:

- `status: Ref<BrainStatus | null>` — reactive, polled every 30s when projectPath set
- `loading: Ref<boolean>`
- `error: Ref<string | null>`
- `ingest(filePath: string): Promise<number>` — calls `ingestKnowledge`, updates status
- `search(query: string, limit?: number): Promise<ContextSection[]>` — calls `searchKnowledge`
- `getContext(maxTokens?: number): Promise<ContextResponse>` — calls `getProjectContext`
- `getFocusedContext(question: string, maxTokens?: number): Promise<ContextResponse>`
- `refresh(): void` — force-refetch status

### `frontend/src/views/BrainView.vue` (new file)

A dedicated brain view showing:

- Current `BrainStatus` (entry count, total tokens, last updated)
- Search input → FTS results list (topic, content preview, token count)
- Context query input → rendered `ContextResponse` sections
- Ingest button (trigger file-picker if Wails, show path input in web mode)

### `frontend/src/router/routes.ts`

Add route:

```ts
{
  icon: 'pi-brain',
  name: 'brain',
  path: '/brain',
  label: 'Brain',
  nav: true,
  component: () => import('../views/BrainView.vue'),
}
```

### Type reconciliation

In `domain.ts`: remove `KnowledgeResult` or align it with `ContextSection` fields.
Ensure `ContextSection` has all fields returned by the backend (`topic`, `content`, `tokenCount`,
`relevanceScore`, `kind`).

## File Targets

- `frontend/src/composables/useBrain.ts` (new)
- `frontend/src/views/BrainView.vue` (new)
- `frontend/src/router/routes.ts`
- `frontend/src/types/domain.ts`

## Acceptance Criteria

- `cd frontend && vue-tsc --noEmit` clean
- `/brain` route navigates to `BrainView.vue`
- `useBrain` composable exports all listed reactive refs and methods
- No dead exports in `wails.ts` for brain functions (all called by at least one consumer)
- `KnowledgeResult` type removed or reconciled with `ContextSection`
