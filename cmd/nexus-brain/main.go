// Package main is the entry point for the standalone nexus-brain binary.
// It provides a dedicated CLI and MCP server for the Nexus Brain RAG engine,
// supporting both local standalone operation (via SQLite with FTS5) and remote
// execution against a running nexus-daemon.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"nexus-orchestrator/internal/adapters/inbound/cli"
	"nexus-orchestrator/internal/adapters/outbound/httpapi_client"
	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

// main runs the nexus-brain CLI and exits with its status.
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes the nexus-brain CLI with the given arguments and I/O streams.
func run(args []string, stdout, stderr io.Writer) int {
	var (
		addr   string
		dbPath string
	)

	// Pre-scan for global --addr and --db flags to resolve the service before cobra dispatch.
	cleanArgs := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--addr" && i+1 < len(args):
			addr = args[i+1]
			i++
		case strings.HasPrefix(arg, "--addr="):
			addr = strings.TrimPrefix(arg, "--addr=")
		case arg == "--db" && i+1 < len(args):
			dbPath = args[i+1]
			i++
		case strings.HasPrefix(arg, "--db="):
			dbPath = strings.TrimPrefix(arg, "--db=")
		default:
			cleanArgs = append(cleanArgs, arg)
		}
	}

	brain, cleanup, err := resolveBrain(addr, dbPath)
	if err != nil {
		fmt.Fprintln(stderr, "nexus-brain:", err)
		return 1
	}
	if cleanup != nil {
		defer cleanup()
	}

	root := cli.NewBrainStandaloneCmd(brain)
	root.Version = version + " (" + commit + " " + buildDate + ")"
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(cleanArgs)
	root.SilenceErrors = true
	root.SilenceUsage = true

	if err := root.Execute(); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

// resolveBrain instantiates either a remote HTTP client or a local SQLite brain service.
func resolveBrain(addr, dbPath string) (ports.BrainService, func(), error) {
	if addr == "" {
		addr = os.Getenv("NEXUS_ADDR")
	}
	if addr != "" {
		return httpapi_client.NewBrainClient(addr), nil, nil
	}

	targetDB := resolveDatabasePath(dbPath)
	if err := os.MkdirAll(filepath.Dir(targetDB), 0750); err != nil {
		return nil, nil, fmt.Errorf("create database directory: %w", err)
	}

	repo, err := repo_sqlite.New(targetDB)
	if err != nil {
		return nil, nil, fmt.Errorf("open knowledge database: %w", err)
	}

	knowledgeRepo := repo_sqlite.NewKnowledgeRepo(repo)
	brain := services.NewBrainService(knowledgeRepo, repo)
	cleanup := func() {
		_ = repo.Close()
	}
	return brain, cleanup, nil
}

// userHomeDir resolves the user home directory; tests override it for deterministic paths.
var userHomeDir = os.UserHomeDir

// resolveDatabasePath determines the SQLite file path from flag, environment, or default hierarchy.
func resolveDatabasePath(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if env := os.Getenv("NEXUS_BRAIN_DB"); env != "" {
		return env
	}
	if env := os.Getenv("NEXUS_DB_PATH"); env != "" {
		return env
	}

	localNexus := filepath.Join(".nexus", "brain.db")
	if _, err := os.Stat(localNexus); err == nil {
		return localNexus
	}

	home, err := userHomeDir()
	if err == nil {
		userGlobal := filepath.Join(home, ".nexus", "orchestrator.db")
		if _, err := os.Stat(userGlobal); err == nil {
			return userGlobal
		}
	}

	return localNexus
}
