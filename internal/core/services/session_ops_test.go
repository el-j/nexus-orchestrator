package services_test

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

// faultyAIRepo wraps memAISessionRepo with injectable failures per method.
type faultyAIRepo struct {
	*memAISessionRepo
	mu   sync.Mutex
	fail map[string]error
}

func newFaultyAIRepo() *faultyAIRepo {
	return &faultyAIRepo{memAISessionRepo: newMemAISessionRepo(), fail: map[string]error{}}
}

func (f *faultyAIRepo) set(method string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fail[method] = err
}

func (f *faultyAIRepo) err(method string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.fail[method]
}

func (f *faultyAIRepo) SaveAISession(ctx context.Context, s domain.AISession) error {
	if err := f.err("Save"); err != nil {
		return err
	}
	return f.memAISessionRepo.SaveAISession(ctx, s)
}

func (f *faultyAIRepo) ListAISessions(ctx context.Context) ([]domain.AISession, error) {
	if err := f.err("List"); err != nil {
		return nil, err
	}
	return f.memAISessionRepo.ListAISessions(ctx)
}

func (f *faultyAIRepo) UpdateAISessionStatus(ctx context.Context, id string, st domain.AISessionStatus, at time.Time) error {
	if err := f.err("UpdateStatus"); err != nil {
		return err
	}
	return f.memAISessionRepo.UpdateAISessionStatus(ctx, id, st, at)
}

func (f *faultyAIRepo) PurgeDisconnected(ctx context.Context, d time.Duration) (int, error) {
	if err := f.err("Purge"); err != nil {
		return 0, err
	}
	return f.memAISessionRepo.PurgeDisconnected(ctx, d)
}

func newSessionOrch(t *testing.T, opts ...services.Option) (*services.OrchestratorService, *faultyAIRepo, *capturingBroadcaster) {
	t.Helper()
	repo := newFaultyAIRepo()
	b := &capturingBroadcaster{}
	opts = append([]services.Option{services.WithDisableBackgroundWorkers()}, opts...)
	orch := services.NewOrchestrator(services.NewDiscoveryService(), newMemRepo(), &noopWriter{}, nil, opts...)
	orch.SetAISessionRepo(repo)
	orch.SetBroadcaster(b)
	t.Cleanup(orch.Stop)
	return orch, repo, b
}

func TestSessionOps_NoRepoConfigured(t *testing.T) {
	orch := services.NewOrchestrator(services.NewDiscoveryService(), newMemRepo(), &noopWriter{}, nil, services.WithDisableBackgroundWorkers())
	defer orch.Stop()
	if _, err := orch.RegisterAISession(bg, domain.AISession{}); err == nil {
		t.Error("Register: expected error")
	}
	if got, err := orch.ListAISessions(bg); err != nil || got == nil || len(got) != 0 {
		t.Errorf("List must be an empty non-nil slice: %v %v", got, err)
	}
	if err := orch.DeregisterAISession(bg, "x"); err == nil {
		t.Error("Deregister: expected error")
	}
	if err := orch.TerminateAISession(bg, "x", false); err == nil {
		t.Error("Terminate: expected error")
	}
	if err := orch.HeartbeatAISession(bg, "x"); err == nil {
		t.Error("Heartbeat: expected error")
	}
	if _, err := orch.PurgeDisconnectedSessions(bg); err == nil {
		t.Error("Purge: expected error")
	}
	if _, err := orch.DelegateToNexus(bg, "x"); err == nil {
		t.Error("Delegate: expected error")
	}
}

func TestRegisterAISession_IdempotentByExternalID(t *testing.T) {
	orch, repo, b := newSessionOrch(t)

	first, err := orch.RegisterAISession(bg, domain.AISession{ExternalID: "ext-1", AgentName: "claude"})
	if err != nil || first.ID == "" || first.Status != domain.SessionStatusActive {
		t.Fatalf("first: %+v %v", first, err)
	}
	// Make it look stale, then re-register: it must be reused and reactivated.
	_ = repo.memAISessionRepo.UpdateAISessionStatus(bg, first.ID, domain.SessionStatusDisconnected, time.Now().Add(-time.Hour))
	second, err := orch.RegisterAISession(bg, domain.AISession{ExternalID: "ext-1"})
	if err != nil || second.ID != first.ID || second.Status != domain.SessionStatusActive {
		t.Fatalf("second: %+v %v", second, err)
	}
	if all, _ := orch.ListAISessions(bg); len(all) != 1 {
		t.Errorf("duplicate rows created: %d", len(all))
	}
	if len(b.sessionSnapshot()) != 2 {
		t.Errorf("each registration should broadcast once, got %d", len(b.sessionSnapshot()))
	}

	repo.set("Save", errInjected)
	if _, err := orch.RegisterAISession(bg, domain.AISession{ExternalID: "ext-1"}); !errors.Is(err, errInjected) {
		t.Errorf("refresh save failure: %v", err)
	}
	if _, err := orch.RegisterAISession(bg, domain.AISession{ExternalID: "new"}); !errors.Is(err, errInjected) {
		t.Errorf("create save failure: %v", err)
	}
}

