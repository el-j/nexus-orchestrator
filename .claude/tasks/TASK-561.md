---
id: TASK-561
title: Wire orphan views into router and fix all promote handlers
role: frontend
planId: PLAN-071
status: todo
dependencies: [TASK-556]
createdAt: 2026-04-15T00:00:00Z
---

## Context

`DiscoveryView.vue` and `AISessionsView.vue` are fully implemented but not registered in the Vue Router, making them completely unreachable by users. `DiscoveryView.handlePromote` is a stub that only logs to console with a TODO comment. `DiscoveredPlansView.handlePromote` creates a draft but never refreshes the plan list, leaving the UI stale with an incorrect "todo" badge.

## Files to Read

- `frontend/src/router/index.ts`
- `frontend/src/views/DiscoveryView.vue`
- `frontend/src/views/AISessionsView.vue`
- `frontend/src/views/DiscoveredPlansView.vue`
- `frontend/src/components/AppSidebar.vue`

## Implementation Steps

1. In `router/index.ts`: add a route for `DiscoveryView` at path `/discovery` (lazy-loaded, same pattern as other routes)
2. Evaluate `AISessionsView.vue` — if it duplicates the sessions tab in `AgentsView`, **delete it** and skip its route. If it offers distinct value, add a route at `/ai-sessions`.
3. In `AppSidebar.vue`: add a nav item for Discovery (and AI Sessions if kept) matching the sidebar pattern used by existing routes
4. In `DiscoveryView.vue:handlePromote`: implement the navigate-and-prefill behaviour — call `router.push('/providers')` and pass the discovered provider as a query param or via a shared reactive state so the ProvidersView pre-fills the add-provider form
5. In `DiscoveredPlansView.vue:handlePromote`: after the `createDraft` call succeeds, call the existing `refresh()` or `fetchPlans()` function to reload the plans list

## Acceptance Criteria

- [ ] `vue-tsc --noEmit` exits 0
- [ ] `npm test` passes in `frontend/`
- [ ] `DiscoveryView` is reachable via `/discovery` route
- [ ] `AISessionsView` is either routed at `/ai-sessions` OR deleted (no orphan)
- [ ] `DiscoveryView.handlePromote` navigates to providers view (no longer a console.log stub)
- [ ] `DiscoveredPlansView.handlePromote` triggers a list refresh after draft creation
- [ ] AppSidebar has a nav entry for Discovery

## Anti-patterns to Avoid

- NEVER use `as any` to bypass Vue Router type checks — use typed `RouteLocationRaw`
- NEVER import views directly in AppSidebar — use router.push() for navigation
