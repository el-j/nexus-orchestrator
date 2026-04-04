---
id: TASK-476
plan: PLAN-062
status: done
wave: 2
priority: 2
---

# TASK-476: Replace window.confirm with component-level confirmation dialogs

**Problem:** `SettingsView.vue`, `ProvidersView.vue`, and `ProviderStatus.vue` call `window.confirm()` before destructive actions (delete provider, disable token, etc.). In Wails webview and some Electron-adjacent contexts `window.confirm()` always returns `false` (suppressed by the host), silently blocking or skipping the action. This is also poor UX — native dialogs are unstyled and cannot be themed.

**Fix:**

1. Create `frontend/src/components/AppConfirmDialog.vue` — a Teleport-to-body modal with:
   - Props: `title: string`, `message: string`, `confirmLabel?: string` (default "Confirm"), `destructive?: boolean`
   - Emits: `confirm`, `cancel`
   - Styled with Tailwind; `destructive=true` renders the confirm button in red
2. Create a composable `frontend/src/composables/useConfirmDialog.ts` that exposes:
   - `open(opts: { title, message, confirmLabel?, destructive? }): Promise<boolean>` — resolves `true` on confirm, `false` on cancel
   - Internal `visible`, `resolve` refs wired to a single `AppConfirmDialog` instance mounted in `App.vue`
3. Mount one `<AppConfirmDialog>` instance in `App.vue` connected to `useConfirmDialog`
4. In `SettingsView.vue`: replace every `window.confirm(...)` call with `await confirm({ ... })` from `useConfirmDialog`
5. In `ProvidersView.vue`: same replacement
6. In `ProviderStatus.vue`: same replacement
7. Remove all `window.confirm` calls — `grep -r 'window.confirm' frontend/src` should return empty

**Files:**

- `frontend/src/components/AppConfirmDialog.vue` (new)
- `frontend/src/composables/useConfirmDialog.ts` (new)
- `frontend/src/App.vue`
- `frontend/src/views/SettingsView.vue`
- `frontend/src/views/ProvidersView.vue`
- `frontend/src/components/ProviderStatus.vue`

**Acceptance criteria:**

- `grep -r 'window.confirm' frontend/src` returns zero results
- Destructive actions show the modal before proceeding in both browser and Wails desktop mode
- Pressing Escape or clicking outside the modal cancels the action
- `vue-tsc --noEmit` zero errors
