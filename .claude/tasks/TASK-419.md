---
id: TASK-419
plan: PLAN-057
status: todo
role: frontend
wave: 3
---

# TASK-419: Restructure AppSidebar from 11 to 6 items

Update `AppSidebar.vue` to show 6 navigation items:

1. Mission Control (pi-home) — MissionControlView (default)
2. Agents (pi-users) — AgentsView
3. Providers (pi-server) — ProvidersView (now includes Discovery + Config)
4. Projects (pi-folder) — ProjectActivityView
5. Plans (pi-file) — DiscoveredPlansView
6. Settings (pi-cog) — SettingsView (tokens + runtime config only)

Remove entries for: Task Queue, Backlog, History, Live AI, Discovery, AI Sessions, AI Agents.

**Files:** `frontend/src/components/AppSidebar.vue`, `frontend/src/App.vue`
**Depends on:** TASK-415, TASK-417
