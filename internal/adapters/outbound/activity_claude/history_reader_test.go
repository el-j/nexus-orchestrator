package activity_claude_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/outbound/activity_claude"
	"nexus-orchestrator/internal/core/domain"
)

// historyLine mirrors the JSON structure written to history.jsonl.
type historyLine struct {
	Display   string `json:"display"`
	Timestamp int64  `json:"timestamp"`
	Project   string `json:"project"`
	SessionID string `json:"sessionId"`
}

func writeHistoryFile(t *testing.T, lines []historyLine) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "history.jsonl")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create history file: %v", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, l := range lines {
		if err := enc.Encode(l); err != nil {
			t.Fatalf("encode history line: %v", err)
		}
	}
	return path
}

func newReaderAt(path string) *activity_claude.ClaudeHistoryReader {
	return activity_claude.NewClaudeHistoryReaderAt(path)
}

func TestReadActivities_Basic(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	past := now.Add(-10 * time.Minute)
	older := now.Add(-20 * time.Minute)

	lines := []historyLine{
		{Display: "claude ask something", Timestamp: older.UnixMilli(), Project: "/proj/a", SessionID: "sess-1"},
		{Display: "claude fix bug", Timestamp: past.UnixMilli(), Project: "/proj/b", SessionID: "sess-2"},
		{Display: "claude review", Timestamp: now.UnixMilli(), Project: "/proj/c", SessionID: "sess-3"},
	}
	path := writeHistoryFile(t, lines)
	r := newReaderAt(path)

	// since = older, so all three should be included
	activities, err := r.ReadActivities(context.Background(), older)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(activities) != 3 {
		t.Fatalf("expected 3 activities, got %d", len(activities))
	}
}

func TestReadActivities_SinceFilter(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	past := now.Add(-10 * time.Minute)
	older := now.Add(-20 * time.Minute)

	lines := []historyLine{
		{Display: "old command", Timestamp: older.UnixMilli(), Project: "/proj/a", SessionID: "sess-old"},
		{Display: "recent command", Timestamp: now.UnixMilli(), Project: "/proj/b", SessionID: "sess-new"},
	}
	path := writeHistoryFile(t, lines)
	r := newReaderAt(path)

	// since = past → only the entry at `now` should be included
	activities, err := r.ReadActivities(context.Background(), past)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(activities))
	}
	got := activities[0]
	if got.AgentName != "claude" {
		t.Errorf("AgentName: got %q, want %q", got.AgentName, "claude")
	}
	if got.ActivityType != domain.ActivityTypeMessage {
		t.Errorf("ActivityType: got %q, want %q", got.ActivityType, domain.ActivityTypeMessage)
	}
	if got.Summary != "recent command" {
		t.Errorf("Summary: got %q, want %q", got.Summary, "recent command")
	}
	if got.ProjectPath != "/proj/b" {
		t.Errorf("ProjectPath: got %q, want %q", got.ProjectPath, "/proj/b")
	}
	if got.SessionID != "sess-new" {
		t.Errorf("SessionID: got %q, want %q", got.SessionID, "sess-new")
	}
	if got.Metadata["source"] != "cli-history" {
		t.Errorf("Metadata source: got %q, want %q", got.Metadata["source"], "cli-history")
	}
}

func TestReadActivities_FileNotFound(t *testing.T) {
	r := newReaderAt("/nonexistent/path/history.jsonl")
	activities, err := r.ReadActivities(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("expected nil error for missing file, got: %v", err)
	}
	if activities != nil {
		t.Errorf("expected nil activities for missing file, got: %v", activities)
	}
}

func TestReadActivities_TruncateSummary(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	long := "abcdefghij" // 10 chars repeated 9x = 90, > 80
	for len(long) < 90 {
		long += "x"
	}

	lines := []historyLine{
		{Display: long, Timestamp: now.UnixMilli(), Project: "/proj", SessionID: "sess-trunc"},
	}
	path := writeHistoryFile(t, lines)
	r := newReaderAt(path)

	activities, err := r.ReadActivities(context.Background(), now.Add(-time.Second))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(activities))
	}
	if len([]rune(activities[0].Summary)) != 80 {
		t.Errorf("Summary length: got %d, want 80", len([]rune(activities[0].Summary)))
	}
}

func TestReadActivities_SourceName(t *testing.T) {
	r := activity_claude.NewClaudeHistoryReader()
	if r.SourceName() != "claude-history" {
		t.Errorf("SourceName: got %q, want %q", r.SourceName(), "claude-history")
	}
}

func TestReadActivities_IncrementalRead(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)

	dir := t.TempDir()
	path := filepath.Join(dir, "history.jsonl")

	// Write first entry
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	enc := json.NewEncoder(f)
	_ = enc.Encode(historyLine{Display: "first", Timestamp: now.UnixMilli(), Project: "/p", SessionID: "s1"})
	f.Close()

	r := newReaderAt(path)
	since := now.Add(-time.Second)

	acts, err := r.ReadActivities(context.Background(), since)
	if err != nil {
		t.Fatalf("first read: %v", err)
	}
	if len(acts) != 1 || acts[0].Summary != "first" {
		t.Fatalf("first read: expected [first], got %v", acts)
	}

	// Append a second entry
	f, err = os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	enc = json.NewEncoder(f)
	_ = enc.Encode(historyLine{Display: "second", Timestamp: now.Add(time.Second).UnixMilli(), Project: "/p", SessionID: "s2"})
	f.Close()

	acts2, err := r.ReadActivities(context.Background(), since)
	if err != nil {
		t.Fatalf("second read: %v", err)
	}
	if len(acts2) != 1 || acts2[0].Summary != "second" {
		t.Fatalf("second read: expected [second], got %v", acts2)
	}
}
