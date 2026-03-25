# TASK-320: Frontend — DiscoveredPlansView with kind/format badges

**Plan:** PLAN-048
**Role:** frontend
**Dependencies:** TASK-318

## Goal

Add a "Plans" section to the frontend that lists all discovered plan/task/orchestration files grouped by project path, with visual kind badges so users can see at a glance what AI tooling is active in each project.

## Changes

### `frontend/src/views/DiscoveredPlansView.vue` (new file)

- Calls `GET /api/plans/discovered?projectPath=<currentProject>` on mount and every 30s
- Has a "Scan Now" button that POSTs to `/api/plans/discovered/scan`
- Groups results by `projectPath`
- For each plan file shows:
  - Kind badge with color: nexus=indigo, markdown=green, cursor=blue, claude=orange, mcp-config=purple, crewai=pink
  - Format chip: `json` / `md` / `py`
  - Filename (basename of path)
  - `isActive` green dot if modified in last 24h
  - Summary text (truncated to 80 chars)
  - Last modified relative time
  - Clickable path that opens `vscode://file/<path>` if VS Code is available

### `frontend/src/router/index.ts`

Add route:

```ts
{ path: '/plans', name: 'plans', component: () => import('../views/DiscoveredPlansView.vue') }
```

### `frontend/src/App.vue` or sidebar component

Add "Plans" nav item with icon (e.g., `DocumentTextIcon`) linking to `/plans`.

### `frontend/src/composables/useDiscoveredPlans.ts` (new file)

```ts
export function useDiscoveredPlans(projectPath: Ref<string>) {
  const plans = ref<DiscoveredPlanFile[]>([])
  const loading = ref(false)
  async function fetch() { ... GET /api/plans/discovered ... }
  async function scan() { ... POST /api/plans/discovered/scan ... }
  onMounted(() => { fetch(); setInterval(fetch, 30_000) })
  return { plans, loading, fetch, scan }
}
```

### `frontend/src/types/domain.ts`

Add TypeScript types:

```ts
export type PlanFileKind =
  | 'nexus'
  | 'claude-task'
  | 'markdown'
  | 'cursor'
  | 'mcp-config'
  | 'crewai'
  | 'claude';
export interface DiscoveredPlanFile {
  id: string;
  path: string;
  kind: PlanFileKind;
  format: string;
  projectPath: string;
  summary?: string;
  lastModified: string;
  isActive: boolean;
}
```

## Acceptance Criteria

- [ ] `DiscoveredPlansView.vue` renders plan file list with correct badges
- [ ] Route `/plans` works and is reachable from navigation
- [ ] "Scan Now" button triggers a fresh scan
- [ ] TypeScript types added to `domain.ts`
- [ ] `npm run build` in `frontend/` succeeds with no type errors
