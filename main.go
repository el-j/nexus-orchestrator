// Command nexus-orchestrator is the nexusOrchestrator desktop application.
// It runs a native GUI via Wails with an embedded HTTP API on :9999 and MCP server on :9998.
package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/adapters/inbound/mcp"
	"nexus-orchestrator/internal/adapters/outbound/fs_writer"
	"nexus-orchestrator/internal/adapters/outbound/llm_anthropic"
	"nexus-orchestrator/internal/adapters/outbound/llm_lmstudio"
	"nexus-orchestrator/internal/adapters/outbound/llm_ollama"
	"nexus-orchestrator/internal/adapters/outbound/llm_openaicompat"
	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

var version = "dev"

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	dbPath := os.Getenv("NEXUS_DB_PATH")
	if dbPath == "" {
		dbPath = "nexus.db"
	}

	// 1. Outbound adapters
	repo, err := repo_sqlite.New(dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fatal: open database:", err)
		os.Exit(1)
	}
	defer repo.Close()

	writer := fs_writer.New()

	// 2. Core services (Hexagonal wiring)
	discoverySvc := services.NewDiscoveryService(buildProviders()...)
	sessionRepo := repo_sqlite.NewSessionRepo(repo)
	orchestratorSvc := services.NewOrchestrator(discoverySvc, repo, writer, sessionRepo)
	orchestratorSvc.WithProviderFactory(buildProviderFromConfig)
	defer orchestratorSvc.Stop()

	// 3. Start HTTP API and MCP server with a cancellable context
	httpCtx, cancelHTTP := context.WithCancel(context.Background())
	httpAddr := os.Getenv("NEXUS_LISTEN_ADDR")
	if httpAddr == "" {
		httpAddr = "127.0.0.1:9999"
	}
	mcpAddr := os.Getenv("NEXUS_MCP_ADDR")
	if mcpAddr == "" {
		mcpAddr = "127.0.0.1:9998"
	}
	go func() {
		if err := httpapi.StartServer(httpCtx, orchestratorSvc, httpAddr); err != nil {
			log.Printf("httpapi: %v", err)
		}
	}()
	go func() {
		if err := mcp.StartMCPServer(httpCtx, orchestratorSvc, mcpAddr); err != nil {
			log.Printf("mcp: %v", err)
		}
	}()

	// 4. Initialise Wails app binding
	app := NewApp(orchestratorSvc)

	// 5. Launch Wails desktop window
	if err := wails.Run(&options.App{
		Title:  "nexusOrchestrator",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: func(_ context.Context) { cancelHTTP() },
		Bind: []interface{}{
			app,
		},
	}); err != nil {
		fmt.Fprintln(os.Stderr, "wails:", err)
		os.Exit(1)
	}
}

// buildProviders assembles all configured LLM adapters.
// Local providers are always included; cloud providers require env-var API keys.
func buildProviders() []ports.LLMClient {
	providers := []ports.LLMClient{
		llm_lmstudio.NewLMStudioAdapter("http://127.0.0.1:1234/v1"),
		llm_ollama.NewOllamaAdapter("http://127.0.0.1:11434", "codellama"),
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
	default:
		return nil, fmt.Errorf("unknown provider kind: %q", cfg.Kind)
	}
}
