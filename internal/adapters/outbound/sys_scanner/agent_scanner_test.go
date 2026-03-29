package sys_scanner

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestProbeVSCodeExtensions(t *testing.T) {
	dir := t.TempDir()
	for _, d := range []string{
		"saoudrizwan.claude-dev-1.2.3",
		"continue.continue-0.9.0",
		"unknown.extension-1.0.0",
	} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0755); err != nil {
			t.Fatal(err)
		}
	}
	agents := probeVSCodeExtensionsDir(context.Background(), dir)
	if len(agents) != 2 {
		t.Errorf("expected 2 agents, got %d: %v", len(agents), agents)
	}
	names := map[string]bool{}
	for _, a := range agents {
		names[a.Name] = true
	}
	if !names["Cline"] || !names["Continue"] {
		t.Errorf("expected Cline and Continue, got %v", names)
	}
}

func TestProbeVSCodeExtensions_NonexistentDir(t *testing.T) {
	agents := probeVSCodeExtensionsDir(context.Background(), "/nonexistent/path/that/should/not/exist")
	if agents != nil {
		t.Errorf("expected nil result for missing dir, got %v", agents)
	}
}

func TestProbeMCPPorts(t *testing.T) {
	initResponse := `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","serverInfo":{"name":"test-mcp","version":"1.0"},"capabilities":{}}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, initResponse)
	}))
	defer srv.Close()
	port := srv.Listener.Addr().(*net.TCPAddr).Port
	agents := probeMCPPortList(context.Background(), []int{port})
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}
	if agents[0].Name != "test-mcp" {
		t.Errorf("expected name test-mcp, got %s", agents[0].Name)
	}
	if !agents[0].IsRunning {
		t.Error("expected IsRunning=true")
	}
	if agents[0].MCPEndpoint == "" {
		t.Error("expected non-empty MCPEndpoint")
	}
}

func TestProbeMCPPorts_NoServer(t *testing.T) {
	agents := probeMCPPortList(context.Background(), []int{1})
	if len(agents) != 0 {
		t.Errorf("expected 0 agents for unreachable port, got %d", len(agents))
	}
}

func TestProbeProcessFlagsNoError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("pgrep not available on windows")
	}
	s := &Scanner{}
	agents := s.probeProcessFlags(context.Background())
	_ = agents
}

func TestProbeClaudeSubAgents(t *testing.T) {
	// Build a fake ~/.claude/projects/ tree in a temp dir.
	home := t.TempDir()
	projectPath := "/Users/foo/myproject"
	encodedDir := base64.StdEncoding.EncodeToString([]byte(projectPath))
	sessionDir := filepath.Join(home, ".claude", "projects", encodedDir)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatal(err)
	}

	const rootAgentID = "root-agent-abc123"
	const subAgentID = "sub-agent-xyz456"

	rootJSONL := `{"agentId":"root-agent-abc123","model":"claude-sonnet-4-6","cwd":"/Users/foo/myproject","isSidechain":false}` + "\n"
	subJSONL := `{"agentId":"sub-agent-xyz456","isSidechain":true,"parentUuid":"root-agent-abc123"}` + "\n"

	if err := os.WriteFile(filepath.Join(sessionDir, "root-session.jsonl"), []byte(rootJSONL), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "sub-session.jsonl"), []byte(subJSONL), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	agents := probeClaudeSubAgentsDir(ctx, home)

	// After dedup: one agent per project directory, not one per session file.
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent (folded per project dir), got %d: %v", len(agents), agents)
	}

	a := agents[0]
	expectedID := "claude-cli-" + encodedDir
	if a.ID != expectedID {
		t.Errorf("expected ID %q, got %q", expectedID, a.ID)
	}
	if a.WorkingDir != "/Users/foo/myproject" {
		t.Errorf("expected workingDir /Users/foo/myproject, got %q", a.WorkingDir)
	}
	if a.DetectionMethod != "claude-session-file" {
		t.Errorf("expected detectionMethod claude-session-file, got %q", a.DetectionMethod)
	}
	if string(a.Kind) != "claude-cli" {
		t.Errorf("expected kind %q, got %q", "claude-cli", a.Kind)
	}
	// 2 session files → name should include count.
	expectedName := "Claude Code myproject (2 sessions)"
	if a.Name != expectedName {
		t.Errorf("expected Name %q, got %q", expectedName, a.Name)
	}
}

func TestProbeClaudeSubAgents_MissingDir(t *testing.T) {
	home := t.TempDir() // no .claude/projects inside
	agents := probeClaudeSubAgentsDir(context.Background(), home)
	if len(agents) != 0 {
		t.Errorf("expected 0 agents for missing projects dir, got %d", len(agents))
	}
}
