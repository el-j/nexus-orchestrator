// Package main is the entry point for the nexus-daemon binary.
// It runs the full NexusAI orchestration engine without the desktop GUI,
// suitable for headless server environments or automated workflows.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"nexus-ai/internal/adapters/inbound/httpapi"
	"nexus-ai/internal/adapters/inbound/mcp"
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
	sessionRepo := repo_sqlite.NewSessionRepo(repo)
	orchestratorSvc := services.NewOrchestrator(discoverySvc, repo, writer, sessionRepo)
	defer orchestratorSvc.Stop()

	// 3. Context that cancels on SIGINT / SIGTERM — drives HTTP graceful shutdown
	addr := os.Getenv("NEXUS_LISTEN_ADDR")
	if addr == "" {
		addr = ":9999"
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mcpAddr := os.Getenv("NEXUS_MCP_ADDR")
	if mcpAddr == "" {
		mcpAddr = ":9998"
	}
	go func() {
		if err := mcp.StartMCPServer(ctx, orchestratorSvc, mcpAddr); err != nil {
			log.Printf("daemon: mcp: %v", err)
		}
	}()

	// StartServer blocks until ctx is cancelled, then gracefully shuts down
	if err := httpapi.StartServer(ctx, orchestratorSvc, addr); err != nil {
		log.Printf("daemon: httpapi: %v", err)
	}

	fmt.Println("NexusAI daemon shutting down.")
}
