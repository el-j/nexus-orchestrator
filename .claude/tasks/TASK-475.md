---
id: TASK-475
plan: PLAN-062
status: done
wave: 2
priority: 2
---

# TASK-475: Fix SettingsView — hardcoded server addresses and silent token operation failures

**Problem:** `frontend/src/views/SettingsView.vue` has `serverAddresses` defined as a static hardcoded array (e.g., `['http://127.0.0.1:63987', 'http://localhost:63987']`) instead of reading the actual runtime config. `rotateToken` and `disableToken` have `catch { /* silent fail */ }` — the user receives no feedback on failure, leaving the UI in an inconsistent state.

**Fix:**

1. Remove the hardcoded `serverAddresses` array; replace with a `resolvedAddress` ref populated by calling `resolveServerUrl()` (or equivalent utility already in the codebase) on mount
2. If `resolveServerUrl()` does not exist, create a small utility in `frontend/src/utils/serverUrl.ts` that tries the known addresses in order and returns the first that responds to `GET /health`
3. In `rotateToken`: wrap the API call in try/catch; on catch, call the shared toast/notification system (or `emit('error', ...)`) with message `"Failed to rotate token: <err.message>"`
4. In `disableToken`: same — replace `catch { /* silent fail */ }` with a user-facing toast
5. Add `const tokenError = ref<string | null>(null)` and render it as a red inline error below the token action buttons as a fallback if a global toast is unavailable
6. Ensure `rotateToken` and `disableToken` set a `loading` state that disables the buttons during the async operation to prevent double-submit

**Files:**

- `frontend/src/views/SettingsView.vue`
- `frontend/src/utils/serverUrl.ts` (new, if `resolveServerUrl` does not already exist)

**Acceptance criteria:**

- Server address displayed in settings is fetched from runtime config, not a static literal
- Token rotate/disable failures produce a visible error message (toast or inline)
- Token buttons are disabled while async operation is in-flight
- `vue-tsc --noEmit` zero errors
