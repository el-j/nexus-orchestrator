package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"nexus-ai/internal/adapters/inbound/httpapi"
	"nexus-ai/internal/adapters/outbound/fs_writer"
	"nexus-ai/internal/adapters/outbound/llm_lmstudio"
	"nexus-ai/internal/adapters/outbound/llm_ollama"
	"nexus-ai/internal/adapters/outbound/repo_sqlite"
	"nexus-ai/internal/core/services"
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

	lmStudio := llm_lmstudio.NewLMStudioAdapter("http://127.0.0.1:1234/v1")
	ollama := llm_ollama.NewOllamaAdapter("http://127.0.0.1:11434", "codellama")
	writer := fs_writer.New()

	// 2. Core services (Hexagonal wiring)
	discoverySvc := services.NewDiscoveryService(lmStudio, ollama)
	orchestratorSvc := services.NewOrchestrator(discoverySvc, repo, writer)
	defer orchestratorSvc.Stop()

	// 3. Start VS Code HTTP API in the background
	go httpapi.StartServer(orchestratorSvc, ":9999")

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
		OnStartup: app.startup,
		Bind: []interface{}{
			app,
		},
	}); err != nil {
		fmt.Fprintln(os.Stderr, "wails:", err)
		os.Exit(1)
	}
}
