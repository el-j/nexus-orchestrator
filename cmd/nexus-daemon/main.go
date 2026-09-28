// Package main is the entry point for the nexus-daemon binary.
// It runs the full nexusOrchestrator orchestration engine without the desktop GUI,
// suitable for headless server environments or automated workflows.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/adapters/inbound/mcp"
	"nexus-orchestrator/internal/adapters/outbound/activity_claude"
	"nexus-orchestrator/internal/adapters/outbound/activity_continue"
	"nexus-orchestrator/internal/adapters/outbound/activity_network"
	"nexus-orchestrator/internal/adapters/outbound/cmd_runner"
	"nexus-orchestrator/internal/adapters/outbound/fs_watcher"
	"nexus-orchestrator/internal/adapters/outbound/fs_writer"
	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/adapters/outbound/sys_scanner"
	"nexus-orchestrator/internal/bootstrap"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	// 0. Log hub — capture log output for SSE streaming before anything logs.
	logHub := httpapi.NewLogHub()
	log.SetOutput(logHub)

	cfg := resolveConfig()

	// 1. Outbound adapters
	repo, err := repo_sqlite.New(cfg.dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "daemon: open database:", err)
		os.Exit(1)
	}
	defer repo.Close()

	writer := fs_writer.New()
	runner := cmd_runner.New()

	// 2. Core services
	discoverySvc := services.NewDiscoveryService(bootstrap.BuildProviders()...)
	sessionRepo := repo_sqlite.NewSessionRepo(repo)
	orchestratorSvc := services.NewOrchestrator(
		discoverySvc, repo, writer, sessionRepo,
		services.WithCommandRunner(runner),
	)
	orchestratorSvc.WithProviderFactory(bootstrap.BuildProviderFromConfig)

	providerConfigRepo := repo_sqlite.NewProviderConfigRepo(repo)
	orchestratorSvc.WithProviderConfigRepo(providerConfigRepo)

	knowledgeRepo := repo_sqlite.NewKnowledgeRepo(repo)
	brainSvc := services.NewBrainService(knowledgeRepo, repo)

	// 2a. Real-time workspace change invalidation watcher
	fsWatcher, err := fs_watcher.New(fs_watcher.WithBrain(brainSvc))
	if err != nil {
		log.Printf("startup: create fs watcher: %v", err)
	} else {
		defer func() { _ = fsWatcher.Close() }()
		if cwd, err := os.Getwd(); err == nil {
			if err := fsWatcher.Watch(cwd); err != nil {
				log.Printf("startup: watch workspace %s: %v", cwd, err)
			} else {
				log.Printf("startup: watching workspace %s for changes", cwd)
			}
		}
	}

	runtimeCfgRepo := repo_sqlite.NewRuntimeConfigRepo(repo)
	orchestratorSvc.WithRuntimeConfigRepo(runtimeCfgRepo)
	if rcfg, err := orchestratorSvc.GetRuntimeConfig(context.Background()); err != nil {
		log.Printf("startup: get runtime config: %v", err)
	} else if rcfg.QueueCap > 0 {
		orchestratorSvc.WithQueueCap(rcfg.QueueCap)
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
		for _, pcfg := range cfgs {
			if !pcfg.Enabled {
				continue
			}
			if err := orchestratorSvc.RegisterCloudProvider(pcfg); err != nil {
				log.Printf("startup: register persisted provider %q: %v", pcfg.Name, err)
			}
		}
	}
	defer orchestratorSvc.Stop()

	// 3. Context that cancels on SIGINT / SIGTERM — drives HTTP graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("nexus-daemon %s (%s %s) starting...", version, commit, buildDate)
	// Print a human- and AI-readable ready banner once both servers are about to start.
	go func() {
		time.Sleep(50 * time.Millisecond)
		fmt.Print(formatBanner(version, cfg.listenAddr, cfg.mcpAddr))
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
				if results, err := orchestratorSvc.TriggerScan(ctx); err != nil {
					log.Printf("discovery: scan error: %v", err)
				} else {
					log.Printf("discovery: found %d providers", len(results))
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		if err := mcp.StartMCPServer(ctx, orchestratorSvc, brainSvc, cfg.mcpAddr); err != nil {
			log.Printf("daemon: mcp: %v", err)
		}
	}()

	// StartServerFull blocks until ctx is cancelled, then gracefully shuts down
	if err := httpapi.StartServerFull(ctx, orchestratorSvc, brainSvc, cfg.listenAddr, activitySvc, logHub); err != nil {
		log.Printf("daemon: httpapi: %v", err)
	}

	fmt.Println("nexusOrchestrator daemon shutting down.")
}

// daemonConfig encapsulates startup parameters resolved from environment variables.
type daemonConfig struct {
	dbPath       string
	listenAddr   string
	mcpAddr      string
	scanInterval time.Duration
}

func resolveConfig() daemonConfig {
	dbPath := os.Getenv("NEXUS_DB_PATH")
	if dbPath == "" {
		dbPath = "nexus.db"
	}
	addr := os.Getenv("NEXUS_LISTEN_ADDR")
	if addr == "" {
		addr = "127.0.0.1:63987"
	}
	mcpAddr := os.Getenv("NEXUS_MCP_ADDR")
	if mcpAddr == "" {
		mcpAddr = "127.0.0.1:63988"
	}
	scanInterval := 30 * time.Second
	if v := os.Getenv("NEXUS_SCAN_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			scanInterval = d
		}
	}
	return daemonConfig{
		dbPath:       dbPath,
		listenAddr:   addr,
		mcpAddr:      mcpAddr,
		scanInterval: scanInterval,
	}
}

// formatBanner builds the human- and AI-readable startup banner box.
func formatBanner(ver, addr, mcpAddr string) string {
	httpBase := "http://" + addr
	return fmt.Sprintf("\n"+
		"┌────────────────────────────────────────────────────────┐\n"+
		"│  nexusOrchestrator %-35s │\n"+
		"├────────────────────────────────────────────────────────┤\n"+
		"│  HTTP API  →  %-39s  │\n"+
		"│  Dashboard →  %-39s  │\n"+
		"│  How-to    →  %-39s  │\n"+
		"│  Discovery →  %-39s  │\n"+
		"│  MCP       →  %-39s  │\n"+
		"└────────────────────────────────────────────────────────┘\n\n",
		ver+" — ready", httpBase, httpBase+"/ui", httpBase+"/api/howto", httpBase+"/.well-known/nexus.json", "http://"+mcpAddr+"/mcp")
}
