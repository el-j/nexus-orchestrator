---
id: TASK-217
title: Fix silent error handling in frontend composables
role: backend
planId: PLAN-030
status: todo
dependencies: [TASK-213]
createdAt: 2025-07-25T00:00:00.000Z
---

## Context
Multiple frontend composables have empty `catch {}` blocks that silently swallow errors from daemon communication, making debugging impossible: `useProviders.ts`, `useLogs.ts`, `useDiscovery.ts`. Additionally, `ProviderStatus.vue` calls `refresh()` without `await`, and `BacklogList.vue` catches promotion errors without showing toast notifications.

## Files to Read
- `frontend/src/composables/useProviders.ts` — lines ~14-15 (silent catch)
- `frontend/src/composables/useLogs.ts` — lines ~28-35 (silent malformed log catch)
- `frontend/src/composables/useDiscovery.ts` — lines ~15-16, ~30-32 (silent catches)
- `frontend/src/components/ProviderStatus.vue` — lines ~94-99 (missing await)
- `frontend/src/components/BacklogList.vue` — lines ~69-78 (no error toast on promote)

## Implementation Steps
1. In `useProviders.ts`: replace `catch { /* silent fail */ }` with `catch (e) { console.warn('Failed to refresh providers:', e) }` and optionally expose an `error` ref
2. In `useLogs.ts`: replace `catch { /* ignore malformed */ }` with `catch (e) { console.debug('Log parse fail:', event.data, e) }` — differentiate malformed data from network errors
3. In `useDiscovery.ts`: add `console.warn` in both `refresh()` and `scanNow()` catch blocks
4. In `ProviderStatus.vue` `handleSave()`: add `await` before `props.refresh?.()` so data refreshes before form closes
5. In `BacklogList.vue` `onPromote()`: add toast notification on catch — use PrimeVue `useToast()` if available, or emit event
6. In `WorkspaceScanner.ts` (VSCode ext): add `console.error` for file I/O failures instead of swallowing silently

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] Zero empty `catch {}` or `catch { /* ... */ }` blocks in frontend composables
- [ ] All daemon communication errors are at least console.warn'd
- [ ] PrimeVue toast shows when promotion fails

## Anti-patterns to Avoid
- NEVER use empty catch blocks — at minimum log the error
- NEVER use `console.log` for errors — use `console.warn` or `console.error`
- NEVER swallow network errors — they indicate daemon connectivity issues
