package bootstrap

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
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
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

// Defaults for Config fields left empty or invalid.
const (
	DefaultDBPath       = "nexus.db"
	DefaultListenAddr   = "127.0.0.1:63987"
	DefaultMCPAddr      = "127.0.0.1:63988"
	DefaultScanInterval = 30 * time.Second
)

// Config holds the startup parameters shared by every nexus entry point.
type Config struct {
	DBPath       string        // SQLite file (NEXUS_DB_PATH)
	ListenAddr   string        // REST API address (NEXUS_LISTEN_ADDR)
	MCPAddr      string        // MCP server address (NEXUS_MCP_ADDR)
	ScanInterval time.Duration // provider re-scan period (NEXUS_SCAN_INTERVAL)
}

// ConfigFromEnv reads the configuration from the environment. A scan interval
// that does not parse or is not positive falls back to the default (a
// non-positive interval would panic time.NewTicker).
func ConfigFromEnv() Config {
	cfg := Config{
		DBPath:       os.Getenv("NEXUS_DB_PATH"),
		ListenAddr:   os.Getenv("NEXUS_LISTEN_ADDR"),
		MCPAddr:      os.Getenv("NEXUS_MCP_ADDR"),
		ScanInterval: DefaultScanInterval,
	}
	if v := os.Getenv("NEXUS_SCAN_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			cfg.ScanInterval = d
		}
	}
	return cfg.withDefaults()
}

func (c Config) withDefaults() Config {
	if c.DBPath == "" {
		c.DBPath = DefaultDBPath
	}
	if c.ListenAddr == "" {
		c.ListenAddr = DefaultListenAddr
	}
	if c.MCPAddr == "" {
		c.MCPAddr = DefaultMCPAddr
	}
	if c.ScanInterval <= 0 {
		c.ScanInterval = DefaultScanInterval
	}
	return c
}

// Runtime is a fully wired nexus process: the orchestration services plus the
// REST and MCP servers, background scanners and the workspace watcher. It is the
// single composition root shared by the desktop app and the headless daemon.
type Runtime struct {
	Config       Config
	Orchestrator *services.OrchestratorService
	Brain        *services.BrainServiceImpl
	Activity     *services.ActivityService
	// Watcher is nil when the filesystem watcher could not be created.
	Watcher *fs_watcher.Watcher

	repo      *repo_sqlite.Repository
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	closeOnce sync.Once
}

