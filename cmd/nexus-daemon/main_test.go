package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestDaemon_ResolveConfig_Defaults(t *testing.T) {
	// Temporarily clear environment variables
	origDB := os.Getenv("NEXUS_DB_PATH")
	origAddr := os.Getenv("NEXUS_LISTEN_ADDR")
	origMCP := os.Getenv("NEXUS_MCP_ADDR")
	origScan := os.Getenv("NEXUS_SCAN_INTERVAL")
	defer func() {
		os.Setenv("NEXUS_DB_PATH", origDB)
		os.Setenv("NEXUS_LISTEN_ADDR", origAddr)
		os.Setenv("NEXUS_MCP_ADDR", origMCP)
		os.Setenv("NEXUS_SCAN_INTERVAL", origScan)
	}()

	os.Unsetenv("NEXUS_DB_PATH")
	os.Unsetenv("NEXUS_LISTEN_ADDR")
	os.Unsetenv("NEXUS_MCP_ADDR")
	os.Unsetenv("NEXUS_SCAN_INTERVAL")

	cfg := resolveConfig()
	if cfg.dbPath != "nexus.db" {
		t.Errorf("expected default dbPath nexus.db, got %s", cfg.dbPath)
	}
	if cfg.listenAddr != "127.0.0.1:63987" {
		t.Errorf("expected default listenAddr 127.0.0.1:63987, got %s", cfg.listenAddr)
	}
	if cfg.mcpAddr != "127.0.0.1:63988" {
		t.Errorf("expected default mcpAddr 127.0.0.1:63988, got %s", cfg.mcpAddr)
	}
	if cfg.scanInterval != 30*time.Second {
		t.Errorf("expected default scanInterval 30s, got %v", cfg.scanInterval)
	}
}

func TestDaemon_ResolveConfig_Custom(t *testing.T) {
	os.Setenv("NEXUS_DB_PATH", "/tmp/custom.db")
	os.Setenv("NEXUS_LISTEN_ADDR", "0.0.0.0:8080")
	os.Setenv("NEXUS_MCP_ADDR", "0.0.0.0:8081")
	os.Setenv("NEXUS_SCAN_INTERVAL", "15s")
	defer func() {
		os.Unsetenv("NEXUS_DB_PATH")
		os.Unsetenv("NEXUS_LISTEN_ADDR")
		os.Unsetenv("NEXUS_MCP_ADDR")
		os.Unsetenv("NEXUS_SCAN_INTERVAL")
	}()

	cfg := resolveConfig()
	if cfg.dbPath != "/tmp/custom.db" {
		t.Errorf("expected /tmp/custom.db, got %s", cfg.dbPath)
	}
	if cfg.listenAddr != "0.0.0.0:8080" {
		t.Errorf("expected 0.0.0.0:8080, got %s", cfg.listenAddr)
	}
	if cfg.mcpAddr != "0.0.0.0:8081" {
		t.Errorf("expected 0.0.0.0:8081, got %s", cfg.mcpAddr)
	}
	if cfg.scanInterval != 15*time.Second {
		t.Errorf("expected 15s, got %v", cfg.scanInterval)
	}
}

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
