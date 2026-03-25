// Package main is the entry point for the nexus CLI binary.
// It connects to a running nexusOrchestrator daemon via the HTTP API.
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

func main() {
	// Use a lightweight HTTP-backed orchestrator stub that talks to the daemon.
	orch := httpapi_client.NewClient("http://127.0.0.1:63987")

	root := cli.NewRootCmd(orch)
	root.Version = version + " (" + commit + " " + buildDate + ")"
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
