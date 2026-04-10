// Command nexus-orchestrator is the nexusOrchestrator desktop application.
// It runs a native GUI via Wails with an embedded HTTP API on :63987 and MCP server on :63988.
package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/adapters/inbound/mcp"
	"nexus-orchestrator/internal/adapters/inbound/tray"
	"nexus-orchestrator/internal/adapters/outbound/activity_claude"
	"nexus-orchestrator/internal/adapters/outbound/activity_continue"
	"nexus-orchestrator/internal/adapters/outbound/activity_network"
	"nexus-orchestrator/internal/adapters/outbound/fs_writer"
	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/adapters/outbound/sys_scanner"
	"nexus-orchestrator/internal/bootstrap"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
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
// cleanup (repo.Close, orchestratorSvc.Stop) always executes before os.Exit.
func run() error {
	// 0. Log hub — capture log output for SSE streaming before anything logs.
	logHub := httpapi.NewLogHub()
	log.SetOutput(logHub)

	dbPath := os.Getenv("NEXUS_DB_PATH")
	if dbPath == "" {
		dbPath = "nexus.db"
	}

	// 1. Outbound adapters
	repo, err := repo_sqlite.New(dbPath)
	if err != nil {
		return fmt.Errorf("fatal: open database: %w", err)
	}
	defer repo.Close()

	writer := fs_writer.New()

	// 2. Core services (Hexagonal wiring)
	discoverySvc := services.NewDiscoveryService(bootstrap.BuildProviders()...)
	sessionRepo := repo_sqlite.NewSessionRepo(repo)
	orchestratorSvc := services.NewOrchestrator(discoverySvc, repo, writer, sessionRepo)
	orchestratorSvc.WithProviderFactory(bootstrap.BuildProviderFromConfig)

	providerConfigRepo := repo_sqlite.NewProviderConfigRepo(repo)
	orchestratorSvc.WithProviderConfigRepo(providerConfigRepo)

	runtimeConfigRepo := repo_sqlite.NewRuntimeConfigRepo(repo)
	orchestratorSvc.WithRuntimeConfigRepo(runtimeConfigRepo)
	if cfg, err := orchestratorSvc.GetRuntimeConfig(context.Background()); err != nil {
		log.Printf("startup: get runtime config: %v", err)
	} else if cfg.QueueCap > 0 {
		orchestratorSvc.WithQueueCap(cfg.QueueCap)
	}

	aiSessionRepo := repo_sqlite.NewAISessionRepo(repo)
	orchestratorSvc.SetAISessionRepo(aiSessionRepo)

	// 2b. Activity observatory
	activityRepo := repo_sqlite.NewActivityRepo(repo)
	var activityReaders []ports.ActivityReader
	if homeDir, err := os.UserHomeDir(); err == nil {
		if _, err := os.Stat(filepath.Join(homeDir, ".claude")); err == nil {
			activityReaders = append(activityReaders, activity_claude.NewClaudeJSONLReader(), activity_claude.NewClaudeHistoryReader())
		}
		if _, err := os.Stat(filepath.Join(homeDir, ".continue")); err == nil {
			activityReaders = append(activityReaders, activity_continue.NewContinueSessionReader())
		}
	}
	activityReaders = append(activityReaders, activity_network.NewNetworkProbeReader())
	activitySvc := services.NewActivityService(activityRepo, aiSessionRepo, activityReaders...)
	activitySvc.Start()
	defer activitySvc.Stop()

	// Wire system scanner for provider discovery + agent detection.
	scanner := sys_scanner.New()
	orchestratorSvc.WithSystemScanner(scanner)
	orchestratorSvc.SetAgentScanner(scanner)
	discoveredAgentRepo := repo_sqlite.NewDiscoveredAgentRepo(repo)
	orchestratorSvc.SetDiscoveredAgentRepo(discoveredAgentRepo)
	planFileRepo := repo_sqlite.NewPlanFileRepo(repo)
	services.WithPlanFileRepo(planFileRepo)(orchestratorSvc)

	// Load persisted provider configs and register each enabled one.
	if cfgs, err := providerConfigRepo.ListProviderConfigs(context.Background()); err != nil {
		log.Printf("startup: list provider configs: %v", err)
	} else {
		for _, cfg := range cfgs {
			if !cfg.Enabled {
				continue
			}
			if err := orchestratorSvc.RegisterCloudProvider(cfg); err != nil {
				log.Printf("startup: register persisted provider %q: %v", cfg.Name, err)
			}
		}
	}
	defer orchestratorSvc.Stop()

	// 3. Start HTTP API and MCP server with a cancellable context
	httpCtx, cancelHTTP := context.WithCancel(context.Background())
	httpAddr := os.Getenv("NEXUS_LISTEN_ADDR")
	if httpAddr == "" {
		httpAddr = "127.0.0.1:63987"
	}
	mcpAddr := os.Getenv("NEXUS_MCP_ADDR")
	if mcpAddr == "" {
		mcpAddr = "127.0.0.1:63988"
	}
	go func() {
		if err := httpapi.StartServerFull(httpCtx, orchestratorSvc, httpAddr, activitySvc, logHub); err != nil {
			log.Printf("httpapi: %v", err)
		}
	}()
	go func() {
		if err := mcp.StartMCPServer(httpCtx, orchestratorSvc, mcpAddr); err != nil {
			log.Printf("mcp: %v", err)
		}
	}()

	// Initial non-blocking scan.
	go func() {
		if _, err := orchestratorSvc.TriggerScan(context.Background()); err != nil {
			log.Printf("startup: initial scan: %v", err)
		}
	}()
	// Periodic re-scan.
	scanInterval := 30 * time.Second
	if v := os.Getenv("NEXUS_SCAN_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			scanInterval = d
		}
	}
	go func() {
		ticker := time.NewTicker(scanInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if results, err := orchestratorSvc.TriggerScan(httpCtx); err != nil {
					log.Printf("discovery: scan error: %v", err)
				} else {
					log.Printf("discovery: found %d providers", len(results))
				}
			case <-httpCtx.Done():
				return
			}
		}
	}()

	// 4. Initialise Wails app binding
	app := NewApp(orchestratorSvc, httpAddr).WithActivityService(activitySvc)

	trayAdapter := tray.NewTrayAdapter(orchestratorSvc, func() {
		app.ShowWindow()
	}, func() {
		app.QuitApp()
	})
	trayEnabled := trayAdapter.Enabled()

	if trayEnabled {
		log.Printf("nexusOrchestrator started — closing window hides to tray")
	} else {
		log.Printf("nexusOrchestrator started — closing window quits the app")
	}
	// Print a human- and AI-readable ready banner.
	httpBase := "http://" + httpAddr
	fmt.Printf("\n")
	fmt.Printf("┌────────────────────────────────────────────────────────┐\n")
	fmt.Printf("│  nexusOrchestrator — ready (GUI)                       │\n")
	fmt.Printf("├────────────────────────────────────────────────────────┤\n")
	fmt.Printf("│  HTTP API  →  %-39s  │\n", httpBase)
	fmt.Printf("│  Dashboard →  %-39s  │\n", httpBase+"/ui")
	fmt.Printf("│  How-to    →  %-39s  │\n", httpBase+"/api/howto")
	fmt.Printf("│  Discovery →  %-39s  │\n", httpBase+"/.well-known/nexus.json")
	fmt.Printf("│  MCP       →  %-39s  │\n", "http://"+mcpAddr+"/mcp")
	fmt.Printf("└────────────────────────────────────────────────────────┘\n")
	fmt.Printf("\n")

	// 5. Launch Wails desktop window
	if err := wails.Run(&options.App{
		Title:  "nexusOrchestrator",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		HideWindowOnClose: trayEnabled,
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			if !trayEnabled {
				log.Printf("nexusOrchestrator: window close requested — shutting down")
				return false
			}
			log.Printf("nexusOrchestrator: window close intercepted — hiding to tray")
			runtime.WindowHide(ctx)
			return true
		},
		OnStartup: func(ctx context.Context) {
			app.startup(ctx)
			trayAdapter.Start()
		},
		OnShutdown: func(_ context.Context) {
			trayAdapter.Stop()
			cancelHTTP()
		},
		Bind: []interface{}{
			app,
		},
	}); err != nil {
		return fmt.Errorf("wails: %w", err)
	}
	return nil
}
