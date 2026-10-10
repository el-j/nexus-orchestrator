// Package main is the entry point for the nexus-daemon binary.
// It runs the full nexusOrchestrator orchestration engine without the desktop GUI,
// suitable for headless server environments or automated workflows.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/bootstrap"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	os.Exit(runMain())
}

// runMain installs signal handling and the log hub, runs the daemon and
// returns the process exit code. It is separate from main so deferred cleanup
// (signal.NotifyContext's stop) runs before os.Exit.
func runMain() int {
	// Capture log output for SSE streaming before anything logs.
	logHub := httpapi.NewLogHub()
	log.SetOutput(logHub)

	// Cancel on SIGINT / SIGTERM — drives HTTP graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, bootstrap.ConfigFromEnv(), logHub, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "daemon:", err)
		return 1
	}
	return 0
}

// run starts the shared runtime (services, REST API, MCP server, scanners) and
// blocks until ctx is cancelled, then shuts everything down gracefully. It
// returns an error only when startup cannot proceed (for example, the database
// cannot be opened); runtime server errors are logged. Output for humans (the
// ready banner and shutdown notice) is written to out.
func run(ctx context.Context, cfg bootstrap.Config, logHub *httpapi.LogHub, out io.Writer) error {
	log.Printf("nexus-daemon %s (%s %s) starting...", version, commit, buildDate)
	rt, err := bootstrap.Start(ctx, cfg, logHub)
	if err != nil {
		return err
	}
	defer rt.Close()

	// Print a human- and AI-readable ready banner once both servers are about to start.
	go func() {
		time.Sleep(50 * time.Millisecond)
		fmt.Fprint(out, formatBanner(version, rt.Config.ListenAddr, rt.Config.MCPAddr))
	}()

	<-ctx.Done()
	fmt.Fprintln(out, "nexusOrchestrator daemon shutting down.")
	return nil
}

// formatBanner builds the human- and AI-readable startup banner box.
func formatBanner(ver, addr, mcpAddr string) string {
	return bootstrap.FormatBanner(ver+" — ready", addr, mcpAddr)
}
