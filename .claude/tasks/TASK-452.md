---
id: TASK-452
plan: PLAN-059
status: closed
closedReason: 'stale: planned before v0.10.0 and never started; the work is tracked in PLAN-073 and PLAN-074 or was superseded'
wave: 5
priority: 5
---

# TASK-452: Tray adapter — document as stub

**Problem:** `internal/adapters/inbound/tray/tray.go` is a complete no-op stub. `Start()` logs "systray integration pending". `UpdateStatus()` computes then discards with `_ = fmt.Sprintf()`.

**Fix:** Add clear TODO comments documenting the stub status. Add a build tag or interface-level comment. Leave as-is functionally (systray requires CGO + platform-specific libraries).

**Files:** `internal/adapters/inbound/tray/tray.go`
