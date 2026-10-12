package activity_continue_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/outbound/activity_continue"
)

func index(t *testing.T, dir string, entries ...map[string]any) {
	t.Helper()
	writeJSON(t, filepath.Join(dir, "sessions.json"), map[string]any{"sessions": entries})
}

func session(t *testing.T, dir, id string, body map[string]any) {
	t.Helper()
	writeJSON(t, filepath.Join(dir, id+".json"), body)
}

func TestReadActivities_IndexProblems(t *testing.T) {
	dir := t.TempDir()
	// An index that is a directory cannot be read: a real error, not "no sessions".
	if err := os.Mkdir(filepath.Join(dir, "sessions.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := newReaderWithDir(dir).ReadActivities(context.Background(), time.Unix(0, 0)); err == nil ||
		!strings.Contains(err.Error(), "read sessions index") {
		t.Errorf("err = %v", err)
	}

	// An index in an unknown format is treated as empty.
	dir2 := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir2, "sessions.json"), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if acts, err := newReaderWithDir(dir2).ReadActivities(context.Background(), time.Unix(0, 0)); err != nil || acts != nil {
		t.Errorf("acts = %v, err = %v", acts, err)
	}
}

func TestReadActivities_StopsWhenCancelled(t *testing.T) {
	dir := t.TempDir()
	index(t, dir, map[string]any{"sessionId": "a"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := newReaderWithDir(dir).ReadActivities(ctx, time.Unix(0, 0)); err == nil {
		t.Error("want the context error")
	}
}

func TestReadActivities_SkipsSessionsItCannotUse(t *testing.T) {
	dir := t.TempDir()
	index(t, dir,
		map[string]any{"sessionId": "missing"},
		map[string]any{"sessionId": "unreadable"},
		map[string]any{"sessionId": "garbled"},
		map[string]any{"sessionId": "stale"},
		map[string]any{"sessionId": "old-message"},
	)
	if err := os.Mkdir(filepath.Join(dir, "unreadable.json"), 0o755); err != nil { // Stat works, ReadFile does not
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "garbled.json"), []byte("{oops"), 0o644); err != nil {
		t.Fatal(err)
	}
	session(t, dir, "stale", map[string]any{"history": []map[string]any{{"role": "user", "timestamp": 1700000000}}})
	if err := os.Chtimes(filepath.Join(dir, "stale.json"), time.Unix(1, 0), time.Unix(1, 0)); err != nil {
		t.Fatal(err)
	}
	session(t, dir, "old-message", map[string]any{"history": []map[string]any{{"role": "user", "timestamp": 100}}})

	acts, err := newReaderWithDir(dir).ReadActivities(context.Background(), time.Unix(1700000000, 0))
	if err != nil || len(acts) != 0 {
		t.Errorf("acts = %v, err = %v", acts, err)
	}
}

func TestReadActivities_TitlesAndModels(t *testing.T) {
	dir := t.TempDir()
	long := strings.Repeat("x", 120)
	index(t, dir,
		map[string]any{"sessionId": "from-index", "title": "Index title"},
		map[string]any{"sessionId": "untitled"},
		map[string]any{"sessionId": "long", "title": long},
		map[string]any{"sessionId": "assistant-model"},
	)
	hist := []map[string]any{{"role": "user", "timestamp": 1700000100}}
	session(t, dir, "from-index", map[string]any{"history": hist})
	session(t, dir, "untitled", map[string]any{"history": hist})
	session(t, dir, "long", map[string]any{"history": hist})
	session(t, dir, "assistant-model", map[string]any{"history": []map[string]any{
		{"role": "user", "timestamp": 1700000100},
		{"role": "assistant", "model": "qwen", "timestamp": 1700000101},
		{"role": "assistant", "timestamp": 1700000102},
	}})

	acts, err := newReaderWithDir(dir).ReadActivities(context.Background(), time.Unix(1700000000, 0))
	if err != nil || len(acts) != 4 {
		t.Fatalf("acts = %d, err = %v", len(acts), err)
	}
	byID := map[string]string{}
	models := map[string]string{}
	for _, a := range acts {
		byID[a.SessionID] = a.Summary
		models[a.SessionID] = a.Model
	}
	if byID["from-index"] != "Index title" || byID["untitled"] != "Continue chat" || len(byID["long"]) != 80 {
		t.Errorf("summaries = %v", byID)
	}
	if models["assistant-model"] != "qwen" {
		t.Errorf("models = %v", models)
	}
}

func TestNewContinueSessionReader_WithoutAHomeDirectory(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("home", "")
	r := activity_continue.NewContinueSessionReader()
	if r.SourceName() != "continue-sessions" {
		t.Error("reader not constructed")
	}
	if _, err := r.ReadActivities(context.Background(), time.Unix(0, 0)); err != nil {
		t.Errorf("a missing sessions directory is not an error: %v", err)
	}
}
