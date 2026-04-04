package activity_claude_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/outbound/activity_claude"
	"nexus-orchestrator/internal/core/domain"
)

func TestSourceName(t *testing.T) {
	r := activity_claude.NewClaudeJSONLReader()
	if got := r.SourceName(); got != "claude-jsonl" {
		t.Errorf("SourceName() = %q, want %q", got, "claude-jsonl")
	}
}

func TestJSONLReader_ReadActivities_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "myproject")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Use a timestamp one minute ago so activities pass the since filter.
	ts := time.Now().UTC().Add(-30 * time.Second).Format(time.RFC3339)
	since := time.Now().UTC().Add(-1 * time.Hour)

	jsonl := fmt.Sprintf(
		`{"type":"user","message":{"role":"user","content":"help me code"},"timestamp":%q,"cwd":"/tmp/myproject","sessionId":"sess1","uuid":"uuid1"}`+"\n"+
			`{"type":"assistant","message":{"role":"assistant","content":"I will help you","model":"claude-opus-4-5","usage":{"input_tokens":100,"output_tokens":50}},"timestamp":%q,"cwd":"/tmp/myproject","sessionId":"sess1","uuid":"uuid2","parentUuid":"uuid1"}`+"\n"+
			`{"type":"tool_use","name":"bash","input":{"command":"ls -la"},"timestamp":%q,"cwd":"/tmp/myproject","sessionId":"sess1","uuid":"uuid3"}`+"\n",
		ts, ts, ts,
	)

	file := filepath.Join(projectDir, "session.jsonl")
	if err := os.WriteFile(file, []byte(jsonl), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := activity_claude.NewClaudeJSONLReaderAt(tmpDir)
	activities, err := reader.ReadActivities(context.Background(), since)
	if err != nil {
		t.Fatalf("ReadActivities error: %v", err)
	}
	if len(activities) != 3 {
		t.Fatalf("expected 3 activities, got %d", len(activities))
	}

	// Verify activity types.
	if activities[0].ActivityType != domain.ActivityTypeMessage {
		t.Errorf("activity[0] type = %q, want %q", activities[0].ActivityType, domain.ActivityTypeMessage)
	}
	if activities[1].ActivityType != domain.ActivityTypeGeneration {
		t.Errorf("activity[1] type = %q, want %q", activities[1].ActivityType, domain.ActivityTypeGeneration)
	}
	if activities[2].ActivityType != domain.ActivityTypeToolUse {
		t.Errorf("activity[2] type = %q, want %q", activities[2].ActivityType, domain.ActivityTypeToolUse)
	}

	// No raw message content in summaries.
	for _, a := range activities {
		if strings.Contains(a.Summary, "help me code") || strings.Contains(a.Summary, "I will help you") {
			t.Errorf("summary contains raw message content: %q", a.Summary)
		}
	}

	// Assistant summary mentions token count.
	if activities[1].TokensOut != 50 {
		t.Errorf("activities[1].TokensOut = %d, want 50", activities[1].TokensOut)
	}
	if !strings.Contains(activities[1].Summary, "50 tokens") {
		t.Errorf("activities[1].Summary = %q, want it to contain \"50 tokens\"", activities[1].Summary)
	}

	// Tool use summary names the tool.
	if activities[2].Summary != "Using bash" {
		t.Errorf("activities[2].Summary = %q, want \"Using bash\"", activities[2].Summary)
	}

	// IDs are built from sessionId + uuid.
	if activities[0].ID != "claude-sess1-uuid1" {
		t.Errorf("activities[0].ID = %q, want \"claude-sess1-uuid1\"", activities[0].ID)
	}

	// All activities report the correct agent.
	for i, a := range activities {
		if a.AgentName != "claude" {
			t.Errorf("activities[%d].AgentName = %q, want \"claude\"", i, a.AgentName)
		}
	}

	// Model propagated on assistant activity.
	if activities[1].Model != "claude-opus-4-5" {
		t.Errorf("activities[1].Model = %q, want \"claude-opus-4-5\"", activities[1].Model)
	}

	// ProjectPath propagated from cwd.
	for i, a := range activities {
		if a.ProjectPath != "/tmp/myproject" {
			t.Errorf("activities[%d].ProjectPath = %q, want \"/tmp/myproject\"", i, a.ProjectPath)
		}
	}
}

func TestJSONLReader_ReadActivities_IncrementalOffset(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "proj"), 0o755); err != nil {
		t.Fatal(err)
	}

	ts := time.Now().UTC().Add(-30 * time.Second).Format(time.RFC3339)
	since := time.Now().UTC().Add(-1 * time.Hour)

	line1 := fmt.Sprintf(`{"type":"user","message":{},"timestamp":%q,"sessionId":"s1","uuid":"u1"}`+"\n", ts)
	file := filepath.Join(tmpDir, "proj", "session.jsonl")
	if err := os.WriteFile(file, []byte(line1), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := activity_claude.NewClaudeJSONLReaderAt(tmpDir)

	// First read: should return 1 activity.
	got, err := reader.ReadActivities(context.Background(), since)
	if err != nil {
		t.Fatalf("first ReadActivities error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("first read: expected 1 activity, got %d", len(got))
	}

	// Second read without new content: should return 0 activities.
	got2, err := reader.ReadActivities(context.Background(), since)
	if err != nil {
		t.Fatalf("second ReadActivities error: %v", err)
	}
	if len(got2) != 0 {
		t.Fatalf("second read (no new lines): expected 0 activities, got %d", len(got2))
	}

	// Append a second line to the file.
	line2 := fmt.Sprintf(`{"type":"tool_use","name":"read_file","timestamp":%q,"sessionId":"s1","uuid":"u2"}`+"\n", ts)
	f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(line2); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	// Third read: should return only the new activity.
	got3, err := reader.ReadActivities(context.Background(), since)
	if err != nil {
		t.Fatalf("third ReadActivities error: %v", err)
	}
	if len(got3) != 1 {
		t.Fatalf("third read: expected 1 activity, got %d", len(got3))
	}
	if got3[0].ActivityType != domain.ActivityTypeToolUse {
		t.Errorf("third read activity type = %q, want tool_use", got3[0].ActivityType)
	}
}

func TestJSONLReader_ReadActivities_MissingBaseDir(t *testing.T) {
	reader := activity_claude.NewClaudeJSONLReaderAt("/nonexistent/path/that/does/not/exist")
	got, err := reader.ReadActivities(context.Background(), time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("expected no error for missing base dir, got: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 activities for missing base dir, got %d", len(got))
	}
}

func TestJSONLReader_ReadActivities_SkipsOldActivities(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "proj"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Activity timestamped 2 hours ago.
	oldTS := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	since := time.Now().UTC().Add(-1 * time.Hour)

	line := fmt.Sprintf(`{"type":"user","message":{},"timestamp":%q,"sessionId":"s1","uuid":"u1"}`+"\n", oldTS)
	file := filepath.Join(tmpDir, "proj", "session.jsonl")
	if err := os.WriteFile(file, []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}

	// Force mod time to be recent so file is not skipped by the mod-time check.
	now := time.Now()
	if err := os.Chtimes(file, now, now); err != nil {
		t.Fatal(err)
	}

	reader := activity_claude.NewClaudeJSONLReaderAt(tmpDir)
	got, err := reader.ReadActivities(context.Background(), since)
	if err != nil {
		t.Fatalf("ReadActivities error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 activities (all older than since), got %d", len(got))
	}
}