// Start opens the database, wires every service and starts the REST API, the
// MCP server and the background scans. The servers run until ctx is cancelled or
// Close is called. The caller must Close the returned Runtime. Log output is
// also fed to logHub (which may be nil) for the /api/logs and /api/events streams.
func Start(ctx context.Context, cfg Config, logHub *httpapi.LogHub) (*Runtime, error) {
	cfg = cfg.withDefaults()

	repo, err := repo_sqlite.New(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	rt := &Runtime{Config: cfg, repo: repo}
	runCtx, cancel := context.WithCancel(ctx)
	rt.cancel = cancel

	// Core services (hexagonal wiring).
	discoverySvc := services.NewDiscoveryService(BuildProviders()...)
	sessionRepo := repo_sqlite.NewSessionRepo(repo)
	orch := services.NewOrchestrator(discoverySvc, repo, fs_writer.New(), sessionRepo,
		services.WithCommandRunner(cmd_runner.New()))
	orch.WithProviderFactory(BuildProviderFromConfig)
	rt.Orchestrator = orch

	providerConfigRepo := repo_sqlite.NewProviderConfigRepo(repo)
	orch.WithProviderConfigRepo(providerConfigRepo)
	orch.WithRuntimeConfigRepo(repo_sqlite.NewRuntimeConfigRepo(repo))
	if rcfg, err := orch.GetRuntimeConfig(runCtx); err != nil {
		log.Printf("startup: get runtime config: %v", err)
	} else if rcfg.QueueCap > 0 {
		orch.WithQueueCap(rcfg.QueueCap)
	}

	aiSessionRepo := repo_sqlite.NewAISessionRepo(repo)
	orch.SetAISessionRepo(aiSessionRepo)

	rt.Brain = services.NewBrainService(repo_sqlite.NewKnowledgeRepo(repo), repo)

	// Real-time workspace change invalidation.
	if w, err := fs_watcher.New(fs_watcher.WithBrain(rt.Brain)); err != nil {
		log.Printf("startup: create fs watcher: %v", err)
	} else {
		rt.Watcher = w
		if cwd, err := os.Getwd(); err == nil {
			if err := w.Watch(cwd); err != nil {
				log.Printf("startup: watch workspace %s: %v", cwd, err)
			} else {
				log.Printf("startup: watching workspace %s for changes", cwd)
			}
		}
	}

	// Activity observatory.
	rt.Activity = services.NewActivityService(repo_sqlite.NewActivityRepo(repo), aiSessionRepo, activityReaders()...)
	rt.Activity.Start()

	// Provider discovery and agent detection.
	scanner := sys_scanner.New()
	orch.WithSystemScanner(scanner)
	orch.SetAgentScanner(scanner)
	orch.SetDiscoveredAgentRepo(repo_sqlite.NewDiscoveredAgentRepo(repo))
	services.WithPlanFileRepo(repo_sqlite.NewPlanFileRepo(repo))(orch)

	// Register every persisted, enabled provider.
	if cfgs, err := providerConfigRepo.ListProviderConfigs(runCtx); err != nil {
		log.Printf("startup: list provider configs: %v", err)
	} else {
		for _, pc := range cfgs {
			if !pc.Enabled {
				continue
			}
			if err := orch.RegisterCloudProvider(pc); err != nil {
				log.Printf("startup: register persisted provider %q: %v", pc.Name, err)
			}
		}
	}

	rt.goRun(func() {
		if err := httpapi.StartServerFull(runCtx, orch, rt.Brain, cfg.ListenAddr, rt.Activity, logHub); err != nil {
			log.Printf("httpapi: %v", err)
		}
	})
	rt.goRun(func() {
		if err := mcp.StartMCPServer(runCtx, orch, rt.Brain, cfg.MCPAddr); err != nil {
			log.Printf("mcp: %v", err)
		}
	})
	rt.goRun(func() { rt.scanLoop(runCtx) })
	return rt, nil
}

func (r *Runtime) goRun(fn func()) { r.wg.Go(fn) }

// scanLoop performs an immediate provider scan, then repeats it every
// Config.ScanInterval until ctx is cancelled.
func (r *Runtime) scanLoop(ctx context.Context) {
	if _, err := r.Orchestrator.TriggerScan(ctx); err != nil {
		log.Printf("startup: initial scan: %v", err)
	}
	ticker := time.NewTicker(r.Config.ScanInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if results, err := r.Orchestrator.TriggerScan(ctx); err != nil {
				log.Printf("discovery: scan error: %v", err)
			} else {
				log.Printf("discovery: found %d providers", len(results))
			}
		case <-ctx.Done():
			return
		}
	}
}

// Close stops the servers and background loops, then the services, then closes
// the database, in that order. It is safe to call more than once.
func (r *Runtime) Close() {
	r.closeOnce.Do(func() {
		r.cancel()
		r.wg.Wait()
		if r.Watcher != nil {
			_ = r.Watcher.Close()
		}
		r.Activity.Stop()
		r.Orchestrator.Stop()
		_ = r.repo.Close()
	})
}

// activityReaders returns the passive activity sources present on this machine:
// Claude and Continue session logs when their folders exist, plus network probes.
func activityReaders() []ports.ActivityReader {
	var readers []ports.ActivityReader
	if homeDir, err := os.UserHomeDir(); err == nil {
		if _, err := os.Stat(filepath.Join(homeDir, ".claude")); err == nil {
			readers = append(readers, activity_claude.NewClaudeJSONLReader(), activity_claude.NewClaudeHistoryReader())
		}
		if _, err := os.Stat(filepath.Join(homeDir, ".continue")); err == nil {
			readers = append(readers, activity_continue.NewContinueSessionReader())
		}
	}
	return append(readers, activity_network.NewNetworkProbeReader())
}

// FormatBanner builds the human- and AI-readable "ready" box printed at startup.
// headline is the first line, for example "v1.2.3 — ready".
func FormatBanner(headline, listenAddr, mcpAddr string) string {
	httpBase := "http://" + listenAddr
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
		headline, httpBase, httpBase+"/ui", httpBase+"/api/howto", httpBase+"/.well-known/nexus.json", "http://"+mcpAddr+"/mcp")
}
