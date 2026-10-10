package fs_watcher

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/ports"
)

type changeRecorder struct {
	mu    sync.Mutex
	files []string
}

func (c *changeRecorder) record(_, file string) {
	c.mu.Lock()
	c.files = append(c.files, file)
	c.mu.Unlock()
}

func (c *changeRecorder) has(file string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, f := range c.files {
		if f == file {
			return true
		}
	}
	return false
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func realDir(t *testing.T) string {
	t.Helper()
	d, err := filepath.EvalSymlinks(t.TempDir()) // macOS temp dirs are symlinks; events report real paths
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// /work/app must not claim /work/app2 or /work/application, and nested
// projects resolve to the innermost one.
func TestFindProjectForPath_RespectsPathBoundaries(t *testing.T) {
	w := &Watcher{projects: map[string]bool{"/work/app": true, "/work/app/sub": true}, dirs: map[string]string{}}
	cases := map[string]string{
		"/work/app/main.go":         "/work/app",
		"/work/app/sub/x.go":        "/work/app/sub",
		"/work/app2/main.go":        "",
		"/work/application/main.go": "",
		"/other/main.go":            "",
	}
	for path, want := range cases {
		if got := w.findProjectForPath(path); got != want {
			t.Errorf("findProjectForPath(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestUnwatchOnlyRemovesTheNamedProject(t *testing.T) {
	root := realDir(t)
	app, app2 := filepath.Join(root, "app"), filepath.Join(root, "app2")
	for _, d := range []string{app, app2} {
		if err := os.MkdirAll(filepath.Join(d, "pkg"), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	w, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := w.Watch(app); err != nil {
		t.Fatal(err)
	}
	if err := w.Watch(app2); err != nil {
		t.Fatal(err)
	}
	if err := w.Unwatch(app); err != nil {
		t.Fatal(err)
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	for dir := range w.dirs {
		if isWithin(app, dir) {
			t.Errorf("%s should have been unwatched", dir)
		}
	}
	if w.dirs[filepath.Join(app2, "pkg")] != app2 {
		t.Errorf("the sibling project app2 lost its watches: %v", w.dirs)
	}
}

func TestWatcher_DebouncedChangesNewDirsAndIgnoredDirs(t *testing.T) {
	root := realDir(t)
	rec := &changeRecorder{}
	w, err := New(WithDebounce(20*time.Millisecond), WithOnChange(rec.record))
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "pkg"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := w.Watch(root); err != nil {
		t.Fatal(err)
	}

	changed := filepath.Join(root, "main.go")
	for i := 0; i < 3; i++ { // rapid writes collapse into one debounced callback
		if err := os.WriteFile(changed, []byte("package main // "+string(rune('a'+i))), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	waitFor(t, "debounced change", func() bool { return rec.has(changed) })

	// A directory created after Watch is picked up dynamically.
	sub := filepath.Join(root, "late")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	lateFile := filepath.Join(sub, "x.ts")
	if err := os.WriteFile(lateFile, []byte("export {}"), 0o600); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "change inside a directory created after Watch", func() bool { return rec.has(lateFile) })

	// Ignored directories and unlisted extensions never produce events.
	ignored := filepath.Join(root, "node_modules", "pkg", "i.go")
	if err := os.WriteFile(ignored, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "image.png"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	time.Sleep(150 * time.Millisecond)
	if rec.has(ignored) || rec.has(filepath.Join(root, "image.png")) {
		t.Errorf("ignored paths produced events: %v", rec.files)
	}
}

type ingestRecorder struct {
	ports.BrainService
	mu    sync.Mutex
	calls []string
	err   error
}

func (i *ingestRecorder) IngestFromFile(_ context.Context, project, file string) (int, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.calls = append(i.calls, project+"|"+file)
	return 2, i.err
}

func (i *ingestRecorder) count() int {
	i.mu.Lock()
	defer i.mu.Unlock()
	return len(i.calls)
}

func TestWatcher_ReindexesMarkdownIntoTheBrain(t *testing.T) {
	root := realDir(t)
	brain := &ingestRecorder{}
	w, err := New(WithDebounce(20*time.Millisecond), WithBrain(brain))
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := w.Watch(root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("# t\n## s\nbody body body body body\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "markdown ingest", func() bool { return brain.count() >= 1 })
	if err := os.WriteFile(filepath.Join(root, "code.go"), []byte("package x"), 0o600); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	brain.mu.Lock()
	defer brain.mu.Unlock()
	for _, c := range brain.calls {
		if filepath.Ext(c) != ".md" {
			t.Errorf("only markdown is ingested, got %q", c)
		}
	}
}

func TestTriggerChange_SkipsRemovedFilesAndToleratesIngestErrors(t *testing.T) {
	root := realDir(t)
	rec := &changeRecorder{}
	brain := &ingestRecorder{err: os.ErrPermission}
	w, err := New(WithOnChange(rec.record), WithBrain(brain))
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	gone := filepath.Join(root, "gone.md")
	w.triggerChange(root, gone)
	if rec.has(gone) || brain.count() != 0 {
		t.Error("a file removed before the debounce fired must be ignored")
	}
	present := filepath.Join(root, "here.md")
	if err := os.WriteFile(present, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	w.triggerChange(root, present)
	if !rec.has(present) || brain.count() != 1 {
		t.Error("an ingest error must not suppress the change callback")
	}
}

func TestWatcher_CloseIsIdempotentAndBlocksFurtherWatches(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
	if err := w.Watch(t.TempDir()); err == nil {
		t.Error("Watch after Close must fail")
	}
	// Events delivered after Close are dropped silently.
	w.triggerChange(t.TempDir(), "x")
}

func TestShouldProcess(t *testing.T) {
	for p, want := range map[string]bool{
		"/p/a.go": true, "/p/b.TS": true, "/p/README.md": true, "/p/Makefile": false,
		"/p/node_modules/x.go": false, "/p/.git/config.json": false, "/p/dist/a.js": false, "/p/img.png": false,
	} {
		if got := ShouldProcess(p); got != want {
			t.Errorf("ShouldProcess(%q) = %v, want %v", p, got, want)
		}
	}
}
