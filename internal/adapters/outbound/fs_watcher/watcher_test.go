package fs_watcher_test

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/outbound/fs_watcher"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

var _ ports.BrainService = (*mockBrainService)(nil)

type mockBrainService struct {
	ingestCalls atomic.Int32
	lastProject string
	lastFile    string
}

func (m *mockBrainService) GetContext(ctx context.Context, q domain.ContextQuery) (domain.ContextResponse, error) {
	return domain.ContextResponse{}, nil
}

func (m *mockBrainService) GetFocusedContext(ctx context.Context, q domain.ContextQuery) (domain.ContextResponse, error) {
	return domain.ContextResponse{}, nil
}

func (m *mockBrainService) IngestKnowledge(ctx context.Context, k domain.ProjectKnowledge) (domain.ProjectKnowledge, error) {
	return domain.ProjectKnowledge{}, nil
}

func (m *mockBrainService) IngestFromFile(ctx context.Context, projectPath, filePath string) (int, error) {
	m.ingestCalls.Add(1)
	m.lastProject = projectPath
	m.lastFile = filePath
	return 2, nil
}

func (m *mockBrainService) SearchKnowledge(ctx context.Context, projectPath, query string, maxTokens int) ([]domain.ContextSection, error) {
	return nil, nil
}

func (m *mockBrainService) GetFileMap(ctx context.Context, projectPath, focusArea string) ([]string, error) {
	return nil, nil
}

func (m *mockBrainService) InitProject(ctx context.Context, projectPath, claudeMDPath string) (domain.BrainStatus, error) {
	return domain.BrainStatus{}, nil
}

func (m *mockBrainService) GetStatus(ctx context.Context, projectPath string) (domain.BrainStatus, error) {
	return domain.BrainStatus{}, nil
}

func (m *mockBrainService) ListKnowledge(ctx context.Context, projectPath, kind string) ([]domain.ProjectKnowledge, error) {
	return nil, nil
}

func (m *mockBrainService) DeleteKnowledge(ctx context.Context, id string) error {
	return nil
}

func (m *mockBrainService) GetOnboardingContext(ctx context.Context, projectPath string, maxTokens int) (string, error) {
	return "", nil
}

func TestShouldProcess(t *testing.T) {
	cases := []struct {
		path     string
		expected bool
	}{
		{"src/main.go", true},
		{"frontend/app.tsx", true},
		{"rust/main.rs", true},
		{"script.py", true},
		{"docs/CLAUDE.md", true},
		{"README.md", true},
		{".git/HEAD", false},
		{".git/objects/abc", false},
		{"node_modules/vue/package.json", false},
		{".claude/tasks/TASK-1.md", false},
		{"build/binary", false},
		{"dist/bundle.js", false},
		{"assets/photo.png", false},
		{"temp.log", false},
	}

	for _, c := range cases {
		got := fs_watcher.ShouldProcess(c.path)
		if got != c.expected {
			t.Errorf("ShouldProcess(%q) = %v; expected %v", c.path, got, c.expected)
		}
	}
}

func TestWatcher_Debounce(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "service.go")

	if err := os.WriteFile(testFile, []byte("package main"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	var changeCount atomic.Int32
	changedFileCh := make(chan string, 10)

	w, err := fs_watcher.New(
		fs_watcher.WithDebounce(100*time.Millisecond),
		fs_watcher.WithOnChange(func(proj, file string) {
			changeCount.Add(1)
			changedFileCh <- file
		}),
	)
	if err != nil {
		t.Fatalf("New watcher: %v", err)
	}
	defer w.Close()

	if err := w.Watch(dir); err != nil {
		t.Fatalf("Watch dir: %v", err)
	}

	// Write 5 times rapidly within 30ms (well under the 100ms debounce window)
	for i := 0; i < 5; i++ {
		if err := os.WriteFile(testFile, []byte("package main // v"+string(rune('0'+i))), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		time.Sleep(5 * time.Millisecond)
	}

	select {
	case file := <-changedFileCh:
		if filepath.Clean(file) != filepath.Clean(testFile) {
			t.Errorf("expected %q, got %q", testFile, file)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for debounced change event")
	}

	// Wait an extra 150ms to ensure no trailing duplicate events fire
	time.Sleep(150 * time.Millisecond)
	if got := changeCount.Load(); got != 1 {
		t.Errorf("expected exactly 1 debounced change event, got %d", got)
	}
}

func TestWatcher_AutoIngestBrain(t *testing.T) {
	dir := t.TempDir()
	claudeMD := filepath.Join(dir, "CLAUDE.md")

	if err := os.WriteFile(claudeMD, []byte("## Build\ngo build ./..."), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	mockBrain := &mockBrainService{}
	eventReceived := make(chan struct{}, 1)

	w, err := fs_watcher.New(
		fs_watcher.WithBrain(mockBrain),
		fs_watcher.WithDebounce(50*time.Millisecond),
		fs_watcher.WithOnChange(func(proj, file string) {
			select {
			case eventReceived <- struct{}{}:
			default:
			}
		}),
	)
	if err != nil {
		t.Fatalf("New watcher: %v", err)
	}
	defer w.Close()

	if err := w.Watch(dir); err != nil {
		t.Fatalf("Watch dir: %v", err)
	}

	// Modify CLAUDE.md
	if err := os.WriteFile(claudeMD, []byte("## Build\ngo test ./...\n## Arch\nModular"), 0o644); err != nil {
		t.Fatalf("modify file: %v", err)
	}

	select {
	case <-eventReceived:
		if mockBrain.ingestCalls.Load() == 0 {
			t.Error("expected IngestFromFile to be called on markdown change")
		}
		if mockBrain.lastFile != claudeMD {
			t.Errorf("expected file %q, got %q", claudeMD, mockBrain.lastFile)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for markdown ingest event")
	}
}

func TestWatcher_Unwatch(t *testing.T) {
	dir := t.TempDir()
	w, err := fs_watcher.New(fs_watcher.WithDebounce(50 * time.Millisecond))
	if err != nil {
		t.Fatalf("New watcher: %v", err)
	}
	defer w.Close()

	if err := w.Watch(dir); err != nil {
		t.Fatalf("Watch: %v", err)
	}
	if err := w.Unwatch(dir); err != nil {
		t.Fatalf("Unwatch: %v", err)
	}
}
