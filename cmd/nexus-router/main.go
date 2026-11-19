// Package main is the entry point for the nexus-router binary.
// It runs the standalone LLM Gateway and OpenAI-compatible proxy server.
//
// In standalone mode, nexus-router discovers providers via environment variables
// and local endpoints, routing completions through Nexus's model ladder, cooldown
// engine, and rate-limit recovery. If NEXUS_ADDR is set, it can query a running daemon.
package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"nexus-orchestrator/internal/adapters/inbound/cli"
	"nexus-orchestrator/internal/adapters/inbound/gateway"
	"nexus-orchestrator/internal/adapters/outbound/httpapi_client"
	"nexus-orchestrator/internal/bootstrap"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

// defaultAddr is the default daemon address.
const defaultAddr = "http://127.0.0.1:63987"

// main runs the nexus-router CLI and exits with its status.
func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

// daemonAddr returns NEXUS_ADDR if explicitly configured.
func daemonAddr() string {
	return os.Getenv("NEXUS_ADDR")
}

// run parses arguments, constructs the gateway, and executes the CLI command.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	var (
		orch      ports.Orchestrator
		discovery *services.DiscoveryService
		gw        *gateway.Gateway
	)

	// If NEXUS_ADDR is set, connect client to remote daemon.
	addr := daemonAddr()
	if addr != "" {
		orch = httpapi_client.NewClient(addr)
	}

	// Always instantiate local discovery and gateway engine for standalone operations (serve, prompt).
	providers := bootstrap.BuildProviders()
	discovery = services.NewDiscoveryService(providers...)
	cfg := gateway.Config{
		DefaultRole: "coder",
		ProjectPath: ".",
	}
	gw = gateway.NewGateway(cfg, discovery)

	root := cli.NewRouterStandaloneCmd(gw, discovery, orch)
	root.Version = version + " (" + commit + " " + buildDate + ")"
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)
	root.SilenceErrors = true
	root.SilenceUsage = true

	if err := root.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
