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
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

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
	defer orchestratorSvc.Stop()

	// 3. Start HTTP API and MCP server with a cancellable context
	httpCtx, cancelHTTP := context.WithCancel(context.Background())
	go func() {
		if err := httpapi.StartServer(httpCtx, orchestratorSvc, ":9999"); err != nil {
			log.Printf("httpapi: %v", err)
		}
	}()
	go func() {
		if err := mcp.StartMCPServer(httpCtx, orchestratorSvc, ":9998"); err != nil {
			log.Printf("mcp: %v", err)
		}
	}()

	// 4. Initialise Wails app binding
	app := NewApp(orchestratorSvc)

	// 5. Launch Wails desktop window
	if err := wails.Run(&options.App{
		Title:  "NexusAI",
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
