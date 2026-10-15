package sys_scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// The fixtures below mirror the shape of real Claude Code session files (field
// names and nesting taken from files on a developer machine; identifiers and
// content are invented): a project directory named after the path with dashes,
// main sessions as <sessionId>.jsonl, and sub-agents in
// <sessionId>/subagents/agent-<agentId>.jsonl whose first message has a null
// parentUuid.

func mainSessionLines(sessionID, cwd, model string) string {
	return strings.Join([]string{
		fmt.Sprintf(`{"type":"mode","mode":"normal","sessionId":%q}`, sessionID),
		fmt.Sprintf(`{"parentUuid":null,"isSidechain":false,"userType":"external","cwd":%q,"sessionId":%q,"type":"user","uuid":"u-1","message":{"role":"user","content":"hello"}}`, cwd, sessionID),
		fmt.Sprintf(`{"parentUuid":"u-1","isSidechain":false,"cwd":%q,"sessionId":%q,"type":"assistant","uuid":"a-1","message":{"role":"assistant","model":%q,"content":[]}}`, cwd, sessionID, model),
	}, "\n") + "\n"
}

func subAgentLines(parentSession, agentID, slug, cwd, model string) string {
	return strings.Join([]string{
		fmt.Sprintf(`{"parentUuid":null,"isSidechain":true,"agentId":%q,"sessionId":%q,"slug":%q,"cwd":%q,"type":"user","uuid":"s-1","message":{"role":"user","content":"do a thing"}}`, agentID, parentSession, slug, cwd),
		// later messages point at messages INSIDE the sidechain, never at the parent agent
		fmt.Sprintf(`{"parentUuid":"s-1","isSidechain":true,"agentId":%q,"sessionId":%q,"slug":%q,"cwd":%q,"type":"assistant","uuid":"s-2","message":{"role":"assistant","model":%q,"content":[]}}`, agentID, parentSession, slug, cwd, model),
	}, "\n") + "\n"
}

func touch(t *testing.T, path string, age time.Duration) {
	t.Helper()
	at := time.Now().Add(-age)
	if err := os.Chtimes(path, at, at); err != nil {
		t.Fatal(err)
	}
}

func byID(agents []domain.DiscoveredAgent) map[string]domain.DiscoveredAgent {
	m := map[string]domain.DiscoveredAgent{}
	for _, a := range agents {
		m[a.ID] = a
	}
	return m
}

func TestClaudeSubAgents_DiscoversSubagentFilesAndLinksThemToTheirProject(t *testing.T) {
	home := t.TempDir()
	proj := filepath.Join(home, ".claude", "projects", "-work-app")
	writeFile(t, filepath.Join(proj, "sess-1.jsonl"), mainSessionLines("sess-1", "/work/app", "claude-opus"))
	writeFile(t, filepath.Join(proj, "sess-1", "subagents", "agent-aaa111.jsonl"), subAgentLines("sess-1", "aaa111", "witty-fox", "/work/app", "claude-haiku"))
	writeFile(t, filepath.Join(proj, "sess-1", "subagents", "agent-bbb222.jsonl"), subAgentLines("sess-1", "bbb222", "", "/work/app", ""))

	got := byID(probeClaudeSubAgentsDir(context.Background(), home))

	parent, ok := got["claude-cli--work-app"]
	if !ok {
		t.Fatalf("project agent missing: %v", keys(got))
	}
	if parent.ModelID != "claude-opus" {
		t.Errorf("the model lives under message.model in real files, got %q", parent.ModelID)
	}
	a, ok := got["claude-agent-aaa111"]
	if !ok {
		t.Fatalf("sub-agent files were not discovered: %v", keys(got))
	}
	if a.ParentAgentID != parent.ID {
		t.Errorf("parent = %q, want %q", a.ParentAgentID, parent.ID)
	}
	if a.Name != "Sub-agent witty-fox" || a.ModelID != "claude-haiku" || a.WorkingDir != "/work/app" {
		t.Errorf("sub-agent details: %+v", a)
	}
	if a.DetectionMethod != "claude-subagent-file" || a.Kind != domain.AgentKindClaudeCLI || !a.IsRunning || a.LastSeen.IsZero() {
		t.Errorf("sub-agent metadata: %+v", a)
	}
	if b := got["claude-agent-bbb222"]; b.Name != "Sub-agent bbb222" {
		t.Errorf("without a slug the name falls back to the agent id: %q", b.Name)
	}
}

