---
id: TASK-621
planId: PLAN-075
title: 'Add ProviderLimitBadges & routing preview to ProvidersView, wire Delegate/Terminate in AgentsView, fix LiveActivityView scan, and sync currentProject'
role: coder
wave: 4
status: done
createdAt: 2026-10-19
dependsOn: [TASK-611, TASK-620]
---

# TASK-621 — Add ProviderLimitBadges & routing preview to ProvidersView, wire Delegate/Terminate in AgentsView, fix LiveActivityView scan, and sync currentProject

1. **`frontend/src/views/ProvidersView.vue` & `frontend/src/components/ProviderStatus.vue`**:
   - Render `<ProviderLimitBadges :provider="p" />` on provider cards in `ProvidersView.vue` and `ProviderStatus.vue`.
   - Remove the duplicate `"Configured Providers"` CRUD table in `ProvidersView.vue`.
   - Add a **Routing Preview** section (`GET /api/routing`) in `ProvidersView.vue` showing the effective role ladder order, cooldowns, and quota utilization.
2. **`frontend/src/views/AgentsView.vue` & `LiveActivityView.vue` & `BrainView.vue`**:
   - Wire **Delegate to Nexus** (`POST /api/ai-sessions/{id}/delegate`) and **Terminate Session** (`POST /api/ai-sessions/{id}/terminate`) in `AgentsView.vue`.
   - Fix `"⟳ Scan Now"` in `LiveActivityView.vue` to also call `useActivities().refresh()`.
   - Watch `currentProject` from `useProjectState.ts` in `BrainView.vue`, `MissionControlView.vue`, `AgentsView.vue`, and `LiveActivityView.vue` so switching projects in the sidebar updates the active view.

Done when: the failing test came first, 100% frontend coverage and `make nice` pass, docs and changelog updated.
