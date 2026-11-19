// Package main is the entry point for the nexus-runner binary.
// It executes standalone verification gates, invariant policies, and self-healing task execution loops.
package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"nexus-orchestrator/internal/adapters/inbound/cli"
	"nexus-orchestrator/internal/bootstrap"
	"nexus-orchestrator/pkg/runner"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"

	generatorBuilder = buildGenerator
	buildProvidersFn = bootstrap.BuildProviders
)

// main runs the nexus-runner CLI and exits with its status.
func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

// buildGenerator discovers available providers and wraps the first ready client as a generator.
func buildGenerator() runner.Generator {
	providers := buildProvidersFn()
	if len(providers) > 0 {
		return cli.NewProviderGenerator(providers[0])
	}
	return nil
}

// run parses arguments, constructs the standalone runner command, and executes it.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	gen := generatorBuilder()
	root := cli.NewRunnerStandaloneCmd(gen, ".")
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
