package main

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/bootstrap"
	"nexus-orchestrator/internal/core/domain"
)

// freeAddr returns a loopback address that was free a moment ago.
func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().String()
}

// syncBuffer is a goroutine-safe bytes.Buffer (the banner is written from a goroutine).
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// isolate points HOME and the working directory at temp dirs so the daemon
// never reads or watches the developer's real files.
func isolate(t *testing.T) (home string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	return home
}

func waitHealthy(t *testing.T, base string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(base + "/api/health")
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK && strings.Contains(string(body), `"status":"ok"`) {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("daemon did not become healthy")
}

func TestRun_StartsServesHealthAndShutsDownOnCancel(t *testing.T) {
	home := isolate(t)
	// Presence of these directories enables the Claude / Continue activity readers.
	for _, d := range []string{".claude", ".continue"} {
		if err := os.MkdirAll(filepath.Join(home, d), 0o750); err != nil {
			t.Fatal(err)
		}
	}

	cfg := bootstrap.Config{
		DBPath:       filepath.Join(t.TempDir(), "d.db"),
		ListenAddr:   freeAddr(t),
		MCPAddr:      freeAddr(t),
		ScanInterval: 20 * time.Millisecond, // exercises the periodic re-scan ticker
	}

	// Pre-seed persisted state the daemon must load at startup.
	seed, err := repo_sqlite.New(cfg.DBPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	pcr := repo_sqlite.NewProviderConfigRepo(seed)
	for _, pc := range []domain.ProviderConfig{
		{ID: "ok", Name: "seeded-ollama", Kind: domain.ProviderKind("ollama"), BaseURL: "http://127.0.0.1:1", Enabled: true},
		{ID: "off", Name: "disabled-one", Kind: domain.ProviderKind("ollama"), BaseURL: "http://127.0.0.1:1", Enabled: false},
		{ID: "bad", Name: "bad-kind", Kind: domain.ProviderKind("no-such-kind"), Enabled: true},
	} {
		if err := pcr.SaveProviderConfig(ctx, pc); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo_sqlite.NewRuntimeConfigRepo(seed).SaveRuntimeConfig(ctx, domain.RuntimeConfig{QueueCap: 5}); err != nil {
		t.Fatal(err)
	}
	seed.Close()

	runCtx, cancel := context.WithCancel(context.Background())
	var out syncBuffer
	done := make(chan error, 1)
	go func() { done <- run(runCtx, cfg, httpapi.NewLogHubWithWriter(io.Discard), &out) }()

	waitHealthy(t, "http://"+cfg.ListenAddr)
	time.Sleep(150 * time.Millisecond) // let the ticker fire at least once
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned %v, want nil", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("run did not return after cancel")
	}

	got := out.String()
	for _, want := range []string{"ready", "http://" + cfg.ListenAddr, "shutting down"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestRun_FailsWhenDatabaseCannotBeOpened(t *testing.T) {
	isolate(t)
	cfg := bootstrap.Config{
		DBPath:       filepath.Join(t.TempDir(), "missing", "dir", "d.db"),
		ListenAddr:   freeAddr(t),
		MCPAddr:      freeAddr(t),
		ScanInterval: time.Minute,
	}
	err := run(context.Background(), cfg, httpapi.NewLogHubWithWriter(io.Discard), io.Discard)
	if err == nil || !strings.Contains(err.Error(), "open database") {
		t.Fatalf("expected open-database error, got %v", err)
	}
}

func TestRunMain_ReturnsOneWhenStartupFails(t *testing.T) {
	isolate(t)
	t.Setenv("NEXUS_DB_PATH", filepath.Join(t.TempDir(), "missing", "dir", "d.db"))
	t.Setenv("NEXUS_LISTEN_ADDR", freeAddr(t))
	t.Setenv("NEXUS_MCP_ADDR", freeAddr(t))
	if code := runMain(); code != 1 {
		t.Errorf("exit code %d, want 1", code)
	}
}
