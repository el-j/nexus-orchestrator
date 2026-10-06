package services_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

// eventually polls cond every few ms until it holds or the deadline passes.
func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func (b *capturingBroadcaster) taskEvents() []ports.TaskEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]ports.TaskEvent(nil), b.events...)
}

func TestWatchdog_FailsStaleProcessingTaskAndBroadcastsOnce(t *testing.T) {
	repo := newFaultyRepo()
	repo.setStale(domain.Task{ID: "stuck", ProjectPath: "/p", Status: domain.StatusProcessing, Logs: "started"})
	b := &capturingBroadcaster{}
	orch := services.NewOrchestrator(services.NewDiscoveryService(), repo, &noopWriter{}, nil,
		services.WithWatchdogInterval(10*time.Millisecond), services.WithCleanupInterval(time.Hour))
	orch.SetBroadcaster(b)
	defer orch.Stop()

	eventually(t, "watchdog to fail the stuck task", func() bool {
		got, err := repo.memRepo.GetByID("stuck")
		return err == nil && got.Status == domain.StatusFailed
	})
	got, _ := repo.memRepo.GetByID("stuck")
	if want := "Marked as FAILED by watchdog"; !contains(got.Logs, want) || !contains(got.Logs, "started") {
		t.Errorf("logs should keep history and add the watchdog note, got %q", got.Logs)
	}

	time.Sleep(60 * time.Millisecond) // let any duplicate broadcast arrive
	failed := 0
	for _, ev := range b.taskEvents() {
		if ev.TaskID == "stuck" && ev.Type == ports.EventTaskFailed {
			failed++
		}
	}
	if failed != 1 {
		t.Errorf("want exactly one task.failed event for the watchdog-failed task, got %d", failed)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestWatchdog_SurvivesRepositoryErrors(t *testing.T) {
	repo := newFaultyRepo("GetStaleProcessing")
	orch := services.NewOrchestrator(services.NewDiscoveryService(), repo, &noopWriter{}, nil,
		services.WithWatchdogInterval(5*time.Millisecond), services.WithCleanupInterval(time.Hour))
	time.Sleep(40 * time.Millisecond) // several ticks that all fail
	repo.set("GetStaleProcessing", false)
	repo.setStale(domain.Task{ID: "t", Status: domain.StatusProcessing})
	repo.set("Update", true) // update failures must not kill the loop either
	time.Sleep(40 * time.Millisecond)
	repo.set("Update", false)
	repo.setStale(domain.Task{ID: "t2", Status: domain.StatusProcessing})
	eventually(t, "watchdog to recover after errors", func() bool {
		got, err := repo.memRepo.GetByID("t2")
		return err == nil && got.Status == domain.StatusFailed
	})
	orch.Stop()
}

func TestSessionCleanup_MarksStaleSessionsDisconnectedAndPurges(t *testing.T) {
	aiRepo := newFaultyAIRepo()
	b := &capturingBroadcaster{}
	orch := services.NewOrchestrator(services.NewDiscoveryService(), newMemRepo(), &noopWriter{}, nil,
		services.WithCleanupInterval(10*time.Millisecond), services.WithStaleThreshold(50*time.Millisecond),
		services.WithWatchdogInterval(time.Hour))
	orch.SetAISessionRepo(aiRepo)
	orch.SetBroadcaster(b)
	defer orch.Stop()

	old := time.Now().Add(-time.Hour)
	for _, s := range []domain.AISession{
		{ID: "stale", Status: domain.SessionStatusActive, LastActivity: old},
		{ID: "fresh", Status: domain.SessionStatusActive, LastActivity: time.Now().Add(time.Hour)},
		{ID: "ancient", Status: domain.SessionStatusDisconnected, LastActivity: time.Now().Add(-3 * time.Hour)},
	} {
		if err := aiRepo.memAISessionRepo.SaveAISession(bg, s); err != nil {
			t.Fatal(err)
		}
	}

	eventually(t, "stale session to be disconnected", func() bool {
		got, err := aiRepo.GetAISessionByID(bg, "stale")
		return err == nil && got.Status == domain.SessionStatusDisconnected
	})
	if got, _ := aiRepo.GetAISessionByID(bg, "fresh"); got.Status != domain.SessionStatusActive {
		t.Errorf("fresh session must stay active, got %s", got.Status)
	}
	eventually(t, "ancient disconnected session to be purged", func() bool {
		_, err := aiRepo.GetAISessionByID(bg, "ancient")
		return errors.Is(err, domain.ErrNotFound)
	})
	found := false
	for _, ev := range b.sessionSnapshot() {
		if ev.AISessionID == "stale" && ev.Status == domain.SessionStatusDisconnected {
			found = true
		}
	}
	if !found {
		t.Error("expected a disconnect broadcast for the stale session")
	}
}

func TestSessionCleanup_ToleratesRepositoryErrors(t *testing.T) {
	aiRepo := newFaultyAIRepo()
	orch := services.NewOrchestrator(services.NewDiscoveryService(), newMemRepo(), &noopWriter{}, nil,
		services.WithCleanupInterval(5*time.Millisecond), services.WithStaleThreshold(time.Millisecond),
		services.WithWatchdogInterval(time.Hour))
	defer orch.Stop()

	// No repo yet: the loop must just skip.
	time.Sleep(20 * time.Millisecond)
	orch.SetAISessionRepo(aiRepo)

	aiRepo.set("List", errInjected)
	time.Sleep(20 * time.Millisecond)
	aiRepo.set("List", nil)

	_ = aiRepo.memAISessionRepo.SaveAISession(bg, domain.AISession{ID: "s", Status: domain.SessionStatusActive, LastActivity: time.Now().Add(-time.Hour)})
	aiRepo.set("UpdateStatus", errInjected)
	aiRepo.set("Purge", errInjected)
	time.Sleep(30 * time.Millisecond)
	aiRepo.set("UpdateStatus", nil)
	aiRepo.set("Purge", nil)
	eventually(t, "cleanup to recover after repository errors", func() bool {
		got, err := aiRepo.GetAISessionByID(bg, "s")
		return err == nil && got.Status == domain.SessionStatusDisconnected
	})
}

func TestRecoverStuckTasks_RequeuesProcessingTasksOnStartup(t *testing.T) {
	repo := newFaultyRepo()
	seedTask(t, repo, "was-processing", domain.StatusProcessing)
	seedTask(t, repo, "already-queued", domain.StatusQueued)
	orch := services.NewOrchestrator(services.NewDiscoveryService(), repo, &noopWriter{}, nil, services.WithDisableBackgroundWorkers())
	defer orch.Stop()
	if got, _ := repo.GetByID("was-processing"); got.Status != domain.StatusQueued {
		t.Errorf("status = %s, want QUEUED", got.Status)
	}

	// Errors during recovery are logged, never fatal.
	bad := newFaultyRepo("GetPending")
	services.NewOrchestrator(services.NewDiscoveryService(), bad, &noopWriter{}, nil, services.WithDisableBackgroundWorkers()).Stop()
	bad2 := newFaultyRepo("UpdateStatusIfCurrent")
	seedTask(t, bad2, "p", domain.StatusProcessing)
	services.NewOrchestrator(services.NewDiscoveryService(), bad2, &noopWriter{}, nil, services.WithDisableBackgroundWorkers()).Stop()
	if got, _ := bad2.GetByID("p"); got.Status != domain.StatusProcessing {
		t.Errorf("failed re-queue must leave the task untouched, got %s", got.Status)
	}
}

// memRuntimeRepo is an in-memory ports.RuntimeConfigRepository.
type memRuntimeRepo struct {
	mu      sync.Mutex
	cfg     *domain.RuntimeConfig
	getErr  error
	saveErr error
}

func (m *memRuntimeRepo) GetRuntimeConfig(context.Context) (domain.RuntimeConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getErr != nil {
		return domain.RuntimeConfig{}, m.getErr
	}
	if m.cfg == nil {
		return domain.RuntimeConfig{}, domain.ErrNotFound
	}
	return *m.cfg, nil
}

func (m *memRuntimeRepo) SaveRuntimeConfig(_ context.Context, c domain.RuntimeConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.saveErr != nil {
		return m.saveErr
	}
	m.cfg = &c
	return nil
}

func newRuntimeOrch(t *testing.T, repo ports.RuntimeConfigRepository) *services.OrchestratorService {
	t.Helper()
	orch := services.NewOrchestrator(services.NewDiscoveryService(), newMemRepo(), &noopWriter{}, nil, services.WithDisableBackgroundWorkers())
	t.Cleanup(orch.Stop)
	if repo != nil {
		orch.WithRuntimeConfigRepo(repo)
	}
	return orch
}

func TestGetRuntimeConfig_PrecedenceEnvThenStoredThenDefault(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	t.Setenv("NEXUS_MCP_TOKEN", "")

	// No repo: default cap 50, no tokens.
	got, err := newRuntimeOrch(t, nil).GetRuntimeConfig(bg)
	if err != nil || got.QueueCap != 50 || got.APIToken != "" {
		t.Fatalf("defaults: %+v %v", got, err)
	}

	// Repo with no stored row (ErrNotFound) behaves like defaults.
	if got, err := newRuntimeOrch(t, &memRuntimeRepo{}).GetRuntimeConfig(bg); err != nil || got.QueueCap != 50 {
		t.Fatalf("not found: %+v %v", got, err)
	}

	stored := &memRuntimeRepo{cfg: &domain.RuntimeConfig{QueueCap: 7, APIToken: "stored-api", MCPToken: "stored-mcp", UpdatedAt: time.Unix(100, 0)}}
	got, err = newRuntimeOrch(t, stored).GetRuntimeConfig(bg)
	if err != nil || got.QueueCap != 7 || got.APIToken != "stored-api" || got.MCPToken != "stored-mcp" || !got.UpdatedAt.Equal(time.Unix(100, 0)) {
		t.Fatalf("stored values: %+v %v", got, err)
	}

	// Environment tokens win over stored ones.
	t.Setenv("NEXUS_API_TOKEN", "env-api")
	t.Setenv("NEXUS_MCP_TOKEN", "env-mcp")
	got, _ = newRuntimeOrch(t, stored).GetRuntimeConfig(bg)
	if got.APIToken != "env-api" || got.MCPToken != "env-mcp" {
		t.Errorf("env must win: %+v", got)
	}

	if _, err := newRuntimeOrch(t, &memRuntimeRepo{getErr: errInjected}).GetRuntimeConfig(bg); !errors.Is(err, errInjected) {
		t.Errorf("repo error: %v", err)
	}
}

func TestUpdateRuntimeConfig_AppliesAndPersistsChanges(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	t.Setenv("NEXUS_MCP_TOKEN", "")
	repo := &memRuntimeRepo{}
	orch := newRuntimeOrch(t, repo)

	cap10, api, mcp := 10, "api-set", "mcp-set"
	got, err := orch.UpdateRuntimeConfig(bg, domain.RuntimeConfigUpdate{QueueCap: &cap10, APIToken: &api, MCPToken: &mcp})
	if err != nil || got.QueueCap != 10 || got.APIToken != "api-set" || got.MCPToken != "mcp-set" || got.UpdatedAt.IsZero() {
		t.Fatalf("explicit values: %+v %v", got, err)
	}
	if repo.cfg == nil || repo.cfg.QueueCap != 10 {
		t.Errorf("not persisted: %+v", repo.cfg)
	}
	if eff, _ := orch.GetRuntimeConfig(bg); eff.QueueCap != 10 {
		t.Errorf("queue cap not applied in memory: %+v", eff)
	}

	rotated, err := orch.UpdateRuntimeConfig(bg, domain.RuntimeConfigUpdate{RotateAPIToken: true, RotateMCPToken: true})
	if err != nil || len(rotated.APIToken) < 40 || len(rotated.MCPToken) < 40 ||
		rotated.APIToken == "api-set" || rotated.MCPToken == "mcp-set" || rotated.APIToken == rotated.MCPToken {
		t.Fatalf("rotation: %+v %v", rotated, err)
	}
}

func TestUpdateRuntimeConfig_RejectsInvalidCapAndPropagatesErrors(t *testing.T) {
	zero := 0
	if _, err := newRuntimeOrch(t, &memRuntimeRepo{}).UpdateRuntimeConfig(bg, domain.RuntimeConfigUpdate{QueueCap: &zero}); err == nil {
		t.Error("queueCap=0 must be rejected")
	}
	if _, err := newRuntimeOrch(t, &memRuntimeRepo{getErr: errInjected}).UpdateRuntimeConfig(bg, domain.RuntimeConfigUpdate{}); !errors.Is(err, errInjected) {
		t.Errorf("get failure: %v", err)
	}
}

func TestUpdateRuntimeConfig_FailedSaveDoesNotChangeInMemoryCap(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	repo := &memRuntimeRepo{saveErr: errInjected}
	orch := newRuntimeOrch(t, repo)
	cap3 := 3
	if _, err := orch.UpdateRuntimeConfig(bg, domain.RuntimeConfigUpdate{QueueCap: &cap3}); !errors.Is(err, errInjected) {
		t.Fatalf("got %v", err)
	}
	if eff, _ := orch.GetRuntimeConfig(bg); eff.QueueCap != 50 {
		t.Errorf("a failed save must not change the running cap; got %d", eff.QueueCap)
	}
}

func TestQueueCapOptionIsEnforcedAtSubmission(t *testing.T) {
	orch := services.NewOrchestrator(services.NewDiscoveryService(), newMemRepo(), &noopWriter{}, nil, services.WithDisableBackgroundWorkers())
	defer orch.Stop()
	orch.WithQueueCap(1)
	if _, err := orch.SubmitTask(domain.Task{Instruction: "a", ProjectPath: "/p"}); err != nil {
		t.Fatal(err)
	}
	if _, err := orch.SubmitTask(domain.Task{Instruction: "b", ProjectPath: "/p"}); !errors.Is(err, services.ErrQueueFull) {
		t.Errorf("want ErrQueueFull, got %v", err)
	}
}
