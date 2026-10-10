package bootstrap_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/bootstrap"
	"nexus-orchestrator/internal/core/domain"
)

func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().String()
}

func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
}

func TestConfigFromEnv(t *testing.T) {
	for _, k := range []string{"NEXUS_DB_PATH", "NEXUS_LISTEN_ADDR", "NEXUS_MCP_ADDR", "NEXUS_SCAN_INTERVAL"} {
		t.Setenv(k, "")
	}
	d := bootstrap.ConfigFromEnv()
	if d.DBPath != "nexus.db" || d.ListenAddr != "127.0.0.1:63987" || d.MCPAddr != "127.0.0.1:63988" || d.ScanInterval != 30*time.Second {
		t.Errorf("defaults: %+v", d)
	}
	t.Setenv("NEXUS_DB_PATH", "/tmp/x.db")
	t.Setenv("NEXUS_LISTEN_ADDR", "0.0.0.0:8080")
	t.Setenv("NEXUS_MCP_ADDR", "0.0.0.0:8081")
	t.Setenv("NEXUS_SCAN_INTERVAL", "15s")
	c := bootstrap.ConfigFromEnv()
	if c.DBPath != "/tmp/x.db" || c.ListenAddr != "0.0.0.0:8080" || c.MCPAddr != "0.0.0.0:8081" || c.ScanInterval != 15*time.Second {
		t.Errorf("custom: %+v", c)
	}
}

// time.NewTicker panics on a non-positive interval; these must be ignored.
func TestConfigFromEnv_IgnoresNonPositiveOrInvalidScanInterval(t *testing.T) {
	for _, v := range []string{"0s", "-5s", "garbage"} {
		t.Setenv("NEXUS_SCAN_INTERVAL", v)
		if got := bootstrap.ConfigFromEnv().ScanInterval; got != 30*time.Second {
			t.Errorf("NEXUS_SCAN_INTERVAL=%q: %v, want the 30s default", v, got)
		}
	}
}

func TestFormatBanner(t *testing.T) {
	b := bootstrap.FormatBanner("v1 — ready", "127.0.0.1:1", "127.0.0.1:2")
	for _, want := range []string{"v1 — ready", "http://127.0.0.1:1/ui", "http://127.0.0.1:1/api/howto", "http://127.0.0.1:2/mcp"} {
		if !strings.Contains(b, want) {
			t.Errorf("banner missing %q", want)
		}
	}
}

func TestStart_WiresServersLoadsPersistedProvidersAndClosesCleanly(t *testing.T) {
	isolate(t)
	cfg := bootstrap.Config{
		DBPath: filepath.Join(t.TempDir(), "rt.db"), ListenAddr: freeAddr(t), MCPAddr: freeAddr(t),
		ScanInterval: 20 * time.Millisecond,
	}
	// Pre-seed state Start must pick up.
	seed, err := repo_sqlite.New(cfg.DBPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	pcr := repo_sqlite.NewProviderConfigRepo(seed)
	for _, pc := range []domain.ProviderConfig{
		{ID: "a", Name: "seeded", Kind: domain.ProviderKindOllama, BaseURL: "http://127.0.0.1:1", Enabled: true},
		{ID: "b", Name: "off", Kind: domain.ProviderKindOllama, Enabled: false},
		{ID: "c", Name: "broken", Kind: domain.ProviderKindAnthropic, Enabled: true}, // no API key: rejected, logged
	} {
		if err := pcr.SaveProviderConfig(ctx, pc); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo_sqlite.NewRuntimeConfigRepo(seed).SaveRuntimeConfig(ctx, domain.RuntimeConfig{QueueCap: 5}); err != nil {
		t.Fatal(err)
	}
	seed.Close()

	rt, err := bootstrap.Start(ctx, cfg, httpapi.NewLogHubWithWriter(io.Discard))
	if err != nil {
		t.Fatal(err)
	}
	if rt.Orchestrator == nil || rt.Brain == nil || rt.Activity == nil {
		t.Fatalf("services not wired: %+v", rt)
	}

	deadline := time.Now().Add(10 * time.Second)
	for {
		resp, err := http.Get("http://" + cfg.ListenAddr + "/api/health")
		if err == nil {
			resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("REST API never came up")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if resp, err := http.Get("http://" + cfg.MCPAddr + "/health"); err != nil {
		t.Errorf("MCP server: %v", err)
	} else {
		resp.Body.Close()
	}
	providers, _ := rt.Orchestrator.GetProviders()
	names := map[string]bool{}
	for _, p := range providers {
		names[p.Name] = true
	}
	if !names["Ollama"] || len(providers) < 3 {
		t.Errorf("default plus persisted providers expected, got %v", names)
	}
	if rc, _ := rt.Orchestrator.GetRuntimeConfig(ctx); rc.QueueCap != 5 {
		t.Errorf("persisted queue cap not applied: %d", rc.QueueCap)
	}

	rt.Close()
	rt.Close() // idempotent
	if resp, err := http.Get("http://" + cfg.ListenAddr + "/api/health"); err == nil {
		resp.Body.Close()
		t.Error("REST API still answering after Close")
	}
}

func TestStart_StopsWhenTheParentContextIsCancelled(t *testing.T) {
	isolate(t)
	ctx, cancel := context.WithCancel(context.Background())
	cfg := bootstrap.Config{DBPath: filepath.Join(t.TempDir(), "p.db"), ListenAddr: freeAddr(t), MCPAddr: freeAddr(t)}
	rt, err := bootstrap.Start(ctx, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	done := make(chan struct{})
	go func() { rt.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("Close did not return after the parent context was cancelled")
	}
}

func TestStart_FailsWhenTheDatabaseCannotBeOpened(t *testing.T) {
	isolate(t)
	cfg := bootstrap.Config{DBPath: filepath.Join(t.TempDir(), "no", "such", "dir", "x.db")}
	rt, err := bootstrap.Start(context.Background(), cfg, nil)
	if err == nil || !strings.Contains(err.Error(), "open database") || rt != nil {
		t.Errorf("got %v %v", rt, err)
	}
}
