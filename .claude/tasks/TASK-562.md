---
id: TASK-562
title: Fix frontend leaks, as-any casts, brittle MCP URL, and missing agentStatusFilter UI
role: frontend
planId: PLAN-071
status: todo
dependencies: [TASK-556]
createdAt: 2026-04-15T00:00:00Z
---

## Context

Timer leaks: `SettingsView.vue` creates `tokenCopyTimer` and `copyTimer` without clearing them in `onUnmounted`; the `useGlobalSSE` module-level `reconnectTimer` is never cleared on app shutdown. Type safety: three `as any` casts in `ProvidersView.vue` bypass union types; `AgentsView.vue` has one. The MCP URL in `SettingsView` is derived by brittle port string-replace. `agentStatusFilter` in `AgentsView` is declared but has no UI toggle chip.

## Files to Read

- `frontend/src/views/SettingsView.vue`
- `frontend/src/composables/useGlobalSSE.ts`
- `frontend/src/views/ProvidersView.vue`
- `frontend/src/views/AgentsView.vue`

## Implementation Steps

1. `SettingsView.vue` — add `onUnmounted(() => { if (tokenCopyTimer) clearTimeout(tokenCopyTimer); if (copyTimer) clearTimeout(copyTimer); })`
2. `useGlobalSSE.ts` — add `reconnectTimer` cleanup to the `disconnect()` function: `if (reconnectTimer) { clearTimeout(reconnectTimer); reconnectTimer = null; }`
3. `SettingsView.vue:345` — replace port string-replace with: `const base = await resolveServerUrl(); resolvedMcpUrl.value = base.replace('63987', '63988') + '/mcp'` is still brittle — instead derive MCP URL from a resolved address by calling `resolveServerUrl()` for both HTTP and using a known offset or a separate `resolveMcpUrl()` helper that reads `VITE_MCP_URL` env or falls back to `base.replace(/:\d+/, ':63988')`
4. `ProvidersView.vue` — replace `statusFilter = opt.v as any` with an explicit type assertion: declare `opt.v` as the union type in the options array type definition, removing the need for `as any`
5. `ProvidersView.vue` — same fix for `toolStatusFilter = opt.v as any`
6. `ProvidersView.vue` — fix `agentLabel({ kind } as any)` to pass a properly typed partial: either change the function signature to accept `{ kind: AgentKind }` or create a typed helper
7. `AgentsView.vue` — fix `sessionStatusFilter = opt.v as any` the same way
8. `AgentsView.vue` — add UI chip buttons for `agentStatusFilter` values (`running`/`stopped`/all) in the agents tab header, matching the filter chip pattern used in the sessions tab

## Acceptance Criteria

- [ ] `vue-tsc --noEmit` exits 0
- [ ] `npm test` passes in `frontend/`
- [ ] Zero `as any` casts remain in `ProvidersView.vue` and `AgentsView.vue` for filter values
- [ ] `SettingsView` timers are cleared in `onUnmounted`
- [ ] `useGlobalSSE.disconnect()` clears `reconnectTimer`
- [ ] MCP URL derivation does not use literal port string-replace
- [ ] `AgentsView` agents tab has visible filter chips for agent status

## Anti-patterns to Avoid

- NEVER use `as any` — declare the narrow union type on the options array instead
- NEVER hardcode port numbers in component logic — use composable helpers
