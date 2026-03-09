// Package main is the entry point for the nexus-daemon binary.
// It runs the full NexusAI orchestration engine without the desktop GUI,
// suitable for headless server environments or automated workflows.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"nexus-ai/internal/adapters/inbound/httpapi"
	"nexus-ai/internal/adapters/outbound/fs_writer"
	"nexus-ai/internal/adapters/outbound/llm_lmstudio"
	"nexus-ai/internal/adapters/outbound/llm_ollama"
	"nexus-ai/internal/adapters/outbound/repo_sqlite"
	"nexus-ai/internal/core/services"
)

func main() {
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

	lmStudio := llm_lmstudio.NewLMStudioAdapter("http://127.0.0.1:1234/v1")
	ollama := llm_ollama.NewOllamaAdapter("http://127.0.0.1:11434", "codellama")
	writer := fs_writer.New()

	// 2. Core services
	discoverySvc := services.NewDiscoveryService(lmStudio, ollama)
	orchestratorSvc := services.NewOrchestrator(discoverySvc, repo, writer)
	defer orchestratorSvc.Stop()

	// 3. Start HTTP API in background
	addr := os.Getenv("NEXUS_LISTEN_ADDR")
	if addr == "" {
		addr = ":9999"
	}
	go httpapi.StartServer(orchestratorSvc, addr)

	// 4. Block until SIGINT / SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("NexusAI daemon shutting down.")
}
