---
id: TASK-556
title: Fix frontend critical safety — missing r.ok checks, null-guard, BrainView ingest path, SettingsView loadConfig
role: frontend
planId: PLAN-071
status: done
dependencies: []
createdAt: 2026-04-15T00:00:00Z
---

## Context

Four `r.ok` checks are missing in `wails.ts` dev-fallback paths, meaning HTTP errors silently return malformed data to callers. `ProvidersView.vue` calls `p.baseURL.toLowerCase()` on an optional field — this will throw whenever `baseURL` is `undefined`. `BrainView.vue` passes only `files[i].name` (filename) to `ingestKnowledge` instead of the full path the backend needs. `SettingsView.vue:341` calls `loadConfig()` without await inside `onMounted`, silently swallowing any error.

## Files to Read

- `frontend/src/types/wails.ts`
- `frontend/src/views/ProvidersView.vue`
- `frontend/src/views/BrainView.vue`
- `frontend/src/views/SettingsView.vue`

## Implementation Steps

1. `wails.ts` — in `getTask`, `getQueue`, `getProviders`, and `updateTask` dev-fallback branches: add `if (!r.ok) throw new Error(\`HTTP \${r.status}\`)`before calling`r.json()`
2. `ProvidersView.vue:417` — change `p.baseURL.toLowerCase()` to `(p.baseURL ?? '').toLowerCase()` (or add explicit `if (!p.baseURL) return false` guard)
3. `BrainView.vue:onFilesSelected` — the web file input only exposes `file.name`, not a full path. Change the ingest loop to read the file as text and post content (or `ArrayBuffer`) to the backend instead of passing the filename. Check the backend `IngestKnowledge` signature — if it only accepts a path, add a new endpoint `POST /api/brain/ingest-content` that accepts `{ projectPath, content, fileName }` and wire that as the browser fallback. For Wails mode, file selection should use `runtime.OpenFileDialog` to get the real path and then call `IngestKnowledge`.
4. `SettingsView.vue:341` — change `loadConfig()` to `void loadConfig()` with a `.catch` handler that sets `configError.value`, or simply make the outer `onMounted` async and `await loadConfig()`

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `vue-tsc --noEmit` exits 0 in `frontend/`
- [ ] `npm test` passes in `frontend/`
- [ ] `getTask`, `getQueue`, `getProviders`, `updateTask` all throw on non-ok HTTP responses in dev mode
- [ ] `p.baseURL.toLowerCase()` cannot throw when `baseURL` is undefined
- [ ] BrainView ingest uses Wails `OpenFileDialog` (Wails mode) or content upload (browser mode) — never passes bare filename to backend
- [ ] `loadConfig()` errors surface in `configError.value` in SettingsView

## Anti-patterns to Avoid

- NEVER add `r.ok` checks to Wails binding paths (those throw their own errors from Go)
- NEVER use `as any` to work around missing fields — use optional chaining or explicit guards
