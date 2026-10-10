package main

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/bootstrap"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
	"nexus-orchestrator/internal/testutil/fakes"
)

var errBoom = errors.New("boom")

func TestApp_ForwardsEveryOrchestratorCall(t *testing.T) {
	o := fakes.NewOrchestrator()
	app := NewApp(o, "127.0.0.1:63987")

	if app.GetServerAddr() != "http://127.0.0.1:63987" {
		t.Errorf("GetServerAddr = %q", app.GetServerAddr())
	}
	if id, err := app.SubmitTask(domain.Task{}); err != nil || id != "task-1" {
		t.Errorf("SubmitTask: %q %v", id, err)
	}
	if got, err := app.GetTask("x"); err != nil || got.ID != "x" {
		t.Errorf("GetTask: %+v %v", got, err)
	}
	if q, err := app.GetQueue(); err != nil || len(q) != 1 {
		t.Errorf("GetQueue: %v %v", q, err)
	}
	if all, err := app.GetAllTasks(); err != nil || len(all) == 0 {
		t.Errorf("GetAllTasks: %v %v", all, err)
	}
	if p, err := app.GetProviders(); err != nil || len(p) != 1 {
		t.Errorf("GetProviders: %v %v", p, err)
	}
	if err := app.CancelTask("x"); err != nil {
		t.Error(err)
	}
	if err := app.RegisterCloudProvider(domain.ProviderConfig{}); err != nil {
		t.Error(err)
	}
	if err := app.RemoveProvider("x"); err != nil {
		t.Error(err)
	}
	if m, err := app.GetProviderModels("x"); err != nil || len(m) != 1 {
		t.Errorf("GetProviderModels: %v %v", m, err)
	}
	if _, err := app.AddProviderConfig(domain.ProviderConfig{Name: "n"}); err != nil {
		t.Error(err)
	}
	if l, err := app.ListProviderConfigs(); err != nil || len(l) != 2 {
		t.Errorf("ListProviderConfigs: %v %v", l, err)
	}
	if _, err := app.UpdateProviderConfig(domain.ProviderConfig{ID: "pc"}); err != nil {
		t.Error(err)
	}
	if err := app.RemoveProviderConfig("pc"); err != nil {
		t.Error(err)
	}
	if d, err := app.GetDiscoveredProviders(); err != nil || len(d) != 1 {
		t.Errorf("GetDiscoveredProviders: %v %v", d, err)
	}
	if err := app.TriggerScan(); err != nil {
		t.Error(err)
	}
	if id, err := app.CreateDraft(domain.Task{}); err != nil || id != "draft-1" {
		t.Errorf("CreateDraft: %q %v", id, err)
	}
	if b, err := app.GetBacklog("/p"); err != nil || len(b) != 1 {
		t.Errorf("GetBacklog: %v %v", b, err)
	}
	if r, err := app.PromoteTask("x"); err != nil || !r.Promoted {
		t.Errorf("PromoteTask: %+v %v", r, err)
	}
	if u, err := app.UpdateTask("x", domain.Task{}); err != nil || u.ID != "x" {
		t.Errorf("UpdateTask: %+v %v", u, err)
	}
	if l, err := app.ListAISessions(); err != nil || len(l) != 1 {
		t.Errorf("ListAISessions: %v %v", l, err)
	}
	if s, err := app.RegisterAISession(domain.AISession{AgentName: "a"}); err != nil || s.ID != "s1" {
		t.Errorf("RegisterAISession: %+v %v", s, err)
	}
	if err := app.DeregisterAISession("s1"); err != nil {
		t.Error(err)
	}
	if err := app.HeartbeatAISession("s1"); err != nil {
		t.Error(err)
	}
	if n, err := app.PurgeDisconnectedSessions(); err != nil || n != 3 {
		t.Errorf("Purge: %d %v", n, err)
	}
	if c, err := app.ClaimTask("t", "s"); err != nil || c.AISessionID != "s" {
		t.Errorf("ClaimTask: %+v %v", c, err)
	}
	if u, err := app.UpdateTaskStatus("t", "s", "COMPLETED", "logs"); err != nil || u.Status != domain.StatusCompleted {
		t.Errorf("UpdateTaskStatus: %+v %v", u, err)
	}
	if cfg, err := app.GetRuntimeConfig(); err != nil || cfg.QueueCap != 50 {
		t.Errorf("GetRuntimeConfig: %+v %v", cfg, err)
	}
	cap7 := 7
	if cfg, err := app.UpdateRuntimeConfig(domain.RuntimeConfigUpdate{QueueCap: &cap7}); err != nil || cfg.QueueCap != 7 {
		t.Errorf("UpdateRuntimeConfig: %+v %v", cfg, err)
	}
}

