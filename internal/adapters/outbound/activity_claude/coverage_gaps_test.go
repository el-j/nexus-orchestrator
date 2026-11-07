package activity_claude_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/outbound/activity_claude"
)

func TestHistoryReader_SkipsBlankAndMalformedLinesAndStopsWhenCancelled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.jsonl")
	good := `{"display":"hello","timestamp":1700000000000,"project":"/p","sessionId":"s1"}`
	if err := os.WriteFile(path, []byte("\n"+"{not json}\n"+good+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := activity_claude.NewClaudeHistoryReaderAt(path)
	got, err := r.ReadActivities(context.Background(), time.Time{})
	if err != nil || len(got) != 1 || got[0].SessionID != "s1" {
		t.Fatalf("activities=%+v err=%v", got, err)
	}

	if err := os.WriteFile(path, []byte(good+"\n"+good+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := activity_claude.NewClaudeHistoryReaderAt(path).ReadActivities(ctx, time.Time{}); err == nil {
		t.Error("a cancelled read must say so")
	}
}

func TestHistoryReader_ReportsAFileItCannotOpenOrRead(t *testing.T) {
	t.Run("a folder where the file should be", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "history.jsonl")
		if err := os.Mkdir(dir, 0o750); err != nil {
			t.Fatal(err)
		}
		if _, err := activity_claude.NewClaudeHistoryReaderAt(dir).ReadActivities(context.Background(), time.Time{}); err == nil {
			t.Error("a folder cannot be read as history")
		}
	})
	t.Run("a file nobody may read", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip("file modes do not stop this user")
		}
		path := filepath.Join(t.TempDir(), "history.jsonl")
		if err := os.WriteFile(path, []byte("x\n"), 0o000); err != nil {
			t.Fatal(err)
		}
		if _, err := activity_claude.NewClaudeHistoryReaderAt(path).ReadActivities(context.Background(), time.Time{}); err == nil {
			t.Error("an unreadable history must be an error, not silence")
		}
	})
}

func TestJSONLReader_SkipsALinkToAFolderNamedLikeASessionLog(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating links needs privileges on Windows")
	}
	base := t.TempDir()
	if err := os.Symlink(t.TempDir(), filepath.Join(base, "looks-like.jsonl")); err != nil {
		t.Fatal(err)
	}
	got, err := activity_claude.NewClaudeJSONLReaderAt(base).ReadActivities(context.Background(), time.Time{})
	if err != nil || len(got) != 0 {
		t.Errorf("activities=%+v err=%v", got, err)
	}
}
