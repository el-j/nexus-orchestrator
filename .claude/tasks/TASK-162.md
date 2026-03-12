---
id: TASK-162
title: System tray inbound adapter with getlantern/systray
role: backend
planId: PLAN-023
status: todo
dependencies: [TASK-160]
createdAt: 2026-03-11T21:00:00.000Z
---

## Context
The empty `internal/adapters/inbound/tray/` placeholder needs a real implementation. When the user closes the desktop app window, it should hide to the system tray instead of quitting. The tray icon provides a context menu to show the window, view status, or quit. This works on macOS (menu bar), Windows (notification area), and Linux (appindicator).

## Files to Read
- `internal/adapters/inbound/tray/tray.go` — current empty placeholder
- `internal/core/ports/ports.go` — `Orchestrator` interface (for provider/task counts)
- `main.go` — Wails startup, `wails.Run()` options
- `app.go` — lifecycle hooks
- `go.mod` — check if `github.com/getlantern/systray` or similar already in deps

## Implementation Steps

1. Add `github.com/getlantern/systray` to `go.mod`:
   ```sh
   go get github.com/getlantern/systray
   ```

2. Replace `internal/adapters/inbound/tray/tray.go` with a real implementation:
   ```go
   type TrayAdapter struct {
       orch       ports.Orchestrator
       onShow     func()  // callback to show the Wails window
       onQuit     func()  // callback to quit the app
       mStatus    *systray.MenuItem
       mProviders *systray.MenuItem
       mShow      *systray.MenuItem
       mQuit      *systray.MenuItem
       stopCh     chan struct{}
   }
   ```

3. Implement `NewTrayAdapter(orch ports.Orchestrator, onShow func(), onQuit func()) *TrayAdapter`.

4. Implement `Start()` method:
   - Call `systray.Run(onReady, onExit)` — **IMPORTANT**: `systray.Run` blocks and wants the main thread on macOS. We'll need to handle this carefully with Wails (see step 7).
   - In `onReady`:
     - Set icon (embed a small PNG icon as `[]byte`)
     - Set tooltip: "nexusOrchestrator"
     - Add menu items: "Show Window", separator, "Status: Loading...", "Providers: 0 active", separator, "Quit"
   - In a goroutine, listen for menu item clicks via `<-mShow.ClickedCh` and `<-mQuit.ClickedCh`

5. Implement `UpdateStatus()` method — called periodically to refresh tray tooltip/menu:
   - Query `orch.GetProviders()` for active count
   - Query `orch.GetQueue()` for pending task count
   - Update tooltip: "nexusOrchestrator — 2 providers, 3 tasks queued"

6. Implement `Stop()` method — calls `systray.Quit()` and closes `stopCh`.

7. **Platform threading concern**: `systray.Run()` and `wails.Run()` both want the macOS main thread. Resolution strategy:
   - Use `systray.Register(onReady, onExit)` + `systray.RunWithExternalLoop()` if available in the fork
   - OR use the `fyne.io/systray` fork which supports external loop
   - OR run systray in a separate goroutine (works on Windows/Linux, MAY work on macOS with CGO)
   - Document this risk — prototyping needed. The task should produce a working implementation with a commented fallback approach.

8. Embed a minimal 22×22 PNG icon in `internal/adapters/inbound/tray/icon.go` using `//go:embed`.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] Tray icon appears in macOS menu bar when started
- [ ] Context menu has: Show Window, Status line, Provider count, Quit
- [ ] "Show Window" callback invokes the provided `onShow` function
- [ ] "Quit" callback invokes the provided `onQuit` function
- [ ] `Stop()` gracefully removes the tray icon

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/`
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER block the Wails main thread — systray must coexist with the Wails event loop
- NEVER hardcode platform-specific paths without `runtime.GOOS` checks
