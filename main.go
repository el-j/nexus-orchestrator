// Command nexus-orchestrator is the nexusOrchestrator desktop application.
// It runs a native GUI via Wails with an embedded HTTP API on :63987 and MCP server on :63988.
package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/adapters/outbound/httpapi_client"
	"nexus-orchestrator/internal/bootstrap"
)

var version = "dev"

//go:embed all:build/frontend
var assets embed.FS

var probeClient = &http.Client{Timeout: 500 * time.Millisecond}

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

	cfg := bootstrap.ConfigFromEnv()
	app, cleanup, err := attachOrStart(context.Background(), cfg, logHub)
	if err != nil {
		return fmt.Errorf("fatal: %w", err)
	}
	defer cleanup()

	log.Printf("nexusOrchestrator %s started — closing the window hides it to the dock/taskbar", version)
	fmt.Print(bootstrap.FormatBanner("— ready (GUI)", cfg.ListenAddr, cfg.MCPAddr))

	if err := wails.Run(windowOptions(app, assets)); err != nil {
		return fmt.Errorf("wails: %w", err)
	}
	return nil
}

// attachOrStart connects the desktop app to an already-running Nexus daemon on
// cfg.ListenAddr when one responds to /api/health, or starts an embedded runtime.
func attachOrStart(ctx context.Context, cfg bootstrap.Config, logHub *httpapi.LogHub) (*App, func(), error) {
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = bootstrap.DefaultListenAddr
	}
	if cfg.MCPAddr == "" {
		cfg.MCPAddr = bootstrap.DefaultMCPAddr
	}
	if daemonHealthy(ctx, cfg.ListenAddr) {
		baseURL := "http://" + cfg.ListenAddr
		log.Printf("startup: attaching to running nexus-daemon at %s", baseURL)
		client := httpapi_client.NewClient(baseURL)
		app := NewApp(client, cfg.ListenAddr).
			withActivityService(client).
			withBrainService(httpapi_client.NewBrainClient(baseURL))
		return app, func() {}, nil
	}
	rt, err := bootstrap.Start(ctx, cfg, logHub)
	if err != nil {
		return nil, nil, err
	}
	return newApp(rt), rt.Close, nil
}

func daemonHealthy(ctx context.Context, listenAddr string) bool {
	if strings.HasSuffix(listenAddr, ":0") {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+listenAddr+"/api/health", nil)
	if err != nil {
		return false
	}
	resp, err := probeClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	var body struct {
		Status string `json:"status"`
	}
	return json.NewDecoder(resp.Body).Decode(&body) == nil && body.Status == "ok"
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
