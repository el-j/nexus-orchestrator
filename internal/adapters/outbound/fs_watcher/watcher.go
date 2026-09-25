package fs_watcher

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"nexus-orchestrator/internal/core/ports"
)

// IgnoredDirs contains directory names that are excluded from scanning and event processing.
var IgnoredDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"build":        true,
	"dist":         true,
	".claude":      true,
	"target":       true,
	"__pycache__":  true,
	".venv":        true,
	".idea":        true,
	".vscode":      true,
}

// AllowedExtensions contains file extensions eligible for workspace change events.
var AllowedExtensions = map[string]bool{
	".go":   true,
	".ts":   true,
	".tsx":  true,
	".js":   true,
	".jsx":  true,
	".rs":   true,
	".py":   true,
	".md":   true,
	".json": true,
}

// AllowedBasenames contains specific filenames always eligible regardless of extension.
var AllowedBasenames = map[string]bool{
	"CLAUDE.md": true,
	"README.md": true,
	"GEMINI.md": true,
	"AGENTS.md": true,
}

// ShouldProcess determines whether a given path qualifies for change events.
func ShouldProcess(path string) bool {
	clean := filepath.Clean(path)
	parts := strings.Split(clean, string(filepath.Separator))
	for _, part := range parts {
		if IgnoredDirs[part] {
			return false
		}
	}

	base := filepath.Base(clean)
	if AllowedBasenames[base] {
		return true
	}

	ext := strings.ToLower(filepath.Ext(clean))
	return AllowedExtensions[ext]
}

// Watcher monitors active project workspaces for filesystem changes
// and automatically triggers debounced indexing/invalidation into the brain.
type Watcher struct {
	watcher  *fsnotify.Watcher
	brain    ports.BrainService
	debounce time.Duration
	onChange func(projectPath, filePath string)

	mu       sync.Mutex
	projects map[string]bool        // projectPath -> true
	dirs     map[string]string      // watched dir -> projectPath
	timers   map[string]*time.Timer // filePath -> pending debounce timer

	stopCh  chan struct{}
	stopped bool
	wg      sync.WaitGroup
}

// Option configures a Watcher.
type Option func(*Watcher)

// WithBrain configures the BrainService to auto-ingest markdown changes.
func WithBrain(b ports.BrainService) Option {
	return func(w *Watcher) {
		w.brain = b
	}
}

// WithDebounce sets the debounce duration (defaults to 300ms).
func WithDebounce(d time.Duration) Option {
	return func(w *Watcher) {
		w.debounce = d
	}
}

// WithOnChange sets a custom callback invoked when a debounced change occurs.
func WithOnChange(fn func(projectPath, filePath string)) Option {
	return func(w *Watcher) {
		w.onChange = fn
	}
}

// New creates and starts a new filesystem watcher.
func New(opts ...Option) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("fs_watcher: new watcher: %w", err)
	}

	w := &Watcher{
		watcher:  fsw,
		debounce: 300 * time.Millisecond,
		projects: make(map[string]bool),
		dirs:     make(map[string]string),
		timers:   make(map[string]*time.Timer),
		stopCh:   make(chan struct{}),
	}
	for _, opt := range opts {
		opt(w)
	}

	w.wg.Add(1)
	go w.eventLoop()

	return w, nil
}

// Watch recursively registers a project directory and its eligible subdirectories.
func (w *Watcher) Watch(projectPath string) error {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("fs_watcher: abs path: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopped {
		return fmt.Errorf("fs_watcher: watcher is closed")
	}

	w.projects[absPath] = true

	return filepath.WalkDir(absPath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		base := d.Name()
		if IgnoredDirs[base] && path != absPath {
			return filepath.SkipDir
		}
		if addErr := w.watcher.Add(path); addErr != nil {
			log.Printf("fs_watcher: add dir %s: %v", path, addErr)
		} else {
			w.dirs[path] = absPath
		}
		return nil
	})
}

// Unwatch removes a project directory and all its subdirectories from the watcher.
func (w *Watcher) Unwatch(projectPath string) error {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("fs_watcher: abs path: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	delete(w.projects, absPath)
	for dir, proj := range w.dirs {
		if proj == absPath || strings.HasPrefix(dir, absPath) {
			_ = w.watcher.Remove(dir)
			delete(w.dirs, dir)
		}
	}
	return nil
}

// Close stops the watcher and cancels all pending debounce timers.
func (w *Watcher) Close() error {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return nil
	}
	w.stopped = true
	close(w.stopCh)

	for _, t := range w.timers {
		t.Stop()
	}
	w.timers = make(map[string]*time.Timer)
	w.mu.Unlock()

	err := w.watcher.Close()
	w.wg.Wait()
	return err
}

func (w *Watcher) eventLoop() {
	defer w.wg.Done()

	for {
		select {
		case <-w.stopCh:
			return
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("fs_watcher error: %v", err)
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)
		}
	}
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
		return
	}

	// Dynamically watch newly created subdirectories
	if event.Op&fsnotify.Create != 0 {
		if fi, err := os.Stat(event.Name); err == nil && fi.IsDir() {
			base := filepath.Base(event.Name)
			if !IgnoredDirs[base] {
				w.mu.Lock()
				proj := w.findProjectForPath(event.Name)
				if proj != "" {
					_ = w.watcher.Add(event.Name)
					w.dirs[event.Name] = proj
				}
				w.mu.Unlock()
			}
			return
		}
	}

	if !ShouldProcess(event.Name) {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopped {
		return
	}

	proj := w.findProjectForPath(event.Name)
	if proj == "" {
		return
	}

	targetFile := event.Name
	if existing, ok := w.timers[targetFile]; ok {
		existing.Stop()
	}

	w.timers[targetFile] = time.AfterFunc(w.debounce, func() {
		w.triggerChange(proj, targetFile)
	})
}

func (w *Watcher) findProjectForPath(path string) string {
	dir := filepath.Dir(path)
	if proj, ok := w.dirs[dir]; ok {
		return proj
	}
	for p := range w.projects {
		if strings.HasPrefix(path, p) {
			return p
		}
	}
	return ""
}

func (w *Watcher) triggerChange(projectPath, filePath string) {
	w.mu.Lock()
	delete(w.timers, filePath)
	stopped := w.stopped
	w.mu.Unlock()

	if stopped {
		return
	}

	if _, err := os.Stat(filePath); err != nil {
		return // File may have been removed
	}

	// Auto-ingest Markdown knowledge files into Brain
	if strings.HasSuffix(strings.ToLower(filePath), ".md") && w.brain != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if count, err := w.brain.IngestFromFile(ctx, projectPath, filePath); err != nil {
			log.Printf("fs_watcher: auto-ingest %s: %v", filePath, err)
		} else if count > 0 {
			log.Printf("fs_watcher: re-indexed %s into brain (%d sections)", filePath, count)
		}
	}

	if w.onChange != nil {
		w.onChange(projectPath, filePath)
	}
}
