// Package main is the entry point for the nexus CLI binary.
// It connects to a running nexusOrchestrator daemon via the HTTP API.
//
// The daemon address defaults to http://127.0.0.1:63987 and can be overridden
// with NEXUS_ADDR (the same variable nexus-submit uses). When the daemon
// requires authentication, set NEXUS_API_TOKEN.
package main

import (
	"fmt"
	"io"
	"os"

	"nexus-orchestrator/internal/adapters/inbound/cli"
	"nexus-orchestrator/internal/adapters/outbound/httpapi_client"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

// defaultAddr is the daemon address used when NEXUS_ADDR is unset.
const defaultAddr = "http://127.0.0.1:63987"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, daemonAddr()))
}

// daemonAddr resolves the daemon base URL from NEXUS_ADDR, falling back to defaultAddr.
func daemonAddr() string {
	if v := os.Getenv("NEXUS_ADDR"); v != "" {
		return v
	}
	return defaultAddr
}

// run executes the CLI with args against the daemon at addr and returns the
// process exit code: 0 on success, 1 when the command fails.
func run(args []string, stdout, stderr io.Writer, addr string) int {
	// Use the HTTP client adapter that talks to the running daemon.
	orch := httpapi_client.NewClient(addr)
	brain := httpapi_client.NewBrainClient(addr)

	root := cli.NewRootCmd(orch, brain)
	root.Version = version + " (" + commit + " " + buildDate + ")"
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)
	// Print each failure exactly once (cobra would also print "Error: …" and the
	// usage text for runtime failures such as an unreachable daemon).
	root.SilenceErrors = true
	root.SilenceUsage = true
	if err := root.Execute(); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
