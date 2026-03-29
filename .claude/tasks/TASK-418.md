---
id: TASK-418
plan: PLAN-057
status: done
role: frontend
wave: 3
---

# TASK-418: Merge Settings configuration into ProvidersView

Move the provider configuration CRUD section (Add/Edit/Remove provider configs) from `SettingsView.vue` into `ProvidersView.vue` as a collapsible "Configuration" panel. SettingsView should retain only: token management, queue cap, and runtime config.

This puts provider configuration next to provider status — the natural place.

**Files:** `frontend/src/views/ProvidersView.vue`, `frontend/src/views/SettingsView.vue`
**Depends on:** TASK-415
