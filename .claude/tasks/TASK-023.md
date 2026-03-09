---
id: TASK-023
title: System tray — show/hide window + quick stats + task notifications
role: devops
planId: PLAN-002
status: todo
dependencies: [TASK-019, TASK-020]
createdAt: 2026-03-09T12:00:00.000Z
---

## Context

`internal/adapters/inbound/tray/tray.go` exists but is a stub. On macOS and Linux, Wails supports a system tray via `wailsruntime.MenuSetApplicationMenu` and the `systray` library. The tray should show: the Nexus icon, current queue depth in the title, a "Show/Hide" toggle, a quick "Submit Task" action, and desktop notifications when a task completes or fails.

## Files to Read

- `internal/adapters/inbound/tray/tray.go` — current stub
- `app.go` — App struct and Wails context setup
- `main.go` — how Wails app is started (look for `OnStartup`, `OnDomReady` hooks)
- `go.mod` — check if systray/notification library is already a dependency

## Implementation Steps

1. **Choose tray library**: Use `github.com/fyne-io/systray` (already common with Wails) or Wails built-in menu system via `github.com/wailsapp/wails/v2/pkg/menu`. Prefer Wails native menus to avoid CGO complications.

2. **Implement `tray.go`** using Wails `*menu.Menu`:
   ```go
   package tray

   import (
       "github.com/wailsapp/wails/v2/pkg/menu"
       "github.com/wailsapp/wails/v2/pkg/menu/keys"
       "github.com/wailsapp/wails/v2/pkg/runtime"
   )

   // BuildMenu returns the application menu (macOS menu bar + tray).
   func BuildMenu(app interface{ GetQueue() []any }) *menu.Menu {
       appMenu := menu.NewMenu()
       fileMenu := appMenu.AddSubmenu("Nexus")
       fileMenu.AddText("Show Window", keys.CmdOrCtrl("u"), func(_ *menu.CallbackData) {
           // runtime.WindowShow(ctx) — ctx passed via closure
       })
       fileMenu.AddSeparator()
       fileMenu.AddText("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
           // runtime.Quit(ctx)
       })
       return appMenu
   }
   ```

3. **Queue depth in title**: In `app.go`'s `OnDomReady`, start a goroutine that polls `GetStats()` every 5 seconds and calls `runtime.WindowSetTitle(ctx, fmt.Sprintf("nexusOrchestrator [%d]", stats.QueueDepth))`.

4. **Task completion desktop notification**: After `OrchestratorService` finishes a task, emit a Wails event `"task:completed"` or `"task:failed"` via `runtime.EventsEmit`. The frontend listens with `EventsOn("task:completed", callback)` and shows a toast (already built in TASK-020).
   - Backend: Wails events require the app context — pass `ctx` from `app.go` into the orchestrator callback, or expose via an event emitter port.
   - Alternative (simpler): Frontend polls `GetQueue()` and detects when a task disappears from the queue — shows a toast for the transition.

5. **Wire tray in `main.go`**: Pass `BuildMenu(a)` to `wails.Run(options)` via `options.Menu`.

6. **Platform notes**:
   - macOS: Menu bar item visible automatically.
   - Linux: Requires GTK; Wails handles this via cgo.
   - Windows: System tray from `github.com/wailsapp/wails/v2/pkg/runtime` `TraySetMenu` (if available in v2.11.0).

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build .` exits 0 (Wails binary)
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `tray/tray.go` no longer a stub — implements `BuildMenu` returning `*menu.Menu`
- [ ] Window title updates with queue depth when tasks are queued/completed
- [ ] Quit menu item calls `runtime.Quit(ctx)`
- [ ] Task completion toast appears in GUI when task transitions from processing → completed/failed

## Anti-patterns to Avoid

- NEVER block the Wails main goroutine with polling — use background goroutine
- NEVER import `internal/core/services` from tray — accept only `ports.Orchestrator`
- NEVER use a third-party systray library unless Wails built-in menus cannot achieve the goal
- NEVER emit Wails events from inside `internal/core/services/` — emit from `app.go` which holds the Wails context
