---
id: TASK-191
title: "GUI: Settings view (provider URLs, queue cap, MCP addr)"
status: todo
priority: medium
role: frontend
dependencies: [TASK-189]
estimated_effort: M 1h
---

## Goal

Add a **Settings** page to the Web GUI so users can view and adjust runtime configuration: LM Studio / Ollama base URLs (via provider config forms), queue cap, and MCP listen address — without editing environment variables manually.

## Context

- No `SettingsView.vue` exists in `frontend/src/views/`
- Provider config management is already in `ProvidersView.vue` but mixes with the operational provider list
- Queue cap is new (introduced by TASK-189 backend work)
- MCP address and HTTP listen address are currently env-var only (`NEXUS_MCP_ADDR`, `NEXUS_LISTEN_ADDR`)
- The Wails binding and HTTP API expose `AddProviderConfig`, `ListProviderConfigs`, `UpdateProviderConfig`, `RemoveProviderConfig` — these can manage LM Studio / Ollama URL overrides
- `wails.ts` / `domain.ts` has `ProviderConfig` type with `kind`, `name`, `baseURL`, `apiKey`

## Scope

### Files to create
- `frontend/src/views/SettingsView.vue` — settings page

### Files to modify
- `frontend/src/router/index.ts` (or equivalent) — add `/settings` route
- `frontend/src/components/AppSidebar.vue` — add Settings nav link (last item, with gear icon)

## Implementation

1. Create `SettingsView.vue` with three sections:
   - **Provider Connections** — list `ProviderConfig` entries (from `useProviders` composable); inline edit of `baseURL`; form to add a new config using `ProviderConfigForm.vue`
   - **Queue Settings** — read-only display of queue cap value (or editable if TASK-189 exposes a `GET /api/settings` endpoint; otherwise display with note that it requires restart)
   - **Server Addresses** — read-only display of `NEXUS_LISTEN_ADDR` and `NEXUS_MCP_ADDR` with copy-to-clipboard buttons; note that changes require daemon restart
2. Add route `/settings` to router.
3. Add "Settings" entry in `AppSidebar.vue` with gear icon at the bottom of the nav list.

## Acceptance Criteria
- [ ] `SettingsView.vue` exists and is registered at `/settings`
- [ ] AppSidebar has a working "Settings" link with gear icon
- [ ] Provider Connections section lists configured providers and allows adding/removing them
- [ ] No runtime errors when queue cap API is unavailable (graceful degradation)
- [ ] Server addresses section shows current values read from the API (or a reasonable default)
