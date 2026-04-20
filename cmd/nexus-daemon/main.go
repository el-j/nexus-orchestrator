// Package main is the entry point for the nexus-daemon binary.
// It runs the full nexus-orchestrator orchestration engine without the desktop GUI,
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

	dbPath := os.Getenv("NEXUS_DB_PATH")
	if dbPath == "" {
		dbPath = "nexus.db"
	}

	// 1. Outbound adapters
	repo, err := repo_sqlite.New(dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "daemon: open database:", err)
		os.Exit(1)
	}

	writer := fs_writer.New()

	// 2. Core services
	discoverySvc := services.NewDiscoveryService(bootstrap.BuildProviders()...)
	sessionRepo := repo_sqlite.NewSessionRepo(repo)
	orchestratorSvc, err := services.NewOrchestrator(discoverySvc, repo, writer, sessionRepo)
	if err != nil {
		repo.Close()
		fmt.Fprintln(os.Stderr, "daemon: init orchestrator:", err)
		os.Exit(1)
	}
	defer repo.Close()
	orchestratorSvc.WithProviderFactory(bootstrap.BuildProviderFromConfig)

	providerConfigRepo := repo_sqlite.NewProviderConfigRepo(repo)
	orchestratorSvc.WithProviderConfigRepo(providerConfigRepo)

	knowledgeRepo := repo_sqlite.NewKnowledgeRepo(repo)
	brainSvc := services.NewBrainService(knowledgeRepo, repo)

	runtimeCfgRepo := repo_sqlite.NewRuntimeConfigRepo(repo)
	orchestratorSvc.WithRuntimeConfigRepo(runtimeCfgRepo)
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

	// 3. Context that cancels on SIGINT / SIGTERM — drives HTTP graceful shutdown
	addr := os.Getenv("NEXUS_LISTEN_ADDR")
	if addr == "" {
		addr = "127.0.0.1:63987"
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mcpAddr := os.Getenv("NEXUS_MCP_ADDR")
	if mcpAddr == "" {
		mcpAddr = "127.0.0.1:63988"
	}
	log.Printf("nexus-daemon %s (%s %s) starting...", version, commit, buildDate)
	// Print a human- and AI-readable ready banner once both servers are about to start.
	go func() {
		// Brief pause so the log context lines printed above appear first.
		time.Sleep(50 * time.Millisecond)
		httpBase := "http://" + addr
		fmt.Printf("\n")
		fmt.Printf("┌────────────────────────────────────────────────────────┐\n")
		fmt.Printf("│  nexus-orchestrator %s — ready                     │\n", version)
		fmt.Printf("├────────────────────────────────────────────────────────┤\n")
		fmt.Printf("│  HTTP API  →  %-39s  │\n", httpBase)
		fmt.Printf("│  Dashboard →  %-39s  │\n", httpBase+"/ui")
		fmt.Printf("│  How-to    →  %-39s  │\n", httpBase+"/api/howto")
		fmt.Printf("│  Discovery →  %-39s  │\n", httpBase+"/.well-known/nexus.json")
		fmt.Printf("│  MCP       →  %-39s  │\n", "http://"+mcpAddr+"/mcp")
		fmt.Printf("└────────────────────────────────────────────────────────┘\n")
		fmt.Printf("\n")
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
		if err := mcp.StartMCPServer(ctx, orchestratorSvc, brainSvc, mcpAddr); err != nil {
			log.Printf("daemon: mcp: %v", err)
		}
	}()

	// StartServerFull blocks until ctx is cancelled, then gracefully shuts down
	if err := httpapi.StartServerFull(ctx, orchestratorSvc, brainSvc, addr, activitySvc, logHub); err != nil {
		log.Printf("daemon: httpapi: %v", err)
	}

	fmt.Println("nexus-orchestrator daemon shutting down.")
}
