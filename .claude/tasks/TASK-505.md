---
id: TASK-505
plan: PLAN-065
status: done
wave: 1
priority: 1
---

# TASK-505: Make Wails close semantics fully graceful

## Description

Desktop close behavior currently hides the window while tray support is a no-op, which leaves the app running invisibly. The tray quit callback also uses `os.Exit(0)`, bypassing cleanup. This task makes close/quit deterministic and graceful so when users close the app, nexusOrchestrator is actually shut down.

## Checklist

- [ ] Remove hard-exit (`os.Exit`) quit path from tray callback
- [ ] Route quit behavior through Wails lifecycle so `OnShutdown` and defers run
- [ ] Prevent hide-to-tray behavior when tray implementation is not active
- [ ] Ensure startup logs/messages match real close behavior

## Files

- `main.go`
- `app.go`
- `internal/adapters/inbound/tray/tray.go`

## Acceptance Criteria

- Closing the Wails window exits the process on current tray-stub implementation
- No `os.Exit(0)` is used to quit from tray callback
- Embedded HTTP/MCP servers stop via context cancellation on app shutdown
