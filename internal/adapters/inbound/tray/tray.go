// Package tray provides the system tray inbound adapter for the nexusOrchestrator
// desktop application. It is currently a stub: the full systray integration is
// blocked on macOS main-thread coordination with Wails. All methods are safe to
// call and satisfy the adapter interface, but no tray icon is displayed.
package tray

import (
	"fmt"
	"log"
	"sync"

	"nexus-orchestrator/internal/core/ports"
)

// trayDisabledOnce ensures the "tray disabled" log line is emitted at most once
// per process, regardless of how many times Start() or Enabled() is called.
var trayDisabledOnce sync.Once

// TrayAdapter manages the system tray icon.
// On platforms where systray cannot acquire the main thread alongside Wails,
// Start() is a documented no-op — the tray can be enabled by a future
// platform-specific main-thread integration.
type TrayAdapter struct {
	orch   ports.Orchestrator
	onShow func()
	onQuit func()
	stopCh chan struct{}
	once   sync.Once
}

// NewTrayAdapter creates a TrayAdapter wired to the given orchestrator.
// onShow is called when the user clicks "Show" in the tray menu;
// onQuit is called when the user clicks "Quit".
func NewTrayAdapter(orch ports.Orchestrator, onShow func(), onQuit func()) *TrayAdapter {
	return &TrayAdapter{
		orch:   orch,
		onShow: onShow,
		onQuit: onQuit,
		stopCh: make(chan struct{}),
	}
}

// Start initialises the tray adapter.
// Platform threading note: systray.Run() requires the OS main thread on macOS,
// which is already owned by wails.Run(). Full systray integration requires either
// wails.Run() to yield or a platform-specific native bridge. For now, Start() is
// intentionally a no-op — the app functions without a tray icon and the interface
// is ready for future integration.
//
// TODO(#XXX): implement system tray — requires main-thread dispatch on macOS/Windows
// (e.g. systray library); currently blocked by Wails owning the OS main thread.
func (t *TrayAdapter) Start() {
	trayDisabledOnce.Do(func() {
		log.Println("tray icon disabled (NEXUS_TRAY_ENABLED not set); system tray is not functional")
	})
}

// Enabled reports whether a real tray integration is active.
func (t *TrayAdapter) Enabled() bool {
	return false
}

// UpdateStatus refreshes the tray tooltip with the current provider and queue counts.
// TODO(tray): use the formatted string to update systray tooltip once wired.
func (t *TrayAdapter) UpdateStatus() {
	providers, err := t.orch.GetProviders()
	if err != nil {
		return
	}
	queue, err := t.orch.GetQueue()
	if err != nil {
		return
	}
	// Formatted for future systray tooltip; currently discarded.
	_ = fmt.Sprintf("nexusOrchestrator — %d providers, %d tasks queued", len(providers), len(queue))
	// Would update tray tooltip once systray is wired.
}

// Stop signals the tray adapter to shut down. Safe to call multiple times.
func (t *TrayAdapter) Stop() {
	t.once.Do(func() {
		close(t.stopCh)
	})
}
