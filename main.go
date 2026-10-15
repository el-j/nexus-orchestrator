// Command nexus-orchestrator is the nexusOrchestrator desktop application.
// It runs a native GUI via Wails with an embedded HTTP API on :63987 and MCP server on :63988.
package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/bootstrap"
)

var version = "dev"

//go:embed all:build/frontend
var assets embed.FS

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run is the real entry point. Separating it from main() ensures that deferred
// cleanup (rt.Close) always executes before os.Exit.
func run() error {
	// Capture log output for SSE streaming before anything logs.
	logHub := httpapi.NewLogHub()
	log.SetOutput(logHub)

	rt, err := bootstrap.Start(context.Background(), bootstrap.ConfigFromEnv(), logHub)
	if err != nil {
		return fmt.Errorf("fatal: %w", err)
	}
	defer rt.Close()

	app := newApp(rt)
	log.Printf("nexusOrchestrator %s started — closing the window hides it to the dock/taskbar", version)
	fmt.Print(bootstrap.FormatBanner("— ready (GUI)", rt.Config.ListenAddr, rt.Config.MCPAddr))

	if err := wails.Run(windowOptions(app, assets)); err != nil {
		return fmt.Errorf("wails: %w", err)
	}
	return nil
}

// newApp builds the Wails binding object over the running services.
func newApp(rt *bootstrap.Runtime) *App {
	return NewApp(rt.Orchestrator, rt.Config.ListenAddr).
		withActivityService(rt.Activity).
		withBrainService(rt.Brain).
		withFsWatcher(rt.Watcher)
}

// windowOptions describes the desktop window. Closing the window hides it; the
// dock icon (macOS) or taskbar (Windows) reopens it, and Cmd+Q / the OS quit
// mechanism exits the app.
func windowOptions(app *App, frontend fs.FS) *options.App {
	return &options.App{
		Title:             "nexusOrchestrator",
		Width:             1024,
		Height:            768,
		AssetServer:       &assetserver.Options{Assets: frontend},
		HideWindowOnClose: true,
		Bind:              []interface{}{app},
	}
}
