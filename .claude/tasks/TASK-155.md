---
id: TASK-155
title: GUI — AISessionsView.vue + useAISessions composable + sidebar nav
role: architecture
planId: PLAN-022
status: todo
dependencies: [TASK-153]
priority: high
estimated_effort: L
createdAt: 2026-03-12T11:00:00.000Z
---

## Goal
Add a new "AI Sessions" view to the Wails GUI that shows all discovered external AI agent sessions in real-time, reusing the existing SSE event bus and following the exact same patterns as the existing `DashboardView` / `ProvidersView`.

## Context
The frontend follows Vue 3 + Vite + Tailwind (dark theme: `bg-slate-900/800/700`). The existing pattern:
- Composables in `frontend/src/composables/` — `useTasks.ts` (SSE + fallback polling), `useProviders.ts` (polling)
- Views in `frontend/src/views/`
- `domain.ts` for TypeScript types (camelCase matching Go JSON tags)
- `wails.ts` for HTTP-with-Wails-fallback helpers
- `App.vue` switches views based on `currentView` ref (now `dashboard | providers`)
- `AppSidebar.vue` has a `navItems` array; emits `view-change` events

The new `AISession` Go type (from TASK-150) serialises with these JSON keys:
`id`, `source`, `externalId`, `agentName`, `projectPath`, `status`, `lastActivity`, `routedTaskIds`, `createdAt`, `updatedAt`

## Scope

### Files to create
- `frontend/src/types/domain.ts` — extend with `AISession`, `AISessionStatus`, `AISessionSource` TypeScript interfaces (append to existing file, do NOT rewrite it)
- `frontend/src/composables/useAISessions.ts`
- `frontend/src/views/AISessionsView.vue`

### Files to modify
- `frontend/src/components/AppSidebar.vue` — add "AI Sessions" nav item
- `frontend/src/App.vue` — add `v-else-if="currentView === 'ai-sessions'"` branch + import
- `frontend/src/types/wails.ts` — add `listAISessions()` and `deregisterAISession(id)` HTTP helpers

## Implementation Steps

### 1. domain.ts — append types
At the end of `frontend/src/types/domain.ts`, append:
```typescript
export type AISessionSource = 'mcp' | 'vscode' | 'http'
export type AISessionStatus = 'active' | 'idle' | 'disconnected'

export interface AISession {
  id: string
  source: AISessionSource
  externalId?: string
  agentName: string
  projectPath?: string
  status: AISessionStatus
  lastActivity: string
  routedTaskIds?: string[]
  createdAt: string
  updatedAt: string
}
```

### 2. wails.ts — add helpers
Following the exact HTTP-wrapper + Wails-fallback pattern:
- `listAISessions(): Promise<AISession[]>` — GET `/api/ai-sessions`
- `deregisterAISession(id: string): Promise<void>` — DELETE `/api/ai-sessions/{id}`
When `window.go?.main?.App` has a corresponding binding use it; otherwise fall through to HTTP fetch. (In practice no Wails binding exists for this yet — HTTP-only is fine.)

### 3. useAISessions.ts — composable
Pattern: identical to `useTasks.ts` (SSE-first, polling fallback).
Returns: `sessions: Ref<AISession[]>`, `loading: Ref<boolean>`, `error: Ref<string|null>`, `deregister(id: string): Promise<void>`.
- On mount: fetch `listAISessions()`, then open `EventSource('http://127.0.0.1:9999/api/events')`
- On SSE event with `type === 'ai_session_changed'`: call `listAISessions()` to refresh
- On `onerror`: close EventSource, fall back to `setInterval(refresh, 5000)`
- `deregister(id)` calls `deregisterAISession(id)` then refreshes

### 4. AISessionsView.vue
Dark theme, Tailwind. Structure:
- **Header**: "AI Sessions" h1, active count badge (`sessions.filter(s => s.status === 'active').length`), refresh button
- **Empty state** (when `sessions.length === 0`): centered icon + text "No AI sessions detected" + subtext "Connect VS Code Copilot or an MCP client to see sessions here."
- **Session cards** (v-for loop):
  - Agent name + source pill (color-coded: mcp=purple, vscode=blue, http=slate)
  - Status dot: emerald pulse = active, amber = idle, red = disconnected
  - Project path (monospace, truncated with `title` tooltip)
  - Last activity: relative time display (use `new Date(s.lastActivity).toLocaleString()`)
  - Routed tasks: inline `TaskStatusBadge` components if `routedTaskIds` is non-empty
  - "Disconnect" button: calls `deregister(s.id)`, disabled when `status === 'disconnected'`

### 5. AppSidebar.vue — nav item
Add to the `navItems` array (between existing dashboard and providers items):
```
{ id: 'ai-sessions', label: 'AI Sessions', icon: '🤖' }
```
Follow the exact pattern used for the existing nav items.

### 6. App.vue — add view branch
- Import `AISessionsView` from `./views/AISessionsView.vue`
- Add `v-else-if="currentView === 'ai-sessions'"` between dashboard and providers branches

## Acceptance Criteria
- [ ] `go vet ./...` exits 0 (Go unchanged, but verify frontend types don't conflict)
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `domain.ts` has `AISession`, `AISessionStatus`, `AISessionSource` TypeScript types
- [ ] Clicking "AI Sessions" in the sidebar switches to `AISessionsView`
- [ ] Empty state renders when session list is empty
- [ ] Session cards show: agentName, source pill, status dot, projectPath, lastActivity
- [ ] "Disconnect" button calls `DELETE /api/ai-sessions/{id}` and refreshes the list
- [ ] SSE event `ai_session_changed` triggers list refresh without a page reload

## Anti-patterns to Avoid
- NEVER rewrite domain.ts from scratch — only append the new types
- NEVER hardcode the daemon URL beyond `http://127.0.0.1:9999` (consistent with useTasks.ts)
- NEVER add polling-only when SSE is available
- NEVER import from Vue Router — this project uses manual view switching via emits, not vue-router
- NEVER use `<script>` (Options API) — use `<script setup>` (Composition API) throughout