func TestApp_PropagatesOrchestratorFailures(t *testing.T) {
	o := fakes.NewOrchestrator()
	for _, op := range []string{"SubmitTask", "GetTask", "GetQueue", "GetAllTasks", "GetProviders", "CancelTask",
		"RegisterCloudProvider", "RemoveProvider", "GetProviderModels", "AddProviderConfig", "ListProviderConfigs",
		"UpdateProviderConfig", "RemoveProviderConfig", "GetDiscoveredProviders", "TriggerScan", "CreateDraft",
		"GetBacklog", "PromoteTask", "UpdateTask", "ListAISessions", "RegisterAISession", "DeregisterAISession",
		"HeartbeatAISession", "PurgeDisconnectedSessions", "ClaimTask", "UpdateTaskStatus", "GetRuntimeConfig",
		"UpdateRuntimeConfig"} {
		o.FailWith(op, errBoom)
	}
	app := NewApp(o, "x")
	checks := map[string]func() error{
		"SubmitTask":                func() error { _, e := app.SubmitTask(domain.Task{}); return e },
		"GetTask":                   func() error { _, e := app.GetTask("x"); return e },
		"GetQueue":                  func() error { _, e := app.GetQueue(); return e },
		"GetAllTasks":               func() error { _, e := app.GetAllTasks(); return e },
		"GetProviders":              func() error { _, e := app.GetProviders(); return e },
		"CancelTask":                func() error { return app.CancelTask("x") },
		"RegisterCloudProvider":     func() error { return app.RegisterCloudProvider(domain.ProviderConfig{}) },
		"RemoveProvider":            func() error { return app.RemoveProvider("x") },
		"GetProviderModels":         func() error { _, e := app.GetProviderModels("x"); return e },
		"AddProviderConfig":         func() error { _, e := app.AddProviderConfig(domain.ProviderConfig{}); return e },
		"ListProviderConfigs":       func() error { _, e := app.ListProviderConfigs(); return e },
		"UpdateProviderConfig":      func() error { _, e := app.UpdateProviderConfig(domain.ProviderConfig{}); return e },
		"RemoveProviderConfig":      func() error { return app.RemoveProviderConfig("x") },
		"GetDiscoveredProviders":    func() error { _, e := app.GetDiscoveredProviders(); return e },
		"TriggerScan":               func() error { return app.TriggerScan() },
		"CreateDraft":               func() error { _, e := app.CreateDraft(domain.Task{}); return e },
		"GetBacklog":                func() error { _, e := app.GetBacklog("p"); return e },
		"PromoteTask":               func() error { _, e := app.PromoteTask("x"); return e },
		"UpdateTask":                func() error { _, e := app.UpdateTask("x", domain.Task{}); return e },
		"ListAISessions":            func() error { _, e := app.ListAISessions(); return e },
		"RegisterAISession":         func() error { _, e := app.RegisterAISession(domain.AISession{}); return e },
		"DeregisterAISession":       func() error { return app.DeregisterAISession("x") },
		"HeartbeatAISession":        func() error { return app.HeartbeatAISession("x") },
		"PurgeDisconnectedSessions": func() error { _, e := app.PurgeDisconnectedSessions(); return e },
		"ClaimTask":                 func() error { _, e := app.ClaimTask("t", "s"); return e },
		"UpdateTaskStatus":          func() error { _, e := app.UpdateTaskStatus("t", "s", "FAILED", ""); return e },
		"GetRuntimeConfig":          func() error { _, e := app.GetRuntimeConfig(); return e },
		"UpdateRuntimeConfig":       func() error { _, e := app.UpdateRuntimeConfig(domain.RuntimeConfigUpdate{}); return e },
	}
	for name, call := range checks {
		if err := call(); !errors.Is(err, errBoom) {
			t.Errorf("%s: want the orchestrator error, got %v", name, err)
		}
	}
}

func TestApp_BrainBindings(t *testing.T) {
	b := fakes.NewBrain()
	app := NewApp(fakes.NewOrchestrator(), "x").withBrainService(b)
	if n, err := app.IngestKnowledge("/p", "/f.md"); err != nil || n != 2 {
		t.Errorf("IngestKnowledge: %d %v", n, err)
	}
	if s, err := app.GetBrainStatus("/p"); err != nil || !s.Initialized {
		t.Errorf("GetBrainStatus: %+v %v", s, err)
	}
	if c, err := app.GetProjectContext("/p", 100); err != nil || c.TokensUsed != 5 {
		t.Errorf("GetProjectContext: %+v %v", c, err)
	}
	if c, err := app.GetFocusedContext("/p", "q", 100); err != nil || c.TokensUsed != 5 {
		t.Errorf("GetFocusedContext: %+v %v", c, err)
	}
	if r, err := app.SearchKnowledge("/p", "q", 3); err != nil || len(r) != 1 {
		t.Errorf("SearchKnowledge: %v %v", r, err)
	}
	if s, err := app.InitProject("/p", ""); err != nil || !s.Initialized {
		t.Errorf("InitProject: %+v %v", s, err)
	}
	if l, err := app.ListKnowledge("/p", ""); err != nil || len(l) != 1 {
		t.Errorf("ListKnowledge: %v %v", l, err)
	}
	if err := app.DeleteKnowledge("k"); err != nil {
		t.Error(err)
	}
	if f, err := app.GetFileMap("/p", ""); err != nil || len(f) != 1 {
		t.Errorf("GetFileMap: %v %v", f, err)
	}

	// Every brain call reports a missing service instead of panicking.
	none := NewApp(fakes.NewOrchestrator(), "x")
	calls := map[string]func() error{
		"IngestKnowledge":     func() error { _, e := none.IngestKnowledge("p", "f"); return e },
		"GetBrainStatus":      func() error { _, e := none.GetBrainStatus("p"); return e },
		"GetProjectContext":   func() error { _, e := none.GetProjectContext("p", 1); return e },
		"GetFocusedContext":   func() error { _, e := none.GetFocusedContext("p", "q", 1); return e },
		"SearchKnowledge":     func() error { _, e := none.SearchKnowledge("p", "q", 1); return e },
		"InitProject":         func() error { _, e := none.InitProject("p", ""); return e },
		"ListKnowledge":       func() error { _, e := none.ListKnowledge("p", ""); return e },
		"DeleteKnowledge":     func() error { return none.DeleteKnowledge("k") },
		"GetFileMap":          func() error { _, e := none.GetFileMap("p", ""); return e },
		"WatchWorkspace":      func() error { return none.WatchWorkspace("p") },
		"UnwatchWorkspace":    func() error { return none.UnwatchWorkspace("p") },
		"GetRecentActivities": func() error { _, e := none.GetRecentActivities("", "", "", "", 0); return e },
		"GetActivityTimeline": func() error { _, e := none.GetActivityTimeline("", 0); return e },
	}
	for name, call := range calls {
		if err := call(); err == nil || !strings.Contains(err.Error(), "not available") {
			t.Errorf("%s without its service: %v", name, err)
		}
	}
}

