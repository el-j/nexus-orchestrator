---
id: TASK-415
plan: PLAN-057
status: todo
role: frontend
wave: 3
---

# TASK-415: Delete DiscoveryView, merge into ProvidersView

`DiscoveryView.vue` is 100% redundant with `ProvidersView.vue` — both render `DiscoveredProvidersPanel`. Delete `DiscoveryView.vue`, remove its sidebar entry and route. The "Discovered on System" section already exists in ProvidersView.

**Files:** DELETE `frontend/src/views/DiscoveryView.vue`, update `AppSidebar.vue`, `App.vue`
**Depends on:** none
