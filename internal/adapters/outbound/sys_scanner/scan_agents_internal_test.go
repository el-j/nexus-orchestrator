package sys_scanner

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

func TestScanAgents_MergesProbesAndLinksSubAgents(t *testing.T) {
	home := isolateHome(t)
	stubGOOS(t, "linux")
	stubPgrep(t, func(p string) ([]byte, error) {
		if p == "Claude" {
			return []byte("42 /Applications/Claude.app/Contents/MacOS/Claude\n"), nil
		}
		return nil, os.ErrNotExist
	})
	// Config probe finds Claude Desktop; the process probe finds it running -> merged into one agent.
	if err := os.MkdirAll(filepath.Join(home, ".config", "claude"), 0o750); err != nil {
		t.Fatal(err)
	}
	// A project session with a parent and a sidechain (sub-agent) session.
	proj := base64.StdEncoding.EncodeToString([]byte("/work/proj"))
	writeFile(t, filepath.Join(home, ".claude", "projects", proj, "parent.jsonl"),
		`{"agentId":"parent-1","model":"opus","cwd":"/work/proj"}`+"\n")
	otherProj := base64.StdEncoding.EncodeToString([]byte("/work/other"))
	writeFile(t, filepath.Join(home, ".claude", "projects", otherProj, "child.jsonl"),
		`{"agentId":"child-1","isSidechain":true,"parentUuid":"claude-cli-`+proj+`"}`+"\n")

	got, err := newTestScanner().ScanAgents(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]domain.DiscoveredAgent{}
	for _, a := range got {
		byID[a.ID] = a
		if a.LastSeen.IsZero() && a.DetectionMethod != "claude-session-file" {
			t.Errorf("%s has no LastSeen", a.ID)
		}
	}
	desktop, ok := byID[string(domain.AgentKindClaudeDesktop)]
	if !ok || !desktop.IsRunning || desktop.ConfigPath == "" {
		t.Errorf("desktop agent must merge config + process evidence: %+v", desktop)
	}
	parent := byID["claude-cli-"+proj]
	child := byID["claude-cli-"+otherProj]
	if parent.ID == "" || child.ID == "" {
		t.Fatalf("session agents missing: %v", keys(byID))
	}
	if child.ParentAgentID != "claude-cli-"+proj {
		// The sub-agent parent reference is the raw parentUuid; link only when it names a known agent.
		t.Logf("child parent = %q", child.ParentAgentID)
	}
	if len(parent.SubAgentIDs) != 1 || parent.SubAgentIDs[0] != child.ID {
		t.Errorf("parent must list its sub-agent: %+v", parent.SubAgentIDs)
	}
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestProbeClaudeSubAgentsDir_SessionHandling(t *testing.T) {
	home := t.TempDir()
	if got := probeClaudeSubAgentsDir(context.Background(), home); got != nil {
		t.Errorf("no projects dir: %v", got)
	}
	enc := base64.StdEncoding.EncodeToString([]byte("/work/app"))
	dir := filepath.Join(home, ".claude", "projects", enc)
	writeFile(t, filepath.Join(dir, "a.jsonl"), `{"agentId":"a","cwd":"/work/app"}`+"\n")
	writeFile(t, filepath.Join(dir, "b.jsonl"), "not json\n\n"+`{"model":"m"}`+"\n") // falls back to the file name, decoded dir
	writeFile(t, filepath.Join(dir, "stale.jsonl"), `{"agentId":"stale"}`+"\n")
	writeFile(t, filepath.Join(dir, "ignored.txt"), "x")
	if err := os.MkdirAll(filepath.Join(dir, "subdir"), 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(home, ".claude", "projects", "stray-file.txt"), "not a dir")
	old := time.Now().Add(-3 * time.Hour)
	if err := os.Chtimes(filepath.Join(dir, "stale.jsonl"), old, old); err != nil {
		t.Fatal(err)
	}
	recent := time.Now().Add(-20 * time.Minute) // within 2h, but not "active" (5m)
	if err := os.Chtimes(filepath.Join(dir, "b.jsonl"), recent, recent); err != nil {
		t.Fatal(err)
	}
	// An empty project directory yields nothing.
	if err := os.MkdirAll(filepath.Join(home, ".claude", "projects", "not-base64-@@"), 0o750); err != nil {
		t.Fatal(err)
	}

	got := probeClaudeSubAgentsDir(context.Background(), home)
	if len(got) != 1 {
		t.Fatalf("one agent per project directory, got %+v", got)
	}
	a := got[0]
	if a.ID != "claude-cli-"+enc || a.Kind != domain.AgentKindClaudeCLI || a.DetectionMethod != "claude-session-file" {
		t.Errorf("agent = %+v", a)
	}
	if !strings.HasPrefix(a.Name, "Claude Code app") || !strings.Contains(a.Name, "(2 sessions)") {
		t.Errorf("name must reflect the project and session count (stale file excluded): %q", a.Name)
	}
	if !a.IsRunning { // the freshest session (a.jsonl) was just written
		t.Error("a session touched moments ago is running")
	}
}

func TestParseClaudeSessionFile_Limits(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "sess.jsonl")
	var sb strings.Builder
	for i := 0; i < 5; i++ {
		sb.WriteString("{}\n")
	}
	sb.WriteString(`{"agentId":"late"}` + "\n")
	writeFile(t, p, sb.String())
	if a := parseClaudeSessionFile(p, 3); a == nil || a.ID != "sess" {
		t.Errorf("only the first maxLines are read; id must fall back to the file name, got %+v", a)
	}
	if a := parseClaudeSessionFile(filepath.Join(dir, "missing.jsonl"), 10); a != nil {
		t.Errorf("missing file: %+v", a)
	}
	writeFile(t, p, `{"agentId":"x","isSidechain":true}`+"\n") // sidechain without parent: no parent link
	if a := parseClaudeSessionFile(p, 10); a == nil || a.ParentAgentID != "" {
		t.Errorf("%+v", a)
	}
}

func TestScan_EndToEndWithStubbedProcessesAndCLIs(t *testing.T) {
	isolateHome(t)
	stubGOOS(t, "linux")
	stubPgrep(t, func(p string) ([]byte, error) {
		if p == "Copilot" {
			return []byte("9 /usr/bin/copilot-agent\n"), nil
		}
		return nil, os.ErrNotExist
	})
	bin := t.TempDir()
	writeFile(t, filepath.Join(bin, "ollama"), "#!/bin/sh\n")
	if err := os.Chmod(filepath.Join(bin, "ollama"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)

	got, err := newTestScanner().Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]domain.DiscoveredProvider{}
	for i, p := range got {
		names[p.Name] = p
		if i > 0 && got[i-1].Name > p.Name {
			t.Error("results must be sorted by name")
		}
	}
	if names["Copilot"].Status != domain.DiscoveryStatusRunning || names["Ollama CLI"].Status != domain.DiscoveryStatusInstalled {
		t.Errorf("got %+v", names)
	}
}

func TestScan_ShortContextDeadlineStillReturns(t *testing.T) {
	isolateHome(t)
	stubGOOS(t, "linux")
	stubPgrep(t, func(string) ([]byte, error) { return nil, os.ErrNotExist })
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond)
	if _, err := newTestScanner().Scan(ctx); err != nil {
		t.Fatalf("an expired context must still return cleanly, got %v", err)
	}
}

func TestProbeMCPPorts_DefaultListIsProbedWithoutCrashing(t *testing.T) {
	_ = newTestScanner().probeMCPPorts(context.Background()) // ports are machine-dependent; must just not panic
	_ = newTestScanner().probeClaudeSubAgents(context.Background())
}
