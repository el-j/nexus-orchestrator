package activity_continue_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	activity_continue "nexus-orchestrator/internal/adapters/outbound/activity_continue"
	"nexus-orchestrator/internal/core/domain"
)

func TestReadActivities_ReturnsSingleActivity(t *testing.T) {
	dir := t.TempDir()

	sessionID := "test-session-abc"
	title := "Fix the bug"
	workspaceDir := "/path/to/project"
	msgTimestamp := int64(1700000100) // 2023-11-14 ...

	// Write sessions.json index.
	index := map[string]interface{}{
		"sessions": []map[string]interface{}{
			{
				"sessionId":          sessionID,
				"title":              title,
				"dateCreated":        "2023-11-14T00:01:00Z",
				"workspaceDirectory": workspaceDir,
			},
		},
	}
	writeJSON(t, filepath.Join(dir, "sessions.json"), index)

	// Write individual session file.
	session := map[string]interface{}{
		"sessionId": sessionID,
		"title":     title,
		"history": []map[string]interface{}{
			{"role": "user", "content": "How do I fix this?", "timestamp": msgTimestamp},
			{"role": "assistant", "content": "Try this...", "timestamp": msgTimestamp + 5},
		},
	}
	writeJSON(t, filepath.Join(dir, sessionID+".json"), session)

	r := newReaderWithDir(dir)
	since := time.Unix(msgTimestamp-100, 0) // well before the messages

	acts, err := r.ReadActivities(context.Background(), since)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(acts) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(acts))
	}

	a := acts[0]
	if a.ID != "continue-"+sessionID {
		t.Errorf("ID: got %q, want %q", a.ID, "continue-"+sessionID)
	}
	if a.AgentName != "continue" {
		t.Errorf("AgentName: got %q", a.AgentName)
	}
	if a.ActivityType != domain.ActivityTypeMessage {
		t.Errorf("ActivityType: got %q", a.ActivityType)
	}
	if a.Summary != title {
		t.Errorf("Summary: got %q, want %q", a.Summary, title)
	}
	if a.ProjectPath != workspaceDir {
		t.Errorf("ProjectPath: got %q, want %q", a.ProjectPath, workspaceDir)
	}
	if a.SessionID != sessionID {
		t.Errorf("SessionID: got %q", a.SessionID)
	}
	if a.Metadata["messageCount"] != "2" {
		t.Errorf("messageCount metadata: got %q, want 2", a.Metadata["messageCount"])
	}
	if a.Metadata["source"] != "continue-sessions" {
		t.Errorf("source metadata: got %q", a.Metadata["source"])
	}
	want := time.Unix(msgTimestamp+5, 0)
	if !a.Timestamp.Equal(want) {
		t.Errorf("Timestamp: got %v, want %v", a.Timestamp, want)
	}
}

func TestReadActivities_MissingSessions_ReturnsNil(t *testing.T) {
	dir := t.TempDir()
	r := newReaderWithDir(dir)

	acts, err := r.ReadActivities(context.Background(), time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if acts != nil {
		t.Errorf("expected nil, got %v", acts)
	}
}

func TestReadActivities_SinceFiltersOldSessions(t *testing.T) {
	dir := t.TempDir()

	sessionID := "old-session"
	msgTimestamp := int64(1000000) // very old

	index := map[string]interface{}{
		"sessions": []map[string]interface{}{
			{
				"sessionId":          sessionID,
				"title":              "Old chat",
				"dateCreated":        "2001-01-01T00:00:00Z",
				"workspaceDirectory": "/old",
			},
		},
	}
	writeJSON(t, filepath.Join(dir, "sessions.json"), index)

	session := map[string]interface{}{
		"sessionId": sessionID,
		"title":     "Old chat",
		"history": []map[string]interface{}{
			{"role": "user", "content": "hello", "timestamp": msgTimestamp},
		},
	}
	writeJSON(t, filepath.Join(dir, sessionID+".json"), session)

	r := newReaderWithDir(dir)
	// since is after the message timestamp — should produce no activities
	since := time.Unix(msgTimestamp+1, 0)

	acts, err := r.ReadActivities(context.Background(), since)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(acts) != 0 {
		t.Errorf("expected 0 activities, got %d", len(acts))
	}
}

func TestSourceName(t *testing.T) {
	r := activity_continue.NewContinueSessionReader()
	if r.SourceName() != "continue-sessions" {
		t.Errorf("SourceName: got %q", r.SourceName())
	}
}

// --- helpers ---

func writeJSON(t *testing.T, path string, v interface{}) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal JSON: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// newReaderWithDir creates a ContinueSessionReader pointed at the given directory.
// Exported fields aren't available so we set sessionsDir via the test constructor helper.
func newReaderWithDir(dir string) *activity_continue.ContinueSessionReader {
	return activity_continue.NewContinueSessionReaderWithDir(dir)
}
