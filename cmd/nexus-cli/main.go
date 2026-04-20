// Package main is the entry point for the nexus CLI binary.
// It connects to a running nexus-orchestrator daemon via the HTTP API.
package main

import (
	"fmt"
	"os"

	"nexus-orchestrator/internal/adapters/inbound/cli"
	"nexus-orchestrator/internal/adapters/outbound/httpapi_client"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

const defaultBaseURL = "http://127.0.0.1:63987"

func main() {
	baseURL := defaultBaseURL
	if addr := os.Getenv("NEXUS_ADDR"); addr != "" && baseURL == defaultBaseURL {
		baseURL = addr
	}

	// Use a lightweight HTTP-backed orchestrator stub that talks to the daemon.
	orch := httpapi_client.NewClient(baseURL)
	brain := httpapi_client.NewBrainClient(baseURL)

	root := cli.NewRootCmd(orch, brain)
	root.Version = version + " (" + commit + " " + buildDate + ")"
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
