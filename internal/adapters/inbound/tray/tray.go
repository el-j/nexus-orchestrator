// Package tray provides the system tray inbound adapter for the nexusOrchestrator desktop application.
package tray

import (
	"fmt"
	"log"
	"sync"

	"nexus-orchestrator/internal/core/ports"
)

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
func (t *TrayAdapter) Start() {
	log.Printf("tray: adapter created (systray integration pending main-thread coordination)")
}

// UpdateStatus refreshes the tray tooltip with the current provider and queue counts.
// When systray is not yet wired the formatted string is discarded.
func (t *TrayAdapter) UpdateStatus() {
	providers, err := t.orch.GetProviders()
	if err != nil {
		return
	}
	queue, err := t.orch.GetQueue()
	if err != nil {
		return
	}
	_ = fmt.Sprintf("nexusOrchestrator — %d providers, %d tasks queued", len(providers), len(queue))
	// Would update tray tooltip once systray is wired.
}

// Stop signals the tray adapter to shut down. Safe to call multiple times.
func (t *TrayAdapter) Stop() {
	t.once.Do(func() {
		close(t.stopCh)
	})
}