func TestSessionOps_RepoErrorsAreWrapped(t *testing.T) {
	orch, repo, _ := newSessionOrch(t)
	repo.set("List", errInjected)
	if _, err := orch.ListAISessions(bg); !errors.Is(err, errInjected) {
		t.Errorf("List: %v", err)
	}
	repo.set("UpdateStatus", errInjected)
	if err := orch.DeregisterAISession(bg, "x"); !errors.Is(err, errInjected) {
		t.Errorf("Deregister: %v", err)
	}
	if err := orch.HeartbeatAISession(bg, "x"); !errors.Is(err, errInjected) {
		t.Errorf("Heartbeat: %v", err)
	}
	repo.set("Purge", errInjected)
	if _, err := orch.PurgeDisconnectedSessions(bg); !errors.Is(err, errInjected) {
		t.Errorf("Purge: %v", err)
	}

	seedActiveSession(t, repo.memAISessionRepo, "s")
	if err := orch.TerminateAISession(bg, "s", false); !errors.Is(err, errInjected) {
		t.Errorf("Terminate (status update): %v", err)
	}
	if err := orch.TerminateAISession(bg, "ghost", false); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Terminate unknown: %v", err)
	}
}

func TestDeregisterAndHeartbeat_BroadcastAndRefresh(t *testing.T) {
	orch, repo, b := newSessionOrch(t)
	seedActiveSession(t, repo.memAISessionRepo, "s")

	if err := orch.DeregisterAISession(bg, "s"); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.GetAISessionByID(bg, "s"); got.Status != domain.SessionStatusDisconnected {
		t.Errorf("status = %s", got.Status)
	}
	if evs := b.sessionSnapshot(); len(evs) != 1 || evs[0].Status != domain.SessionStatusDisconnected {
		t.Errorf("events = %+v", evs)
	}
	if err := orch.HeartbeatAISession(bg, "s"); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.GetAISessionByID(bg, "s"); got.Status != domain.SessionStatusActive {
		t.Errorf("heartbeat should reactivate, status = %s", got.Status)
	}
}

func TestTerminateAISession_SignalsOrKillsProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("signal semantics differ on Windows")
	}
	for _, force := range []bool{false, true} {
		cmd := exec.Command("sleep", "30")
		if err := cmd.Start(); err != nil {
			t.Skipf("cannot start helper process: %v", err)
		}
		exited := make(chan error, 1)
		go func() { exited <- cmd.Wait() }()

		orch, repo, b := newSessionOrch(t)
		if err := repo.memAISessionRepo.SaveAISession(bg, domain.AISession{
			ID: "s", Status: domain.SessionStatusActive, PID: cmd.Process.Pid,
		}); err != nil {
			t.Fatal(err)
		}
		if err := orch.TerminateAISession(bg, "s", force); err != nil {
			t.Fatal(err)
		}
		select {
		case <-exited:
		case <-time.After(5 * time.Second):
			_ = cmd.Process.Kill()
			t.Fatalf("force=%v: process was not terminated", force)
		}
		if evs := b.sessionSnapshot(); len(evs) != 1 || evs[0].Type != "ai_session_terminated" {
			t.Errorf("force=%v: events = %+v", force, evs)
		}
	}
}

// stubAgentScanner / stubAgentStore / stubPlanRepo are minimal scripted doubles.
type stubAgentScanner struct {
	mu        sync.Mutex
	agents    []domain.DiscoveredAgent
	agentsErr error
	plans     []domain.DiscoveredPlanFile
	plansErr  error
	scans     int
}

func (s *stubAgentScanner) ScanAgents(context.Context) ([]domain.DiscoveredAgent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scans++
	return s.agents, s.agentsErr
}

func (s *stubAgentScanner) ScanPlanFiles(context.Context, []string) ([]domain.DiscoveredPlanFile, error) {
	return s.plans, s.plansErr
}

