package main

import (
	"strings"
	"testing"
)

func TestDaemon_FormatBanner(t *testing.T) {
	banner := formatBanner("v0.10.0", "127.0.0.1:63987", "127.0.0.1:63988")
	required := []string{
		"nexusOrchestrator",
		"v0.10.0",
		"HTTP API",
		"http://127.0.0.1:63987",
		"Dashboard",
		"http://127.0.0.1:63987/ui",
		"How-to",
		"http://127.0.0.1:63987/api/howto",
		"Discovery",
		"http://127.0.0.1:63987/.well-known/nexus.json",
		"MCP",
		"http://127.0.0.1:63988/mcp",
	}

	for _, req := range required {
		if !strings.Contains(banner, req) {
			t.Errorf("expected banner to contain %q, got:\n%s", req, banner)
		}
	}
}