func TestApp_ActivityBindingsParseTheirArguments(t *testing.T) {
	repo, err := repo_sqlite.New(filepath.Join(t.TempDir(), "a.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	ar := repo_sqlite.NewActivityRepo(repo)
	now := time.Now().UTC()
	for _, a := range []domain.AIActivity{
		{ID: "1", AgentName: "claude", ProjectPath: "/p", ActivityType: domain.ActivityTypeMessage, Timestamp: now.Add(-time.Hour)},
		{ID: "2", AgentName: "copilot", ProjectPath: "/p", ActivityType: domain.ActivityTypeToolUse, Timestamp: now.Add(-48 * time.Hour)},
	} {
		if err := ar.SaveActivity(context.Background(), a); err != nil {
			t.Fatal(err)
		}
	}
	app := NewApp(fakes.NewOrchestrator(), "x").
		withActivityService(services.NewActivityService(ar, repo_sqlite.NewAISessionRepo(repo)))

	if got, err := app.GetRecentActivities("claude", "/p", "message", "", 10); err != nil || len(got) != 1 {
		t.Errorf("filtered: %v %v", got, err)
	}
	since := now.Add(-72 * time.Hour).Format(time.RFC3339)
	if got, err := app.GetRecentActivities("", "", "", since, 0); err != nil || len(got) != 2 {
		t.Errorf("since: %v %v", got, err)
	}
	if got, err := app.GetRecentActivities("", "", "", "not-a-date", 0); err != nil || len(got) != 2 {
		t.Errorf("an unparsable since is ignored: %v %v", got, err)
	}
	if got, err := app.GetActivityTimeline("", 0); err != nil || len(got) != 1 {
		t.Errorf("the default window is 24h: %v %v", got, err)
	}
	if got, err := app.GetActivityTimeline(since, 5); err != nil || len(got) != 2 {
		t.Errorf("explicit window: %v %v", got, err)
	}
}

func TestApp_WatcherBindings(t *testing.T) {
	app := NewApp(fakes.NewOrchestrator(), "x")
	rt, err := bootstrap.Start(context.Background(), bootstrap.Config{
		DBPath: filepath.Join(t.TempDir(), "w.db"), ListenAddr: "127.0.0.1:0", MCPAddr: "127.0.0.1:0",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()
	if rt.Watcher == nil {
		t.Skip("filesystem watcher unavailable on this platform")
	}
	app.withFsWatcher(rt.Watcher)
	dir := t.TempDir()
	if err := app.WatchWorkspace(dir); err != nil {
		t.Error(err)
	}
	if err := app.UnwatchWorkspace(dir); err != nil {
		t.Error(err)
	}
}

func TestNewAppAndWindowOptions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	rt, err := bootstrap.Start(context.Background(), bootstrap.Config{
		DBPath: filepath.Join(t.TempDir(), "n.db"), ListenAddr: "127.0.0.1:0", MCPAddr: "127.0.0.1:0",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()
	app := newApp(rt)
	if app.GetServerAddr() != "http://127.0.0.1:0" || app.brainSvc == nil || app.activitySvc == nil {
		t.Errorf("newApp must wire the runtime's services: %+v", app)
	}

	frontend := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<html></html>")}}
	opts := windowOptions(app, frontend)
	if opts.Title != "nexusOrchestrator" || opts.Width != 1024 || opts.Height != 768 || !opts.HideWindowOnClose {
		t.Errorf("window options: %+v", opts)
	}
	if len(opts.Bind) != 1 || opts.Bind[0] != app || opts.AssetServer == nil {
		t.Errorf("binding/asset server: %+v", opts)
	}
	_ = io.Discard
}