type stubAgentStore struct {
	upserted  []domain.DiscoveredAgent
	upsertErr error
	list      []domain.DiscoveredAgent
	listErr   error
}

func (s *stubAgentStore) UpsertDiscoveredAgent(_ context.Context, a domain.DiscoveredAgent) error {
	s.upserted = append(s.upserted, a)
	return s.upsertErr
}

func (s *stubAgentStore) ListDiscoveredAgents(context.Context) ([]domain.DiscoveredAgent, error) {
	return s.list, s.listErr
}

type stubPlanRepo struct {
	stored    []domain.DiscoveredPlanFile
	upsertErr error
	listErr   error
}

func (s *stubPlanRepo) UpsertPlanFile(_ context.Context, f domain.DiscoveredPlanFile) error {
	s.stored = append(s.stored, f)
	return s.upsertErr
}

func (s *stubPlanRepo) ListPlanFiles(_ context.Context, project string) ([]domain.DiscoveredPlanFile, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	var out []domain.DiscoveredPlanFile
	for _, f := range s.stored {
		if project == "" || f.ProjectPath == project {
			out = append(out, f)
		}
	}
	return out, nil
}

func (s *stubPlanRepo) DeleteStalePlanFiles(context.Context, time.Duration) (int, error) {
	return 0, nil
}

func TestGetDiscoveredAgents_ScansPersistsAndThrottles(t *testing.T) {
	orch, _, _ := newSessionOrch(t)
	scanner := &stubAgentScanner{agents: []domain.DiscoveredAgent{{ID: "a1"}, {ID: "a2"}}}
	store := &stubAgentStore{list: []domain.DiscoveredAgent{{ID: "stored"}}}
	orch.SetAgentScanner(scanner)
	orch.SetDiscoveredAgentRepo(store)

	got, err := orch.GetDiscoveredAgents(bg)
	if err != nil || len(got) != 1 || got[0].ID != "stored" {
		t.Fatalf("with a repo the stored list is returned: %v %v", got, err)
	}
	if len(store.upserted) != 2 || scanner.scans != 1 {
		t.Errorf("scan results must be persisted: upserted=%d scans=%d", len(store.upserted), scanner.scans)
	}
	if _, err := orch.GetDiscoveredAgents(bg); err != nil || scanner.scans != 1 {
		t.Errorf("a second call within 30s must not rescan (scans=%d, err=%v)", scanner.scans, err)
	}
}

func TestGetDiscoveredAgents_ErrorBranchesAndNoRepo(t *testing.T) {
	// Scanner failure is logged, not returned; no repo -> empty result.
	orch, _, _ := newSessionOrch(t)
	orch.SetAgentScanner(&stubAgentScanner{agentsErr: errInjected})
	if got, err := orch.GetDiscoveredAgents(bg); err != nil || len(got) != 0 {
		t.Errorf("scanner error: %v %v", got, err)
	}

	// Without a repo the freshly scanned agents are returned directly.
	orch2, _, _ := newSessionOrch(t)
	orch2.SetAgentScanner(&stubAgentScanner{agents: []domain.DiscoveredAgent{{ID: "fresh"}}})
	if got, _ := orch2.GetDiscoveredAgents(bg); len(got) != 1 || got[0].ID != "fresh" {
		t.Errorf("no repo: %v", got)
	}

	// Upsert failure is tolerated; List failure is returned.
	orch3, _, _ := newSessionOrch(t)
	orch3.SetAgentScanner(&stubAgentScanner{agents: []domain.DiscoveredAgent{{ID: "x"}}})
	orch3.SetDiscoveredAgentRepo(&stubAgentStore{upsertErr: errInjected, listErr: errInjected})
	if _, err := orch3.GetDiscoveredAgents(bg); !errors.Is(err, errInjected) {
		t.Errorf("list error: %v", err)
	}

	// No scanner at all: just the repo.
	orch4, _, _ := newSessionOrch(t)
	orch4.SetDiscoveredAgentRepo(&stubAgentStore{list: []domain.DiscoveredAgent{{ID: "only-stored"}}})
	if got, _ := orch4.GetDiscoveredAgents(bg); len(got) != 1 {
		t.Errorf("repo only: %v", got)
	}
}

