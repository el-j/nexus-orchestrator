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
	"strings"
	"syscall"
	"time"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/adapters/inbound/mcp"
	"nexus-orchestrator/internal/adapters/outbound/fs_writer"
	"nexus-orchestrator/internal/adapters/outbound/llm_anthropic"
	"nexus-orchestrator/internal/adapters/outbound/llm_lmstudio"
	"nexus-orchestrator/internal/adapters/outbound/llm_ollama"
	"nexus-orchestrator/internal/adapters/outbound/llm_openaicompat"
	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/adapters/outbound/sys_scanner"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

var version = "dev"

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
	defer repo.Close()

	writer := fs_writer.New()

	// 2. Core services
	discoverySvc := services.NewDiscoveryService(buildProviders()...)
	sessionRepo := repo_sqlite.NewSessionRepo(repo)
	orchestratorSvc := services.NewOrchestrator(discoverySvc, repo, writer, sessionRepo)
	orchestratorSvc.WithProviderFactory(buildProviderFromConfig)

	providerConfigRepo := repo_sqlite.NewProviderConfigRepo(repo)
	orchestratorSvc.WithProviderConfigRepo(providerConfigRepo)

	aiSessionRepo := repo_sqlite.NewAISessionRepo(repo)
	orchestratorSvc.SetAISessionRepo(aiSessionRepo)

	// Wire system scanner for provider discovery.
	scanner := sys_scanner.New()
	orchestratorSvc.WithSystemScanner(scanner)

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
		addr = "127.0.0.1:9999"
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mcpAddr := os.Getenv("NEXUS_MCP_ADDR")
	if mcpAddr == "" {
		mcpAddr = "127.0.0.1:9998"
	}
	log.Printf("nexus-daemon %s starting...", version)
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
		if err := mcp.StartMCPServer(ctx, orchestratorSvc, mcpAddr); err != nil {
			log.Printf("daemon: mcp: %v", err)
		}
	}()

	// StartServer blocks until ctx is cancelled, then gracefully shuts down
	if err := httpapi.StartServer(ctx, orchestratorSvc, addr, logHub); err != nil {
		log.Printf("daemon: httpapi: %v", err)
	}

	fmt.Println("nexusOrchestrator daemon shutting down.")
}

// buildProviders assembles all configured LLM adapters.
// Local providers are always included; cloud providers require env-var API keys.
func buildProviders() []ports.LLMClient {
	lmStudioURL := os.Getenv("NEXUS_LMSTUDIO_URL")
	if lmStudioURL == "" {
		lmStudioURL = "http://127.0.0.1:1234/v1"
	}
	ollamaURL := os.Getenv("NEXUS_OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = llm_ollama.DefaultBaseURL
	}
	providers := []ports.LLMClient{
		llm_lmstudio.NewLMStudioAdapter(lmStudioURL),
		llm_ollama.NewOllamaAdapter(ollamaURL, "codellama"),
	}
	if key := os.Getenv("NEXUS_OPENAI_API_KEY"); key != "" {
		model := os.Getenv("NEXUS_OPENAI_MODEL")
		if model == "" {
			model = "gpt-4o-mini"
		}
		providers = append(providers, llm_openaicompat.NewAdapter("OpenAI", "https://api.openai.com/v1", key, model))
	}
	if token := os.Getenv("NEXUS_GITHUBCOPILOT_TOKEN"); token != "" {
		model := os.Getenv("NEXUS_GITHUBCOPILOT_MODEL")
		if model == "" {
			model = "gpt-4o"
		}
		providers = append(providers, llm_openaicompat.NewAdapter("GitHub Copilot", "https://api.githubcopilot.com", token, model))
	}
	if key := os.Getenv("NEXUS_ANTHROPIC_API_KEY"); key != "" {
		model := os.Getenv("NEXUS_ANTHROPIC_MODEL")
		if model == "" {
			model = "claude-3-5-sonnet-20241022"
		}
		providers = append(providers, llm_anthropic.NewAdapter(key, model))
	}
	antigravityURL := os.Getenv("NEXUS_ANTIGRAVITY_URL")
	if antigravityURL == "" {
		antigravityURL = "http://127.0.0.1:4315/v1"
	}
	providers = append(providers, llm_openaicompat.NewAdapter("Antigravity", antigravityURL, "", ""))
	return providers
}

// buildProviderFromConfig constructs a single LLM adapter from a runtime ProviderConfig.
// Injected into OrchestratorService to keep the services package free of adapter imports.
func buildProviderFromConfig(cfg domain.ProviderConfig) (ports.LLMClient, error) {
	switch cfg.Kind {
	case domain.ProviderKindLMStudio:
		if cfg.BaseURL == "" {
			cfg.BaseURL = "http://127.0.0.1:1234/v1"
		}
		return llm_lmstudio.NewLMStudioAdapter(cfg.BaseURL), nil
	case domain.ProviderKindOllama:
		if cfg.BaseURL == "" {
			cfg.BaseURL = "http://127.0.0.1:11434"
		}
		return llm_ollama.NewOllamaAdapter(cfg.BaseURL, cfg.Model), nil
	case domain.ProviderKindOpenAICompat:
		return llm_openaicompat.NewAdapter(cfg.Name, cfg.BaseURL, cfg.APIKey, cfg.Model), nil
	case domain.ProviderKindAnthropic:
		return llm_anthropic.NewAdapter(cfg.APIKey, cfg.Model), nil
	case domain.ProviderKindDesktopApp, domain.ProviderKindLocalAI, domain.ProviderKindVLLM, domain.ProviderKindTextGenUI:
		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = "http://127.0.0.1:4315/v1"
		} else if !strings.HasSuffix(baseURL, "/v1") {
			baseURL = strings.TrimRight(baseURL, "/") + "/v1"
		}
		return llm_openaicompat.NewAdapter(cfg.Name, baseURL, cfg.APIKey, cfg.Model), nil
	default:
		return nil, fmt.Errorf("unknown provider kind: %q", cfg.Kind)
	}
}