func TestClaudeSubAgents_ScanLinksTheTree(t *testing.T) {
	home := isolateHome(t)
	stubGOOS(t, "linux")
	stubPgrep(t, func(string) ([]byte, error) { return nil, os.ErrNotExist })
	proj := filepath.Join(home, ".claude", "projects", "-work-app")
	writeFile(t, filepath.Join(proj, "sess-1.jsonl"), mainSessionLines("sess-1", "/work/app", "claude-opus"))
	writeFile(t, filepath.Join(proj, "sess-1", "subagents", "agent-aaa111.jsonl"), subAgentLines("sess-1", "aaa111", "x", "/work/app", ""))

	agents, err := newTestScanner().ScanAgents(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	m := byID(agents)
	parent := m["claude-cli--work-app"]
	if len(parent.SubAgentIDs) != 1 || parent.SubAgentIDs[0] != "claude-agent-aaa111" {
		t.Fatalf("the sub-agent tree must not be flat: %+v", parent.SubAgentIDs)
	}
}

func TestClaudeSubAgents_ParentUuidIsNotTreatedAsAnAgentID(t *testing.T) {
	// Regression: probes used to set ParentAgentID = parentUuid, a message id that can never match an agent.
	home := t.TempDir()
	proj := filepath.Join(home, ".claude", "projects", "-p")
	writeFile(t, filepath.Join(proj, "s.jsonl"), mainSessionLines("s", "/p", "m"))
	writeFile(t, filepath.Join(proj, "s", "subagents", "agent-x.jsonl"), subAgentLines("s", "x", "", "/p", ""))
	for _, a := range probeClaudeSubAgentsDir(context.Background(), home) {
		if a.ParentAgentID != "" && a.ParentAgentID != "claude-cli--p" {
			t.Errorf("%s has parent %q", a.ID, a.ParentAgentID)
		}
	}
}

func TestClaudeSubAgents_SkipsStaleFilesAndIgnoresOtherFiles(t *testing.T) {
	home := t.TempDir()
	proj := filepath.Join(home, ".claude", "projects", "-work-app")
	sub := filepath.Join(proj, "sess-1", "subagents")
	writeFile(t, filepath.Join(proj, "sess-1.jsonl"), mainSessionLines("sess-1", "/work/app", "m"))
	writeFile(t, filepath.Join(sub, "agent-fresh.jsonl"), subAgentLines("sess-1", "fresh", "", "/work/app", ""))
	writeFile(t, filepath.Join(sub, "agent-old.jsonl"), subAgentLines("sess-1", "old", "", "/work/app", ""))
	writeFile(t, filepath.Join(sub, "notes.txt"), "not an agent")
	writeFile(t, filepath.Join(sub, "other.jsonl"), subAgentLines("sess-1", "other", "", "/work/app", ""))
	if err := os.MkdirAll(filepath.Join(sub, "agent-dir.jsonl"), 0o750); err != nil {
		t.Fatal(err)
	}
	touch(t, filepath.Join(sub, "agent-old.jsonl"), 3*time.Hour)

	got := byID(probeClaudeSubAgentsDir(context.Background(), home))
	if _, ok := got["claude-agent-fresh"]; !ok {
		t.Error("a fresh sub-agent must be reported")
	}
	for _, id := range []string{"claude-agent-old", "claude-agent-other", "claude-agent-dir"} {
		if _, ok := got[id]; ok {
			t.Errorf("%s must not be reported", id)
		}
	}
}

func TestClaudeSubAgents_ARecentlyWrittenSubagentIsRunningAnOldOneIsNot(t *testing.T) {
	home := t.TempDir()
	proj := filepath.Join(home, ".claude", "projects", "-work-app")
	sub := filepath.Join(proj, "sess-1", "subagents")
	writeFile(t, filepath.Join(sub, "agent-live.jsonl"), subAgentLines("sess-1", "live", "", "/w", ""))
	writeFile(t, filepath.Join(sub, "agent-idle.jsonl"), subAgentLines("sess-1", "idle", "", "/w", ""))
	touch(t, filepath.Join(sub, "agent-idle.jsonl"), 30*time.Minute)
	got := byID(probeClaudeSubAgentsDir(context.Background(), home))
	if !got["claude-agent-live"].IsRunning || got["claude-agent-idle"].IsRunning {
		t.Errorf("running flags: live=%v idle=%v", got["claude-agent-live"].IsRunning, got["claude-agent-idle"].IsRunning)
	}
}

func TestClaudeSubAgents_AreCappedPerProjectNewestFirst(t *testing.T) {
	home := t.TempDir()
	sub := filepath.Join(home, ".claude", "projects", "-work-app", "sess-1", "subagents")
	for i := 0; i < claudeMaxSubAgents+10; i++ {
		id := fmt.Sprintf("a%03d", i)
		p := filepath.Join(sub, "agent-"+id+".jsonl")
		writeFile(t, p, subAgentLines("sess-1", id, "", "/w", ""))
		touch(t, p, time.Duration(i+1)*time.Second) // a000 is the newest
	}
	agents := probeClaudeSubAgentsDir(context.Background(), home)
	if len(agents) != claudeMaxSubAgents {
		t.Fatalf("got %d sub-agents, want the cap of %d", len(agents), claudeMaxSubAgents)
	}
	if byID(agents)["claude-agent-a000"].ID == "" || byID(agents)["claude-agent-a034"].ID != "" {
		t.Error("the newest sub-agents must be kept and the oldest dropped")
	}
}

func TestClaudeSubAgents_AProjectWithOnlySubagentsStillReportsThem(t *testing.T) {
	// The main session is stale, so there is no project agent; the sub-agent is reported and
	// its parent reference simply has nothing to attach to (UIs show it as a root).
	home := t.TempDir()
	proj := filepath.Join(home, ".claude", "projects", "-work-app")
	writeFile(t, filepath.Join(proj, "sess-1.jsonl"), mainSessionLines("sess-1", "/w", "m"))
	touch(t, filepath.Join(proj, "sess-1.jsonl"), 5*time.Hour)
	writeFile(t, filepath.Join(proj, "sess-1", "subagents", "agent-x.jsonl"), subAgentLines("sess-1", "x", "", "/w", ""))
	got := byID(probeClaudeSubAgentsDir(context.Background(), home))
	if len(got) != 1 || got["claude-agent-x"].ID == "" {
		t.Fatalf("got %v", keys(got))
	}
}

func TestClaudeSubAgents_UnreadableOrEmptyFilesAreSkipped(t *testing.T) {
	home := t.TempDir()
	sub := filepath.Join(home, ".claude", "projects", "-work-app", "sess-1", "subagents")
	writeFile(t, filepath.Join(sub, "agent-empty.jsonl"), "")
	writeFile(t, filepath.Join(sub, "agent-garbage.jsonl"), "not json\n{also bad\n")
	got := byID(probeClaudeSubAgentsDir(context.Background(), home))
	// Both still identify themselves by file name; neither crashes the probe.
	for _, id := range []string{"claude-agent-empty", "claude-agent-garbage"} {
		if a, ok := got[id]; !ok || a.ParentAgentID != "claude-cli--work-app" {
			t.Errorf("%s: %+v", id, a)
		}
	}
}

func TestReadClaudeJSONL_UsesTheFirstValuesAndBothModelLocations(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.jsonl")
	writeFile(t, p, `{"cwd":"/first","agentId":"one","slug":"s1","model":"top-level"}`+"\n"+
		`{"cwd":"/second","agentId":"two","slug":"s2","message":{"model":"nested"}}`+"\n\n")
	m := readClaudeJSONL(p, 10)
	if m == nil || m.CWD != "/first" || m.AgentID != "one" || m.Slug != "s1" || m.Model != "top-level" {
		t.Fatalf("%+v", m)
	}
	writeFile(t, p, `{"message":{"role":"user"}}`+"\n"+`{"message":{"model":"nested"}}`+"\n")
	if m := readClaudeJSONL(p, 10); m == nil || m.Model != "nested" {
		t.Fatalf("nested model: %+v", m)
	}
	if readClaudeJSONL(filepath.Join(dir, "missing.jsonl"), 10) != nil {
		t.Error("a missing file must return nil")
	}
}

func TestReadClaudeJSONL_HandlesVeryLongLines(t *testing.T) {
	// Tool results make real session lines megabytes long; the head must still be readable.
	dir := t.TempDir()
	p := filepath.Join(dir, "big.jsonl")
	writeFile(t, p, `{"cwd":"/w","padding":"`+strings.Repeat("x", 3<<20)+`"}`+"\n"+`{"agentId":"late"}`+"\n")
	m := readClaudeJSONL(p, 10)
	if m == nil || m.CWD != "/w" || m.AgentID != "late" {
		t.Fatalf("a 3 MiB line broke parsing: %+v", m)
	}
}
