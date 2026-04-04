---
id: TASK-452
plan: PLAN-059
status: todo
wave: 5
priority: 5
---

# TASK-452: Tray adapter — document as stub

**Problem:** `internal/adapters/inbound/tray/tray.go` is a complete no-op stub. `Start()` logs "systray integration pending". `UpdateStatus()` computes then discards with `_ = fmt.Sprintf()`.

**Fix:** Add clear TODO comments documenting the stub status. Add a build tag or interface-level comment. Leave as-is functionally (systray requires CGO + platform-specific libraries).

**Files:** `internal/adapters/inbound/tray/tray.go`
