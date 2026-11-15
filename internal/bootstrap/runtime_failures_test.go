package bootstrap_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/outbound/fs_watcher"
	"nexus-orchestrator/internal/adapters/outbound/fs_writer"
	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/bootstrap"
	"nexus-orchestrator/internal/core/services"
)

// syncBuffer is a log destination safe for the servers' goroutines.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// captureLog sends the standard logger to a buffer for the rest of the test.
func captureLog(t *testing.T) *syncBuffer {
	t.Helper()
	out := &syncBuffer{}
	log.SetOutput(out)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return out
}

func TestDefaultDBPath_FallsBackToTheWorkingDirectoryWithoutAHomeDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the home directory comes from USERPROFILE there")
	}
	t.Setenv("HOME", "")
	if got := bootstrap.DefaultDBPath(); got != "nexus.db" {
		t.Errorf("DefaultDBPath() = %q, want nexus.db", got)
	}
}

func TestStart_FailsWhenTheDataDirectoryCannotBeCreated(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the home directory comes from USERPROFILE there")
	}
	home := filepath.Join(t.TempDir(), "home-is-a-file")
	if err := os.WriteFile(home, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	_, err := bootstrap.Start(context.Background(), bootstrap.Config{DBPath: bootstrap.DefaultDBPath()}, nil)
	if err == nil || !strings.Contains(err.Error(), "create data directory") {
		t.Errorf("err = %v", err)
	}
}

// seedBrokenDatabase creates a database whose task update is refused (with one task another tool
// wrote), whose runtime configuration is not JSON, and whose provider table fails when read.
func seedBrokenDatabase(t *testing.T, path string) {
	t.Helper()
	seed, err := repo_sqlite.New(path)
	if err != nil {
		t.Fatal(err)
	}
	_ = seed.Close()
	raw, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	for _, q := range []string{
		`INSERT INTO tasks (id, project_path, target_file, instruction, status, created_at, updated_at) VALUES ('f', '/p', 'f.go', 'x', 'in_progress', 1700000000000, 1700000000000)`,
		`CREATE TRIGGER refuse_task_updates BEFORE UPDATE ON tasks BEGIN SELECT RAISE(ABORT, 'injected failure'); END`,
		`INSERT OR REPLACE INTO runtime_config (id, json, updated_at) VALUES (1, '{not json', 1)`,
		`ALTER TABLE provider_configs RENAME TO provider_configs_real`,
		`CREATE VIEW provider_configs AS SELECT * FROM provider_configs_real UNION ALL SELECT abs(-9223372036854775808), NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL`,
	} {
		if _, err := raw.Exec(q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
}

func TestStart_KeepsGoingWhenTheDatabaseIsDamagedAndSaysSo(t *testing.T) {
	isolate(t)
	logs := captureLog(t)
	cfg := bootstrap.Config{DBPath: filepath.Join(t.TempDir(), "damaged.db"), ListenAddr: freeAddr(t), MCPAddr: freeAddr(t), ScanInterval: time.Hour}
	seedBrokenDatabase(t, cfg.DBPath)

	rt, err := bootstrap.Start(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("a damaged table must not stop the daemon: %v", err)
	}
	rt.Close()
	for _, want := range []string{"startup: repair task rows", "startup: get runtime config", "startup: list provider configs"} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("the log lacks %q:\n%s", want, logs.String())
		}
	}
}

func TestStart_ReportsAWatcherThatCannotBeCreatedOrCannotWatchTheWorkspace(t *testing.T) {
	cfg := func(t *testing.T) bootstrap.Config {
		isolate(t)
		return bootstrap.Config{DBPath: filepath.Join(t.TempDir(), "w.db"), ListenAddr: freeAddr(t), MCPAddr: freeAddr(t), ScanInterval: time.Hour}
	}
	t.Run("creation fails", func(t *testing.T) {
		c := cfg(t)
		logs := captureLog(t)
		defer bootstrap.SetNewWatcher(func(...fs_watcher.Option) (*fs_watcher.Watcher, error) { return nil, errors.New("no watches left") })()
		rt, err := bootstrap.Start(context.Background(), c, nil)
		if err != nil {
			t.Fatal(err)
		}
		rt.Close()
		if !strings.Contains(logs.String(), "startup: create fs watcher") || rt.Watcher != nil {
			t.Errorf("watcher = %v, log:\n%s", rt.Watcher, logs.String())
		}
	})
	t.Run("the workspace cannot be watched", func(t *testing.T) {
		c := cfg(t)
		project := t.TempDir()
		if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Chdir(project)
		logs := captureLog(t)
		defer bootstrap.SetNewWatcher(func(opts ...fs_watcher.Option) (*fs_watcher.Watcher, error) {
			w, err := fs_watcher.New(opts...)
			if err == nil {
				_ = w.Close() // a closed watcher refuses Watch
			}
			return w, err
		})()
		rt, err := bootstrap.Start(context.Background(), c, nil)
		if err != nil {
			t.Fatal(err)
		}
		rt.Close()
		if !strings.Contains(logs.String(), "startup: watch workspace") {
			t.Errorf("log:\n%s", logs.String())
		}
	})
}

func TestStart_KeepsOnlyTheFirstServerFailure(t *testing.T) {
	isolate(t)
	httpPort, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer httpPort.Close()
	mcpPort, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer mcpPort.Close()
	cfg := bootstrap.Config{DBPath: filepath.Join(t.TempDir(), "both.db"), ListenAddr: httpPort.Addr().String(), MCPAddr: mcpPort.Addr().String(), ScanInterval: time.Hour}
	rt, err := bootstrap.Start(context.Background(), cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()
	time.Sleep(500 * time.Millisecond) // both servers have failed by now; only the first error was kept
	select {
	case err := <-rt.Failed():
		if err == nil {
			t.Error("a nil failure")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("neither busy port was reported")
	}
	select {
	case err := <-rt.Failed():
		t.Errorf("a second failure was queued: %v", err)
	default:
	}
}

func TestScanLoop_LogsAScanThatFails(t *testing.T) {
	logs := captureLog(t)
	repo, err := repo_sqlite.New(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	// No system scanner is wired, so every scan fails.
	orch := services.NewOrchestrator(services.NewDiscoveryService(), repo, fs_writer.New(), nil, services.WithDisableBackgroundWorkers())
	defer orch.Stop()
	rt := &bootstrap.Runtime{Config: bootstrap.Config{ScanInterval: 10 * time.Millisecond}, Orchestrator: orch}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	rt.ScanLoop(ctx)
	for _, want := range []string{"startup: initial scan", "discovery: scan error"} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("the log lacks %q:\n%s", want, logs.String())
		}
	}
}

func TestStart_WatchesAProjectWorkspaceReadsAgentLogsAndRescansProviders(t *testing.T) {
	isolate(t)
	home := os.Getenv("HOME")
	for _, dir := range []string{".claude", ".continue"} { // the agent log folders the activity readers look for
		if err := os.Mkdir(filepath.Join(home, dir), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)
	logs := captureLog(t)
	cfg := bootstrap.Config{DBPath: filepath.Join(t.TempDir(), "ok.db"), ListenAddr: freeAddr(t), MCPAddr: freeAddr(t), ScanInterval: 20 * time.Millisecond}
	rt, err := bootstrap.Start(context.Background(), cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(logs.String(), "discovery: found") && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	rt.Close()
	for _, want := range []string{"startup: watching workspace", "discovery: found"} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("the log lacks %q:\n%s", want, logs.String())
		}
	}
}
