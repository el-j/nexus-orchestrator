# PLAN-060 — Vue Router Migration

**Status:** active
**Created:** 2026-03-29
**Author:** copilot

## Goal

Replace the manual `currentView` ref + v-if chain in App.vue with proper Vue Router navigation. All views become named routes. The sidebar uses `useRouter().push()`. App.vue becomes a clean `<RouterView />` shell.

## Route Map

| Name              | Path                      | Component           |
| ----------------- | ------------------------- | ------------------- |
| `mission-control` | `/`                       | MissionControlView  |
| `agents`          | `/agents`                 | AgentsView          |
| `providers`       | `/providers`              | ProvidersView       |
| `projects`        | `/projects`               | ProjectActivityView |
| `live-activity`   | `/projects/live-activity` | LiveActivityView    |
| `plans`           | `/plans`                  | DiscoveredPlansView |
| `history`         | `/history`                | HistoryView         |
| `backlog`         | `/backlog`                | BacklogView         |
| `settings`        | `/settings`               | SettingsView        |

Use `createWebHashHistory` (works in Wails desktop + file:// without server config).

## Waves

### Wave 1 — Router module (TASK-454, TASK-455)

| Task     | Description                                                       |
| -------- | ----------------------------------------------------------------- |
| TASK-454 | Write `router/index.ts` — createRouter + all routes (lazy-loaded) |
| TASK-455 | Wire router into `main.ts`                                        |

### Wave 2 — Sidebar + App shell (TASK-456, TASK-457)

| Task     | Description                                                                           |
| -------- | ------------------------------------------------------------------------------------- |
| TASK-456 | Refactor AppSidebar: drop emit, use useRouter().push() + useRoute() for active state  |
| TASK-457 | Simplify App.vue: remove currentView ref + all v-if chain, keep only `<RouterView />` |

### Wave 3 — View cleanups (TASK-458)

| Task     | Description                                                                                                         |
| -------- | ------------------------------------------------------------------------------------------------------------------- |
| TASK-458 | ProjectActivityView: replace `emit('navigate', 'live-activity')` with `useRouter().push('/projects/live-activity')` |

### Wave 4 — Test fixes (TASK-459)

| Task     | Description                                                                                                             |
| -------- | ----------------------------------------------------------------------------------------------------------------------- |
| TASK-459 | Fix MissionControlView + other spec files: provide router mock via createTestingPinia / createRouter stubs where needed |

## Validation

- `vue-tsc --noEmit` zero errors
- All Vitest tests pass (25+)
- `go vet ./...` clean (backend unaffected)
