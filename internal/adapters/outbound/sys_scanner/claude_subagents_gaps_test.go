package sys_scanner

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func skipIfModesDoNotApply(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("file modes do not stop this user")
	}
}

func TestClaudeSubAgents_WithoutAHomeDirectoryThereIsNothingToFind(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the home directory comes from USERPROFILE there")
	}
	t.Setenv("HOME", "")
	if got := probeClaudeSubAgentsDir(context.Background(), ""); got != nil {
		t.Errorf("got %v", got)
	}
}

func TestClaudeSubAgents_AnUnreadableProjectsFolderOrProjectIsSkipped(t *testing.T) {
	skipIfModesDoNotApply(t)
	t.Run("the projects folder", func(t *testing.T) {
		home := t.TempDir()
		projects := filepath.Join(home, ".claude", "projects")
		if err := os.MkdirAll(projects, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(projects, 0o000); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chmod(projects, 0o700) }()
		if got := probeClaudeSubAgentsDir(context.Background(), home); got != nil {
			t.Errorf("got %v", got)
		}
	})
	t.Run("one project folder", func(t *testing.T) {
		home := t.TempDir()
		locked := filepath.Join(home, ".claude", "projects", "-locked")
		if err := os.MkdirAll(locked, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(locked, 0o000); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chmod(locked, 0o700) }()
		writeFile(t, filepath.Join(home, ".claude", "projects", "-open", "s.jsonl"), mainSessionLines("s", "/open", "m"))
		got := byID(probeClaudeSubAgentsDir(context.Background(), home))
		if _, ok := got["claude-cli--open"]; !ok {
			t.Errorf("the readable project must still be found: %v", keys(got))
		}
	})
}

func TestClaudeSubAgents_UnreadableSessionFilesAreSkippedAndLongIDsGetAShortLabel(t *testing.T) {
	skipIfModesDoNotApply(t)
	home := t.TempDir()
	proj := filepath.Join(home, ".claude", "projects", "-work-app")
	bad := filepath.Join(proj, "bad.jsonl")
	writeFile(t, bad, mainSessionLines("bad", "/work/app", "m"))
	if err := os.Chmod(bad, 0o000); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(bad, 0o600) }()
	badSub := filepath.Join(proj, "sess-1", "subagents", "agent-locked1.jsonl")
	writeFile(t, badSub, subAgentLines("sess-1", "locked1", "x", "/work/app", ""))
	if err := os.Chmod(badSub, 0o000); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(badSub, 0o600) }()
	writeFile(t, filepath.Join(proj, "sess-1", "subagents", "agent-0123456789abcdef.jsonl"), subAgentLines("sess-1", "0123456789abcdef", "", "/work/app", ""))

	got := byID(probeClaudeSubAgentsDir(context.Background(), home))
	if _, ok := got["claude-agent-locked1"]; ok {
		t.Error("an unreadable sub-agent file must be skipped")
	}
	long, ok := got["claude-agent-0123456789abcdef"]
	if !ok || !strings.HasSuffix(long.Name, "01234567") {
		t.Errorf("a sub-agent without a slug is labelled by the first 8 characters of its id: %+v", long)
	}
}
