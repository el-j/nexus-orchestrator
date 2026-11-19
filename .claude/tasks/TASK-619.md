---
id: TASK-619
planId: PLAN-075
title: 'Fix browser UI API token lockout in wails.ts and make queueCap editable in SettingsView.vue'
role: coder
wave: 4
status: done
createdAt: 2026-10-19
dependsOn: [TASK-611]
---

# TASK-619 — Fix browser UI API token lockout in wails.ts and make queueCap editable in SettingsView.vue

1. **`frontend/src/types/wails.ts` & `frontend/src/views/SettingsView.vue`**:
   - Persist the active HTTP API token in `localStorage` (`nexus:apiToken`) when generated/rotated or entered in `SettingsView.vue`, and attach `Authorization: Bearer <token>` in `request()` and direct `fetch()` calls when present so generating an API token in browser mode does not lock out the web UI with `401 Unauthorized`.
   - Allow editing and saving `queueCap` in `SettingsView.vue` via `updateRuntimeConfig({ queueCap })`.

Done when: the failing test came first, 100% frontend coverage and `make nice` pass, docs and changelog updated.
