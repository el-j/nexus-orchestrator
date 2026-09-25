package main

import (
	"bytes"
	"strings"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/cli"
	"nexus-orchestrator/internal/adapters/outbound/httpapi_client"
)

func TestNexusCLI_RootCmdConstruction(t *testing.T) {
	orch := httpapi_client.NewClient("http://127.0.0.1:63987")
	brain := httpapi_client.NewBrainClient("http://127.0.0.1:63987")

	root := cli.NewRootCmd(orch, brain)
	root.Version = "0.10.0 (test-commit 2026-09-25)"

	if root.Use != "nexus" {
		t.Errorf("expected root use 'nexus', got %q", root.Use)
	}

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("root.Execute --help: %v", err)
	}

	helpOut := buf.String()
	subcommands := []string{"tasks", "providers", "brain", "status"}
	for _, sub := range subcommands {
		if !strings.Contains(helpOut, sub) {
			t.Errorf("expected help output to mention subcommand %q, got:\n%s", sub, helpOut)
		}
	}
}

func TestNexusCLI_Version(t *testing.T) {
	orch := httpapi_client.NewClient("http://127.0.0.1:63987")
	brain := httpapi_client.NewBrainClient("http://127.0.0.1:63987")

	root := cli.NewRootCmd(orch, brain)
	root.Version = "v0.10.0 (commit-abc 2026-09-25)"

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"--version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("root.Execute --version: %v", err)
	}

	if !strings.Contains(buf.String(), "v0.10.0") {
		t.Errorf("expected version output to contain 'v0.10.0', got %q", buf.String())
	}
}
