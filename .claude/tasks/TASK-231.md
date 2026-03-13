---
id: TASK-231
title: Add null-coalescing guards to all frontend composable Wails calls
role: backend
planId: PLAN-032
status: todo
dependencies: []
createdAt: 2026-03-13T00:30:00.000Z
---

## Context
Even after the Go repo methods are fixed to return non-nil slices, the frontend should defensively guard all array assignments from Wails calls with `?? []` to prevent `null is not an object` crashes. This is a defense-in-depth pattern — the Go side might still send `null` from other code paths (e.g. HTTP API fallback, error scenarios).

## Files to Read
- `frontend/src/composables/useTasks.ts` — line 31: `tasks.value = await getQueue()`
- `frontend/src/composables/useAISessions.ts` — line 19: `sessions.value = await listAISessions()`
- `frontend/src/composables/useProviders.ts` — line 12: `providers.value = await getProviders()`
- `frontend/src/views/SettingsView.vue` — line ~50: `configs.value = await listProviderConfigs()`
- `frontend/src/composables/useDiscovery.ts` — line 19: `discovered.value = await getDiscoveredProviders()`

## Implementation Steps
1. In `useTasks.ts` `refresh()`: change `tasks.value = await getQueue()` to `tasks.value = (await getQueue()) ?? []`
2. In `useAISessions.ts` `refresh()`: change `sessions.value = await listAISessions()` to `sessions.value = (await listAISessions()) ?? []`
3. In `useProviders.ts` `refresh()`: change to `providers.value = (await getProviders()) ?? []`
4. In `SettingsView.vue` load function: change to `configs.value = (await listProviderConfigs()) ?? []`
5. In `useDiscovery.ts` `refresh()`: change to `discovered.value = (await getDiscoveredProviders()) ?? []`
6. Run `npx vue-tsc --noEmit` to verify no type errors

## Acceptance Criteria
- [ ] `npx vue-tsc --noEmit` exits 0
- [ ] Every composable that assigns a Wails array result to a `.value` ref uses `?? []`
- [ ] App no longer crashes with `null is not an object (evaluating 'e.value.map')` on empty database
- [ ] All `v-for` templates render empty state correctly when arrays are empty

## Anti-patterns to Avoid
- NEVER leave a Wails array call without `?? []` — Go nil slices marshal to `null`
- NEVER use `as` type assertions to silence the null — use runtime guards
