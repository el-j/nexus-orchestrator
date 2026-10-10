package activity_claude_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/outbound/activity_claude"
	"nexus-orchestrator/internal/core/domain"
)

func jsonl(lines ...string) string {
	out := ""
	for _, l := range lines {
		out += l + "\n"
	}
	return out
}

func appendTo(t *testing.T, path, text string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(text); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, r *activity_claude.ClaudeJSONLReader) []domain.AIActivity {
	t.Helper()
	got, err := r.ReadActivities(context.Background(), time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return got
}

var now = time.Now().UTC().Format(time.RFC3339)

func TestReader_IsIncrementalAndSurvivesPartialLines(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "proj", "s.jsonl")
	if err := os.MkdirAll(filepath.Dir(file), 0o750); err != nil {
		t.Fatal(err)
	}
	r := activity_claude.NewClaudeJSONLReaderAt(dir)

	appendTo(t, file, jsonl(`{"type":"user","uuid":"u1","sessionId":"s","timestamp":"`+now+`"}`))
	if got := read(t, r); len(got) != 1 || got[0].Summary != "User prompt" {
		t.Fatalf("first read: %+v", got)
	}
	if got := read(t, r); len(got) != 0 {
		t.Errorf("nothing new, nothing returned: %+v", got)
	}

	// A line still being written (no newline yet) is held back, then delivered whole.
	appendTo(t, file, `{"type":"assistant","uuid":"a1","sessionId":"s","timestamp":"`+now+`","message":{"model":"opus","usage":{"input_tokens":3,"output_tokens":9}}}`)
	if got := read(t, r); len(got) != 0 {
		t.Fatalf("an incomplete line must not be consumed: %+v", got)
	}
	appendTo(t, file, "\n")
	got := read(t, r)
	if len(got) != 1 || got[0].Model != "opus" || got[0].TokensOut != 9 || got[0].Summary != "Responding (9 tokens)" {
		t.Fatalf("completed line: %+v", got)
	}
}

// A truncated or rotated log used to wedge the reader until the file outgrew
// its old offset.
func TestReader_RecoversWhenTheFileIsTruncated(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "s.jsonl")
	r := activity_claude.NewClaudeJSONLReaderAt(dir)

	long := jsonl(
		`{"type":"user","uuid":"u1","sessionId":"s","timestamp":"`+now+`"}`,
		`{"type":"user","uuid":"u2","sessionId":"s","timestamp":"`+now+`"}`,
		`{"type":"user","uuid":"u3","sessionId":"s","timestamp":"`+now+`"}`,
	)
	appendTo(t, file, long)
	if got := read(t, r); len(got) != 3 {
		t.Fatalf("initial: %d", len(got))
	}
	// Rotated: replaced by a much shorter file with a new record.
	if err := os.WriteFile(file, []byte(jsonl(`{"type":"tool_use","uuid":"t1","name":"Bash","timestamp":"`+now+`"}`)), 0o600); err != nil {
		t.Fatal(err)
	}
	got := read(t, r)
	if len(got) != 1 || got[0].Summary != "Using Bash" {
		t.Fatalf("after truncation the reader must restart from the beginning: %+v", got)
	}
}

func TestReader_ParsesAllRecordTypesAndSkipsJunk(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "s.jsonl")
	appendTo(t, file, jsonl(
		`not json`,
		``,
		`{"type":"summary","uuid":"x","timestamp":"`+now+`"}`, // unrecognised type
		`{"type":"assistant","uuid":"a","timestamp":"`+now+`"}`,
		`{"type":"tool_use","uuid":"t","timestamp":"`+now+`"}`,                   // no tool name
		`{"type":"user","cwd":"/work/p","sessionId":"s9","timestamp":"`+now+`"}`, // no uuid: hashed id
		`{"type":"user","uuid":"only-uuid","timestamp":"`+now+`"}`,
		`{"type":"user","uuid":"old","timestamp":"2001-01-01T00:00:00Z"}`, // before the cutoff
	))
	got := read(t, activity_claude.NewClaudeJSONLReaderAt(dir))
	byID := map[string]domain.AIActivity{}
	for _, a := range got {
		byID[a.ID] = a
	}
	if len(got) != 4 {
		t.Fatalf("want 4 activities, got %d: %+v", len(got), got)
	}
	if a := byID["claude-a"]; a.Summary != "Responding" || a.ActivityType != domain.ActivityTypeGeneration || a.AgentName != "claude" {
		t.Errorf("assistant without usage: %+v", a)
	}
	if a := byID["claude-t"]; a.Summary != "Using unknown" || a.ActivityType != domain.ActivityTypeToolUse {
		t.Errorf("tool without name: %+v", a)
	}
	if _, ok := byID["claude-only-uuid"]; !ok {
		t.Errorf("id without session: %v", byID)
	}
	var hashed int
	for id, a := range byID {
		if len(id) > len("claude-") && a.ProjectPath == "/work/p" {
			hashed++
			if a.SessionID != "s9" {
				t.Errorf("session id: %+v", a)
			}
		}
	}
	if hashed != 1 {
		t.Errorf("the uuid-less record must get a stable derived id: %v", byID)
	}
	// Re-reading a fresh reader yields identical IDs (stable across restarts).
	again := read(t, activity_claude.NewClaudeJSONLReaderAt(dir))
	for _, a := range again {
		if _, ok := byID[a.ID]; !ok {
			t.Errorf("unstable id %q", a.ID)
		}
	}
}

func TestReader_MissingDirAndStaleFiles(t *testing.T) {
	if got := read(t, activity_claude.NewClaudeJSONLReaderAt(filepath.Join(t.TempDir(), "absent"))); got != nil {
		t.Errorf("missing dir: %v", got)
	}
	dir := t.TempDir()
	stale := filepath.Join(dir, "old.jsonl")
	appendTo(t, stale, jsonl(`{"type":"user","uuid":"u","timestamp":"`+now+`"}`))
	old := time.Now().Add(-72 * time.Hour)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}
	appendTo(t, filepath.Join(dir, "notes.txt"), "ignored")
	if got := read(t, activity_claude.NewClaudeJSONLReaderAt(dir)); len(got) != 0 {
		t.Errorf("files untouched since the cutoff and non-jsonl files are skipped: %+v", got)
	}
}
