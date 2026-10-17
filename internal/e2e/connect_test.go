//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runCLIWithHome runs a binary with HOME pointing at a scratch directory, so nothing of the real
// machine's agent configuration is read or written.
func runCLIWithHome(t *testing.T, name, addr, home, mcpURL string, args ...string) string {
	t.Helper()
	cmd := exec.Command(bin(name), args...)
	cmd.Env = append(cmd.Environ(), "NEXUS_ADDR="+addr, "HOME="+home, "USERPROFILE="+home, "NEXUS_MCP_URL="+mcpURL)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out.String())
	}
	return out.String()
}

// `nexus-cli connect` reports an agent that cannot call Nexus, writes its MCP config when asked (keeping
// other servers and a backup), and the entry it writes really reaches the running daemon.
func TestNexusConnectWritesAWorkingConfigForCursor(t *testing.T) {
	skipOnWindows(t)
	d := startDaemon(t, nil, daemonOpts{})
	d.waitHealthy()
	mcpURL := d.mcp + "/mcp"

	home := t.TempDir()
	cfg := filepath.Join(home, ".cursor", "mcp.json")
	mustWrite(t, cfg, `{"mcpServers":{"other":{"command":"keep-me"}}}`)

	report := runCLIWithHome(t, "nexus-cli", d.http, home, mcpURL, "connect", "--agent", "cursor")
	if !contains(report, "cursor") || mustRead(t, cfg) != `{"mcpServers":{"other":{"command":"keep-me"}}}` {
		t.Fatalf("without --write it only reports:\n%s", report)
	}

	runCLIWithHome(t, "nexus-cli", d.http, home, mcpURL, "connect", "--write", "--agent", "cursor")
	var written struct {
		Servers map[string]struct {
			URL     string `json:"url"`
			Command string `json:"command"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal([]byte(mustRead(t, cfg)), &written); err != nil {
		t.Fatal(err)
	}
	nexus, ok := written.Servers["nexus"]
	if !ok || written.Servers["other"].Command != "keep-me" {
		t.Fatalf("the nexus server is added and the other one stays: %+v", written)
	}
	if backups, _ := filepath.Glob(cfg + "*"); len(backups) < 2 {
		t.Errorf("the old file is kept as a backup: %v", backups)
	}
	if nexus.URL == "" && nexus.Command == "" {
		t.Fatalf("the entry has no URL or command: %+v", nexus)
	}

	// The entry points at the daemon: a client that follows it gets the tool list.
	if nexus.URL != "" {
		resp, err := http.Post(nexus.URL, "application/json", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
		if err != nil {
			t.Fatalf("following the written URL: %v", err)
		}
		defer resp.Body.Close()
		var rpc struct {
			Result struct {
				Tools []struct{ Name string } `json:"tools"`
			} `json:"result"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&rpc); err != nil || len(rpc.Result.Tools) < 40 {
			t.Errorf("the written URL must reach the daemon's MCP tools: %v %d", err, len(rpc.Result.Tools))
		}
	}

	again := runCLIWithHome(t, "nexus-cli", d.http, home, mcpURL, "connect", "--agent", "cursor")
	if !contains(again, "cursor") {
		t.Errorf("a second report mentions cursor:\n%s", again)
	}
	if _, err := os.Stat(filepath.Join(home, ".gemini")); err == nil {
		t.Error("only the asked agent is touched")
	}
}
