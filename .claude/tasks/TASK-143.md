---
id: TASK-143
title: Fix ProviderInfo TypeScript type and ProviderStatus template
role: frontend
planId: PLAN-020
status: todo
dependencies: []
createdAt: 2026-03-11T19:00:00.000Z
---

## Context

The Go backend serializes `ProviderInfo` as `{"name":…,"active":…,"activeModel":…,"models":…}`
(explicit `json:"…"` tags). The TypeScript interface in `domain.ts` declares the same
fields as `Name`, `Active`, `ActiveModel`, `Models` (PascalCase). This means every
`v-for` item renders with `p.Name === undefined`, `p.Active === undefined`, etc. —
provider cards appear completely invisible on the dark background.

## Files to Read

- `frontend/src/types/domain.ts`
- `frontend/src/components/ProviderStatus.vue`
- `frontend/src/composables/useProviders.ts`

## Implementation Steps

### 1. Fix `frontend/src/types/domain.ts`

Change `ProviderInfo` field names from PascalCase to camelCase:

```typescript
// BEFORE
export interface ProviderInfo {
  Name: string
  Active: boolean
  ActiveModel: string
  Models: string[]
  baseURL?: string
  error?: string
}

// AFTER
export interface ProviderInfo {
  name: string
  active: boolean
  activeModel: string
  models: string[]
  baseURL?: string
  error?: string
}
```

### 2. Fix `frontend/src/components/ProviderStatus.vue`

Update every template reference to use the corrected camelCase names.
There are exactly 6 occurrences:

| Old reference           | New reference          |
|-------------------------|------------------------|
| `:key="p.Name"`         | `:key="p.name"`        |
| `p.Active` (class)      | `p.active`             |
| `p.Active` (span class) | `p.active`             |
| `{{ p.Name }}`          | `{{ p.name }}`         |
| `p.Active && p.ActiveModel` | `p.active && p.activeModel` |
| `{{ p.ActiveModel }}`   | `{{ p.activeModel }}`  |
| `!p.Active && p.error`  | `!p.active && p.error` |

### 3. Verify `useProviders.ts`

Read `frontend/src/composables/useProviders.ts` — it only stores the providers array and
calls `getProviders()`. No field-name references there, so no changes needed unless
you find any.

## Acceptance Criteria

- [ ] `domain.ts` `ProviderInfo` has lowercase `name`, `active`, `activeModel`, `models`
- [ ] `ProviderStatus.vue` uses lowercase field names throughout the template
- [ ] TypeScript compiler reports no errors: run `cd frontend && npm run build 2>&1`

## Anti-patterns to Avoid

- Do not change `Task` or `TaskInput` types — their PascalCase field names match Go's
  default JSON output (no struct tags on `Task`)
- Do not change `ProviderConfig` type — its fields are already camelCase and correct
