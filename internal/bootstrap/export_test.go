package bootstrap

import (
	"context"

	"nexus-orchestrator/internal/adapters/outbound/fs_watcher"
)

// SetNewWatcher replaces how Start creates the workspace watcher and returns a function that restores it.
func SetNewWatcher(f func(...fs_watcher.Option) (*fs_watcher.Watcher, error)) (restore func()) {
	old := newWatcher
	newWatcher = f
	return func() { newWatcher = old }
}

// ScanLoop runs the provider scan loop on its own.
func (r *Runtime) ScanLoop(ctx context.Context) { r.scanLoop(ctx) }
