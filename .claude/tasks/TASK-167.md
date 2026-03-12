---
id: TASK-167
title: Frontend DiscoveredProvidersPanel and DiscoveryView
role: frontend
planId: PLAN-023
status: todo
dependencies: [TASK-166]
createdAt: 2026-03-11T21:00:00.000Z
---

## Context
The GUI needs to show auto-detected AI providers from the system scan. Users see discovered providers as cards with status indicators (reachable=green, installed=amber, running=amber) and can promote reachable ones to active backends with a single click. A "Scan Now" button triggers immediate re-scan.

## Files to Read
- `frontend/src/types/domain.ts` — existing TypeScript types
- `frontend/src/composables/useProviders.ts` — existing provider composable pattern
- `frontend/src/components/ProviderStatus.vue` — existing provider card pattern
- `frontend/src/components/AppSidebar.vue` — sidebar nav items
- `frontend/src/views/ProvidersView.vue` — existing providers page (if it exists)
- `internal/core/domain/provider.go` — `DiscoveredProvider` Go struct to mirror in TS

## Implementation Steps

1. Add TypeScript types in `frontend/src/types/domain.ts`:
   ```typescript
   export interface DiscoveredProvider {
     id: string
     name: string
     kind: string
     method: 'port' | 'cli' | 'process'
     status: 'reachable' | 'installed' | 'running'
     baseUrl?: string
     cliPath?: string
     processName?: string
     models?: string[]
     lastSeen: string
   }
   ```

2. Create `frontend/src/composables/useDiscovery.ts`:
   - `const discovered = ref<DiscoveredProvider[]>([])`
   - `const scanning = ref(false)`
   - `refresh()` → `GET http://127.0.0.1:9999/api/providers/discovered`
   - `scanNow()` → `POST http://127.0.0.1:9999/api/providers/discovered/scan`
   - `promote(id)` → `POST http://127.0.0.1:9999/api/providers/promote/{id}`
   - SSE listener on `/api/events` for event type `provider_discovered` → auto-refresh
   - Fallback: poll every 10s (discovery is less time-sensitive than tasks)
   - `onMounted` → initial `refresh()`
   - `onUnmounted` → close SSE + clear interval

3. Create `frontend/src/components/DiscoveredProvidersPanel.vue`:
   - Card grid layout (CSS grid, 1-3 columns responsive)
   - Each card shows: provider name, kind badge, method icon (🌐 port, 💻 cli, ⚙️ process)
   - Status color: green border = reachable, amber border = installed/running
   - If reachable: show model count, base URL, "Promote to Active" button
   - If installed: show CLI path, "Installed only — needs API server" label
   - If running: show process name, "Running but no API detected" label
   - "Scan Now" button at the top triggers `scanNow()`, shows spinner while `scanning` is true
   - "Last scanned: X seconds ago" from `lastSeen` timestamp

4. Create `frontend/src/views/DiscoveryView.vue`:
   - Full-page view wrapping `DiscoveredProvidersPanel`
   - Header: "System Discovery" with subtitle "Auto-detected AI providers on this machine"

5. Update `frontend/src/components/AppSidebar.vue`:
   - Add "Discovery" nav item between "Providers" and the bottom spacer
   - Icon: magnifying glass or radar icon
   - Route: `/discovery`

6. Add route in Vue router for `/discovery` → `DiscoveryView`.

## Acceptance Criteria
- [ ] `DiscoveredProvider` TypeScript type matches Go struct
- [ ] `useDiscovery` composable fetches, scans, promotes, and auto-refreshes
- [ ] Cards are color-coded by status
- [ ] "Promote to Active" button only shows for `reachable` providers
- [ ] "Scan Now" button triggers scan and shows spinner
- [ ] Discovery nav item appears in sidebar
- [ ] `/discovery` route loads DiscoveryView

## Anti-patterns to Avoid
- NEVER hardcode API base URL—use the existing pattern from other composables
- NEVER poll faster than 10s for discovery (it's not real-time critical)
- NEVER show "Promote" button for providers that aren't API-reachable
