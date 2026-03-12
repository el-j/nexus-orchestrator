---
id: TASK-165
title: Wails hide-to-tray and build config for console suppression
role: devops
planId: PLAN-023
status: todo
dependencies: [TASK-162]
createdAt: 2026-03-11T21:00:00.000Z
---

## Context
The desktop app currently opens a console window on Windows and quits when the window is closed on any platform. The user wants: close button → hide to tray (app keeps running), no visible console window, and the ability to fully quit only via the tray "Quit" menu item.

## Files to Read
- `main.go` — `wails.Run()` call with current `options.App` config
- `app.go` — lifecycle hooks (currently only `startup`)
- `internal/adapters/inbound/tray/tray.go` — tray adapter from TASK-162
- `build/darwin/Info.plist` — macOS app config
- `build/windows/wails.exe.manifest` — Windows manifest
- `Makefile` — build targets and LDFLAGS

## Implementation Steps

1. **Wails options in `main.go`**: Update `wails.Run()` call to add:
   ```go
   HideWindowOnClose: true,   // close button hides instead of quits
   StartHidden:       false,  // still show on first launch
   ```

2. **OnBeforeClose hook in `main.go`**: Add an `OnBeforeClose` callback that hides the window:
   ```go
   OnBeforeClose: func(ctx context.Context) (prevent bool) {
       // Hide to tray instead of closing
       runtime.WindowHide(ctx)
       return true  // prevent actual close
   },
   ```
   Import `github.com/wailsapp/wails/v2/pkg/runtime` for `runtime.WindowHide`.

3. **Integrate tray adapter in `main.go`**:
   - Construct `TrayAdapter` with `onShow` callback that calls `runtime.WindowShow(ctx)` and `onQuit` callback that calls `runtime.Quit(ctx)`
   - Start the tray adapter before `wails.Run()` in a goroutine (or use the Wails `OnStartup` hook if threading allows)
   - Stop the tray adapter in the `OnShutdown` hook

4. **Update `app.go`** if needed — add the `wailsCtx` reference so tray callbacks can call `runtime.WindowShow(a.ctx)`.

5. **Windows console suppression in `Makefile`**: Update `build-gui` and Windows cross-compile targets to add:
   ```makefile
   LDFLAGS_GUI := -s -w -H windowsgui
   ```
   This tells the Go linker to create a Windows GUI subsystem binary (no console window).

6. **macOS Info.plist**: Keep `LSUIElement: false` for now (user sees app in Dock). The tray icon supplements the Dock icon; it doesn't replace it. If the user later wants a Dock-less agent mode, flip to `true`.

7. **Add startup log message** visible in the in-app log console: `log.Printf("nexusOrchestrator started — hiding window closes to tray")`

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `wails build` succeeds (desktop GUI)
- [ ] Closing the window hides it instead of quitting the process
- [ ] Tray icon appears with "Show Window" and "Quit" menu items
- [ ] "Show Window" from tray restores the hidden window
- [ ] "Quit" from tray fully terminates the app
- [ ] Windows build uses `-H windowsgui` (no console window)

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER call `systray.Run()` on the main thread if Wails already owns it — use goroutine or external loop
- NEVER set `LSUIElement: true` without user consent (removes Dock icon, breaking Cmd+Tab switching)
