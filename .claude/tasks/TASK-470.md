---
id: TASK-470
plan: PLAN-062
status: done
wave: 1
priority: 1
---

# TASK-470: Fix wails.ts getRuntimeConfig/updateRuntimeConfig — broken in desktop mode

**Problem:** `frontend/src/wailsjs/wails.ts` exposes `getRuntimeConfig()` and `updateRuntimeConfig()` as plain `fetch('/api/config')` calls with no `isWails()` guard. In Wails desktop mode there is no HTTP server reachable at a bare `/api/config` path, so the calls 404 silently and `SettingsView.vue` renders the entire security/token section with stale or empty data. The `Window.go.main.App` interface in `wails.ts` has no `GetRuntimeConfig` or `UpdateRuntimeConfig` entries, meaning the generated Wails JS bindings are never wired up.

**Fix:**

1. Add `GetRuntimeConfig(): Promise<RuntimeConfig>` to the `App` interface in `frontend/src/wailsjs/wails.ts`
2. Add `UpdateRuntimeConfig(config: RuntimeConfig): Promise<void>` to the same `App` interface
3. Update `getRuntimeConfig()` wrapper: if `isWails()` call `window.go.main.App.GetRuntimeConfig()`, else `fetch('/api/config')` with JSON parse
4. Update `updateRuntimeConfig()` wrapper: if `isWails()` call `window.go.main.App.UpdateRuntimeConfig(config)`, else `fetch('/api/config', { method: 'PUT', body: JSON.stringify(config) })`
5. Add `RuntimeConfig` type export to `frontend/src/types/wails.ts` (or domain.ts if shared) with at least `{ tokenEnabled: boolean; token: string; listenAddr: string; mcpAddr: string }`
6. Verify `SettingsView.vue` imports and calls via the wails wrapper (not direct fetch) — update import if needed
7. Add a Go method `GetRuntimeConfig() (RuntimeConfig, error)` and `UpdateRuntimeConfig(config RuntimeConfig) error` in `app.go` (Wails binding layer) that delegates to the HTTP config handler logic
8. Run `wails generate module` (or manually sync the JS binding stub) so `window.go.main.App` reflects the new methods
9. Test in browser mode: network tab shows `GET /api/config` 200
10. Test in Wails dev mode: no 404 in Go stdout, settings loads correctly

**Files:**

- `frontend/src/wailsjs/wails.ts`
- `frontend/src/types/wails.ts` (or `domain.ts`)
- `frontend/src/views/SettingsView.vue`
- `app.go`

**Acceptance criteria:**

- `isWails()` branch calls `window.go.main.App.GetRuntimeConfig()` / `UpdateRuntimeConfig()`
- Browser (HTTP) mode falls back to `fetch('/api/config')` as before
- `vue-tsc --noEmit` has no type errors on `RuntimeConfig`
- SettingsView security section renders token state correctly in both modes