func TestGetDiscoveredPlanFiles_AllBranches(t *testing.T) {
	// Nothing configured.
	orch, _, _ := newSessionOrch(t)
	if _, err := orch.GetDiscoveredPlanFiles(bg, "/p"); !errors.Is(err, domain.ErrSubsystemNotConfigured) {
		t.Errorf("unconfigured: %v", err)
	}

	// Repo only: empty project lists everything; a project lists that project.
	repo := &stubPlanRepo{stored: []domain.DiscoveredPlanFile{{ID: "1", ProjectPath: "/a"}, {ID: "2", ProjectPath: "/b"}}}
	withRepo := services.NewOrchestrator(services.NewDiscoveryService(), newMemRepo(), &noopWriter{}, nil,
		services.WithDisableBackgroundWorkers(), services.WithPlanFileRepo(repo))
	defer withRepo.Stop()
	if got, err := withRepo.GetDiscoveredPlanFiles(bg, ""); err != nil || len(got) != 2 {
		t.Errorf("all: %v %v", got, err)
	}
	if got, err := withRepo.GetDiscoveredPlanFiles(bg, "/a"); err != nil || len(got) != 1 {
		t.Errorf("project: %v %v", got, err)
	}

	// Scanner only: results are returned directly; empty project is unsupported.
	scanOnly, _, _ := newSessionOrch(t)
	scanOnly.SetAgentScanner(&stubAgentScanner{plans: []domain.DiscoveredPlanFile{{ID: "s1"}}})
	if got, err := scanOnly.GetDiscoveredPlanFiles(bg, "/p"); err != nil || len(got) != 1 {
		t.Errorf("scanner only: %v %v", got, err)
	}
	if _, err := scanOnly.GetDiscoveredPlanFiles(bg, ""); !errors.Is(err, domain.ErrSubsystemNotConfigured) {
		t.Errorf("scanner only, empty project: %v", err)
	}
	// Scanner that found nothing and no repo.
	empty, _, _ := newSessionOrch(t)
	empty.SetAgentScanner(&stubAgentScanner{})
	if _, err := empty.GetDiscoveredPlanFiles(bg, "/p"); !errors.Is(err, domain.ErrSubsystemNotConfigured) {
		t.Errorf("nothing found: %v", err)
	}

	// Scanner error is returned; scanner + repo persists then lists.
	failing, _, _ := newSessionOrch(t)
	failing.SetAgentScanner(&stubAgentScanner{plansErr: errInjected})
	if _, err := failing.GetDiscoveredPlanFiles(bg, "/p"); !errors.Is(err, errInjected) {
		t.Errorf("scan error: %v", err)
	}
	both := &stubPlanRepo{upsertErr: errInjected} // upsert failures are tolerated
	full := services.NewOrchestrator(services.NewDiscoveryService(), newMemRepo(), &noopWriter{}, nil,
		services.WithDisableBackgroundWorkers(), services.WithPlanFileRepo(both))
	defer full.Stop()
	full.SetAgentScanner(&stubAgentScanner{plans: []domain.DiscoveredPlanFile{{ID: "n", ProjectPath: "/p"}}})
	if got, err := full.GetDiscoveredPlanFiles(bg, "/p"); err != nil || len(got) != 1 {
		t.Errorf("scanner+repo: %v %v", got, err)
	}
}

func TestDelegateToNexus_BuildsInstructionFromConfiguredAddress(t *testing.T) {
	orch, repo, _ := newSessionOrch(t, services.WithDaemonAddr("http://nexus.example:9"))
	if err := repo.memAISessionRepo.SaveAISession(bg, domain.AISession{
		ID: "s1", Status: domain.SessionStatusActive, ProjectPath: "/work/proj",
	}); err != nil {
		t.Fatal(err)
	}
	text, err := orch.DelegateToNexus(bg, "s1")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"http://nexus.example:9/api/tasks", "/work/proj", "Nexus session ID: s1"} {
		if !strings.Contains(text, want) {
			t.Errorf("instruction missing %q:\n%s", want, text)
		}
	}
	if got, _ := repo.GetAISessionByID(bg, "s1"); !got.DelegatedToNexus || got.DelegationTimestamp == nil {
		t.Errorf("session not marked delegated: %+v", got)
	}

	if _, err := orch.DelegateToNexus(bg, "ghost"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("unknown session: %v", err)
	}
	repo.set("Save", errInjected)
	if _, err := orch.DelegateToNexus(bg, "s1"); !errors.Is(err, errInjected) {
		t.Errorf("save failure: %v", err)
	}
}

var _ ports.AgentScanner = (*stubAgentScanner)(nil)
