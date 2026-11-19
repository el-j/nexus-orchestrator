package fs_watcher

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

// manualWatcher builds a Watcher around a hand-made fsnotify watcher, so a test can feed its event
// and error channels and close them, and runs the event loop on it.
func manualWatcher(t *testing.T) (w *Watcher, events chan fsnotify.Event, errs chan error) {
	t.Helper()
	events, errs = make(chan fsnotify.Event), make(chan error)
	w = &Watcher{
		watcher:  &fsnotify.Watcher{Events: events, Errors: errs},
		debounce: time.Hour,
		projects: map[string]bool{},
		dirs:     map[string]string{},
		timers:   map[string]*time.Timer{},
		stopCh:   make(chan struct{}),
	}
	w.wg.Add(1)
	go w.eventLoop()
	return w, events, errs
}

func waitStopped(t *testing.T, w *Watcher) {
	t.Helper()
	done := make(chan struct{})
	go func() { w.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the event loop did not stop")
	}
}

func TestEventLoop_LogsAWatcherErrorAndStopsWhenEitherChannelCloses(t *testing.T) {
	t.Run("an error is logged and the loop goes on", func(t *testing.T) {
		w, _, errs := manualWatcher(t)
		errs <- errors.New("inotify queue overflow")
		close(errs)
		waitStopped(t, w)
	})
	t.Run("the events channel closes", func(t *testing.T) {
		w, events, _ := manualWatcher(t)
		close(events)
		waitStopped(t, w)
	})
}

func TestHandleEvent_IgnoresEventsAfterCloseAndOutsideAnyProject(t *testing.T) {
	w, _, _ := manualWatcher(t)
	defer close(w.stopCh)
	event := fsnotify.Event{Name: filepath.Join(t.TempDir(), "notes.md"), Op: fsnotify.Write}

	w.handleEvent(event) // no project is watched: nothing is scheduled
	if len(w.timers) != 0 {
		t.Errorf("an event outside every project scheduled %d timers", len(w.timers))
	}
	w.stopped = true
	w.handleEvent(event)
	if len(w.timers) != 0 {
		t.Errorf("a closed watcher scheduled %d timers", len(w.timers))
	}
}

func TestClose_StopsTheTimersOfPendingChanges(t *testing.T) {
	w, err := New(WithDebounce(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	fired := make(chan struct{}, 1)
	w.mu.Lock()
	w.timers["pending.md"] = time.AfterFunc(50*time.Millisecond, func() { fired <- struct{}{} })
	w.mu.Unlock()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-fired:
		t.Error("a change that was still waiting fired after Close")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestNewWatchAndUnwatch_ReportTheSystemFailingUnderThem(t *testing.T) {
	t.Run("no file watches left", func(t *testing.T) {
		defer func(old func() (*fsnotify.Watcher, error)) { newFSWatcher = old }(newFSWatcher)
		newFSWatcher = func() (*fsnotify.Watcher, error) { return nil, errors.New("too many open files") }
		if _, err := New(); err == nil || !strings.Contains(err.Error(), "new watcher") {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("the working directory is gone", func(t *testing.T) {
		defer func(old func(string) (string, error)) { absPath = old }(absPath)
		absPath = func(string) (string, error) { return "", errors.New("no working directory") }
		w, err := New()
		if err != nil {
			t.Fatal(err)
		}
		defer w.Close()
		if err := w.Watch("p"); err == nil || !strings.Contains(err.Error(), "abs path") {
			t.Errorf("Watch: %v", err)
		}
		if err := w.Unwatch("p"); err == nil || !strings.Contains(err.Error(), "abs path") {
			t.Errorf("Unwatch: %v", err)
		}
	})
	t.Run("a directory cannot be added", func(t *testing.T) {
		w, err := New()
		if err != nil {
			t.Fatal(err)
		}
		if err := w.watcher.Close(); err != nil { // the system watcher is gone, Add now fails
			t.Fatal(err)
		}
		if err := w.Watch(t.TempDir()); err != nil {
			t.Fatalf("a directory that cannot be added is logged, not fatal: %v", err)
		}
		if len(w.dirs) != 0 {
			t.Errorf("dirs = %v, want none registered", w.dirs)
		}
		_ = w.Close()
	})
}

func TestHandleEvent_IgnoresEventsThatAreNeitherWritesNorCreates(t *testing.T) {
	w, _, _ := manualWatcher(t)
	defer close(w.stopCh)
	for _, op := range []fsnotify.Op{fsnotify.Remove, fsnotify.Rename, fsnotify.Chmod} {
		w.handleEvent(fsnotify.Event{Name: "notes.md", Op: op})
	}
	if len(w.timers) != 0 {
		t.Errorf("%d timers scheduled for events that change no content", len(w.timers))
	}
}
