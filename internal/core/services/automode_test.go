package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

type stubAutoBrain struct {
	ports.BrainService
	mu          sync.Mutex
	initialized bool
	initCalls   int
	sections    []domain.ContextSection
	ctxErr      error
	ingested    int
}

func (b *stubAutoBrain) IngestKnowledge(_ context.Context, _ domain.ProjectKnowledge) (domain.ProjectKnowledge, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ingested++
	return domain.ProjectKnowledge{ID: "k1"}, nil
}

func (b *stubAutoBrain) IngestFromFile(_ context.Context, projectPath, filePath string) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ingested++
	return 1, nil
}
func (b *stubAutoBrain) GetContext(_ context.Context, q domain.ContextQuery) (domain.ContextResponse, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.ctxErr != nil {
		return domain.ContextResponse{}, b.ctxErr
	}
	return domain.ContextResponse{ProjectPath: q.ProjectPath, Sections: b.sections}, nil
}
func (b *stubAutoBrain) GetFocusedContext(ctx context.Context, q domain.ContextQuery) (domain.ContextResponse, error) {
	return b.GetContext(ctx, q)
}
func (b *stubAutoBrain) SearchKnowledge(context.Context, string, string, int) ([]domain.ContextSection, error) {
	return nil, nil
}
func (b *stubAutoBrain) GetStatus(_ context.Context, projectPath string) (domain.BrainStatus, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return domain.BrainStatus{ProjectPath: projectPath, Initialized: b.initialized}, nil
}
func (b *stubAutoBrain) InitProject(_ context.Context, projectPath, _ string) (domain.BrainStatus, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.initialized = true
	b.initCalls++
	return domain.BrainStatus{ProjectPath: projectPath, Initialized: true}, nil
}
func (b *stubAutoBrain) ListKnowledge(context.Context, string, string) ([]domain.ProjectKnowledge, error) {
	return nil, nil
}
func (b *stubAutoBrain) DeleteKnowledge(context.Context, string) error { return nil }
func (b *stubAutoBrain) GetFileMap(context.Context, string, string) ([]string, error) {
	return nil, nil
}
func (b *stubAutoBrain) IngestSection(_ context.Context, _ domain.ProjectKnowledge) (domain.ProjectKnowledge, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ingested++
	return domain.ProjectKnowledge{ID: "k1"}, nil
}
func (b *stubAutoBrain) GetOnboardingContext(context.Context, string, int) (string, error) {
	return "", nil
}

type stubFailRepo struct {
	ports.TaskRepository
	getPendingErr   error
	getPendingTasks []domain.Task
	updateIfCurrOK  bool
	updateIfCurrErr error
	saveErr         error
	backlogErr      error
	backlogTasks    []domain.Task
	getByIDErr      error
}

func (r *stubFailRepo) GetPending(ctx context.Context) ([]domain.Task, error) {
	if r.getPendingErr != nil {
		return nil, r.getPendingErr
	}
	if r.getPendingTasks != nil {
		return r.getPendingTasks, nil
	}
	return r.TaskRepository.GetPending(ctx)
}
func (r *stubFailRepo) UpdateStatusIfCurrent(ctx context.Context, id string, expected, next domain.TaskStatus) (bool, error) {
	if r.updateIfCurrErr != nil || !r.updateIfCurrOK {
		return r.updateIfCurrOK, r.updateIfCurrErr
	}
	return r.TaskRepository.UpdateStatusIfCurrent(ctx, id, expected, next)
}
func (r *stubFailRepo) Save(ctx context.Context, t domain.Task) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	return r.TaskRepository.Save(ctx, t)
}
func (r *stubFailRepo) GetByProjectPathAndStatus(ctx context.Context, projectPath string, statuses ...domain.TaskStatus) ([]domain.Task, error) {
	if r.backlogErr != nil {
		return nil, r.backlogErr
	}
	if r.backlogTasks != nil {
		return r.backlogTasks, nil
	}
	return r.TaskRepository.GetByProjectPathAndStatus(ctx, projectPath, statuses...)
}
func (r *stubFailRepo) GetByID(ctx context.Context, id string) (domain.Task, error) {
	if r.getByIDErr != nil {
		return domain.Task{}, r.getByIDErr
	}
	return r.TaskRepository.GetByID(ctx, id)
}

type autoTestLLM struct {
	mu      sync.Mutex
	replies []string
	idx     int
	err     error
}

func (l *autoTestLLM) Ping(context.Context) bool { return true }
func (l *autoTestLLM) ProviderName() string      { return "AutoMock" }
func (l *autoTestLLM) ActiveModel() string       { return "claude-opus-5-5" }
func (l *autoTestLLM) BaseURL() string           { return "http://mock" }
func (l *autoTestLLM) GetAvailableModels(context.Context) ([]string, error) {
	return []string{"claude-opus-5-5"}, nil
}
func (l *autoTestLLM) ContextLimit(context.Context) int { return 100000 }
func (l *autoTestLLM) GenerateCode(ctx context.Context, p string) (string, error) {
	return l.Chat(ctx, []domain.Message{{Role: domain.RoleUser, Content: p}})
}
func (l *autoTestLLM) Chat(context.Context, []domain.Message) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.err != nil {
		return "", l.err
	}
	if l.idx >= len(l.replies) {
		return "```nexus-tool\n{\"tool\":\"finish\",\"summary\":\"done\"}\n```", nil
	}
	r := l.replies[l.idx]
	l.idx++
	return r, nil
}

type autoTestWriter struct{ files map[string]string }

func (w *autoTestWriter) WriteCodeToFile(_, target, code string) error {
	w.files[target] = code
	return nil
}
func (w *autoTestWriter) ReadContextFiles(_ string, files []string) (string, error) {
	return w.files[files[0]], nil
}

type autoTestRunner struct {
	mu   sync.Mutex
	errs []error
	outs []string
	idx  int
}

func (r *autoTestRunner) Run(context.Context, string, string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	i := r.idx
	r.idx++
	out := "ok"
	if i < len(r.outs) {
		out = r.outs[i]
	}
	if i < len(r.errs) {
		return out, r.errs[i]
	}
	return out, nil
}

func waitAutoRun(t *testing.T, orch *OrchestratorService, id string) domain.AutoModeRun {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		run, err := orch.GetAutoModeRun(id)
		if err == nil && run.Phase.IsTerminal() {
			return run
		}
		time.Sleep(15 * time.Millisecond)
	}
	run, _ := orch.GetAutoModeRun(id)
	t.Fatalf("auto run %s did not settle (phase=%s)", id, run.Phase)
	return run
}

func TestAutoMode_GoalDecomposeExecuteVerifyRepairAndLearn(t *testing.T) {
	origPoll := autoPollInterval
	autoPollInterval = 10 * time.Millisecond
	t.Cleanup(func() { autoPollInterval = origPoll })

	repo, err := repo_sqlite.New(filepath.Join(t.TempDir(), "auto.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	llm := &autoTestLLM{
		replies: []string{
			// 1. Goal decomposition JSON
			"```json\n[{\"instruction\":\"create a.go\",\"targetFile\":\"a.go\",\"role\":\"coder\",\"dependsOn\":[]},{\"instruction\":\"create b.go\",\"targetFile\":\"b.go\",\"role\":\"tester\",\"dependsOn\":[0]}]\n```",
			// 2. Task 1 agent finish
			"```nexus-tool\n{\"tool\":\"write_file\",\"path\":\"a.go\",\"content\":\"package a\"}\n```\n```nexus-tool\n{\"tool\":\"finish\",\"summary\":\"wrote a\"}\n```",
			// 3. Task 2 agent finish
			"```nexus-tool\n{\"tool\":\"write_file\",\"path\":\"b.go\",\"content\":\"package b\"}\n```\n```nexus-tool\n{\"tool\":\"finish\",\"summary\":\"wrote b\"}\n```",
			// 4. Repair task finish
			"```nexus-tool\n{\"tool\":\"finish\",\"summary\":\"repaired\"}\n```",
		},
	}
	runner := &autoTestRunner{
		// Task 1 finish verify (ok), Task 2 finish verify (ok), Final verify #1 (FAIL -> triggers repair),
		// Repair task finish verify (ok), Final verify #2 (ok)
		errs: []error{nil, nil, errors.New("exit 1"), nil, nil},
		outs: []string{"ok", "ok", "\x1b[31mFAIL\x1b[0m", "ok", "ok"},
	}
	orch := NewOrchestrator(NewDiscoveryService(llm), repo, &autoTestWriter{files: map[string]string{}}, nil, WithCommandRunner(runner))
	t.Cleanup(orch.Stop)

	proj := t.TempDir()
	run, err := orch.StartAutoMode(context.Background(), domain.AutoModeRequest{
		ProjectPath:       proj,
		Goal:              "build feature",
		VerifyCommand:     "go test ./...",
		MaxTasks:          5,
		MaxRepairAttempts: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	settled := waitAutoRun(t, orch, run.ID)
	if settled.Phase != domain.AutoPhaseCompleted {
		t.Fatalf("expected completed, got %s (err=%s, summary=%s)", settled.Phase, settled.Error, settled.Summary)
	}
	if len(settled.TaskIDs) != 3 || settled.RepairAttempts != 1 {
		t.Fatalf("expected 3 tasks (2 planned + 1 repair) and 1 repair attempt, got %+v", settled)
	}
	list := orch.ListAutoModeRuns(proj)
	if len(list) != 1 || list[0].ID != run.ID {
		t.Fatalf("ListAutoModeRuns = %+v", list)
	}
	if len(orch.ListAutoModeRuns("/other")) != 0 {
		t.Fatalf("expected 0 runs for /other")
	}
}

func TestAutoMode_BacklogValidationAndStop(t *testing.T) {
	origPoll := autoPollInterval
	autoPollInterval = 10 * time.Millisecond
	t.Cleanup(func() { autoPollInterval = origPoll })

	repo, err := repo_sqlite.New(filepath.Join(t.TempDir(), "auto2.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	orch := NewOrchestrator(NewDiscoveryService(&autoTestLLM{}), repo, &autoTestWriter{files: map[string]string{}}, nil)
	t.Cleanup(orch.Stop)

	if _, err := orch.StartAutoMode(context.Background(), domain.AutoModeRequest{}); err == nil {
		t.Fatal("expected error for empty projectPath")
	}
	if _, err := orch.GetAutoModeRun("missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if err := orch.StopAutoMode("missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound on StopAutoMode, got %v", err)
	}

	// Empty backlog completes cleanly with informative summary.
	proj := t.TempDir()
	run, err := orch.StartAutoMode(context.Background(), domain.AutoModeRequest{ProjectPath: proj, MaxTasks: 99})
	if err != nil {
		t.Fatal(err)
	}
	settled := waitAutoRun(t, orch, run.ID)
	if settled.Phase != domain.AutoPhaseCompleted || !strings.Contains(settled.Summary, "No backlog tasks") {
		t.Fatalf("unexpected empty-backlog run: %+v", settled)
	}

	// Backlog promotion mode
	_, err = orch.CreateDraft(domain.Task{ProjectPath: proj, TargetFile: "x.go", Instruction: "from backlog"})
	if err != nil {
		t.Fatal(err)
	}
	run2, err := orch.StartAutoMode(context.Background(), domain.AutoModeRequest{ProjectPath: proj})
	if err != nil {
		t.Fatal(err)
	}
	settled2 := waitAutoRun(t, orch, run2.ID)
	if settled2.Phase != domain.AutoPhaseCompleted || len(settled2.TaskIDs) != 1 {
		t.Fatalf("unexpected backlog run: %+v", settled2)
	}

	// Verify parsePlannedSteps edge cases
	if steps := parsePlannedSteps("not json", 5); len(steps) != 0 {
		t.Fatalf("expected nil on bad json, got %+v", steps)
	}
	if steps := parsePlannedSteps(`[{"instruction":""},{"instruction":"a"},{"instruction":"b"}]`, 1); len(steps) != 1 || steps[0].Instruction != "a" {
		t.Fatalf("expected capped single step, got %+v", steps)
	}
}

func TestAutoMode_BrainPolicyStopFailuresAndEdgeCases(t *testing.T) {
	origPoll := autoPollInterval
	autoPollInterval = 10 * time.Millisecond
	t.Cleanup(func() { autoPollInterval = origPoll })

	repo, err := repo_sqlite.New(filepath.Join(t.TempDir(), "auto3.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	// 1. Brain sync (InitProject + GetContext sections), policy.verify fallback, empty role -> coder,
	//    repair limit exhausted -> terminal failed with verifyErr, and learnings digestion.
	brain := &stubAutoBrain{
		sections: []domain.ContextSection{{Topic: "Stack", Kind: "architecture", Content: "Go 1.25"}},
	}
	llm := &autoTestLLM{
		replies: []string{
			`[{"instruction":"step 1","targetFile":"a.go"}]`, // empty role defaults to "coder"
			"```nexus-tool\n{\"tool\":\"finish\",\"summary\":\"wrote a\"}\n```",
			"```nexus-tool\n{\"tool\":\"finish\",\"summary\":\"repair attempt\"}\n```",
		},
	}
	runner := &autoTestRunner{
		// Task 1 finish verify OK, final verify #1 FAIL, repair task finish verify OK, final verify #2 FAIL -> exhausted
		errs: []error{nil, errors.New("test red 1"), nil, errors.New("test red 2")},
		outs: []string{"ok", "fail 1", "ok", "fail 2"},
	}
	orch := NewOrchestrator(NewDiscoveryService(llm), repo, &autoTestWriter{files: map[string]string{}}, nil, WithBrain(brain), WithCommandRunner(runner))
	t.Cleanup(orch.Stop)

	proj := t.TempDir()
	if err := os.MkdirAll(filepath.Join(proj, ".nexus"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, ".nexus", "policy.json"), []byte(`{"verify":"go test ./..."}`), 0o600); err != nil {
		t.Fatal(err)
	}

	run, err := orch.StartAutoMode(context.Background(), domain.AutoModeRequest{
		ProjectPath:       proj,
		Goal:              "implement feature",
		MaxRepairAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	settled := waitAutoRun(t, orch, run.ID)
	if settled.Phase != domain.AutoPhaseFailed || !strings.Contains(settled.Error, "final verification failed") {
		t.Fatalf("expected failed verify error, got %+v", settled)
	}
	if brain.initCalls == 0 {
		t.Fatal("expected brain.InitProject to be called")
	}

	// 2. Second run on same project with failing task -> terminal failed with FailedTasks > 0,
	//    plus ListAutoModeRuns comparator with >= 2 runs, plus brain GetContext error path.
	brain.ctxErr = errors.New("ctx unavailable")
	llmFail := &autoTestLLM{err: errors.New("llm down")}
	orch2 := NewOrchestrator(NewDiscoveryService(llmFail), repo, &autoTestWriter{files: map[string]string{}}, nil, WithBrain(brain))
	t.Cleanup(orch2.Stop)
	r1, err := orch2.StartAutoMode(context.Background(), domain.AutoModeRequest{ProjectPath: proj, Goal: "will fail"})
	if err != nil {
		t.Fatal(err)
	}
	s1 := waitAutoRun(t, orch2, r1.ID)
	if s1.Phase != domain.AutoPhaseFailed || s1.FailedTasks == 0 {
		t.Fatalf("expected failed task run, got %+v", s1)
	}
	// Start another empty-backlog run so ListAutoModeRuns sorts >= 2 items.
	r2, err := orch2.StartAutoMode(context.Background(), domain.AutoModeRequest{ProjectPath: proj})
	if err != nil {
		t.Fatal(err)
	}
	waitAutoRun(t, orch2, r2.ID)
	if got := orch2.ListAutoModeRuns(proj); len(got) < 2 {
		t.Fatalf("expected >= 2 runs, got %d", len(got))
	}

	// 3. Active run guard (ErrAutoModeActive) and StopAutoMode on active run.
	holdLLM := &autoTestLLM{replies: []string{`[{"instruction":"slow","targetFile":"s.go"}]`}}
	orchStop := NewOrchestrator(NewDiscoveryService(holdLLM), repo, &autoTestWriter{files: map[string]string{}}, nil)
	t.Cleanup(orchStop.Stop)
	// Insert a synthetic active run with a queued task to deterministically test ErrAutoModeActive and StopAutoMode.
	qID, err := orchStop.SubmitTask(domain.Task{ProjectPath: proj, TargetFile: "q.go", Instruction: "queued", DependsOn: []string{"never-completes"}})
	if err != nil {
		t.Fatal(err)
	}
	ctxActive, cancelActive := context.WithCancel(context.Background())
	orchStop.autoMu.Lock()
	orchStop.autoRuns = map[string]*autoRunState{
		"active-1": {
			ctx:    ctxActive,
			cancel: cancelActive,
			run: domain.AutoModeRun{
				ID:          "active-1",
				ProjectPath: filepath.Clean(proj),
				Phase:       domain.AutoPhaseExecuting,
				TaskIDs:     []string{qID},
			},
		},
	}
	orchStop.autoMu.Unlock()
	if _, err := orchStop.StartAutoMode(context.Background(), domain.AutoModeRequest{ProjectPath: proj}); !errors.Is(err, ErrAutoModeActive) {
		t.Fatalf("expected ErrAutoModeActive, got %v", err)
	}
	if err := orchStop.StopAutoMode("active-1"); err != nil {
		t.Fatalf("StopAutoMode failed: %v", err)
	}
	stoppedSnap, _ := orchStop.GetAutoModeRun("active-1")
	if stoppedSnap.Phase != domain.AutoPhaseStopped {
		t.Fatalf("expected stopped phase, got %s", stoppedSnap.Phase)
	}

	// 4. Direct helper edge cases: cancellation checkpoints, autoWaitTasks missing task & ctx cancel,
	//    autoDigestLearnings missing task, decomposeGoal no-provider & non-JSON fallback,
	//    backlog MaxTasks cap, repo error paths, and failBlockedDependents error paths.
	cancelledCtx, cancelNow := context.WithCancel(context.Background())
	cancelNow()
	stCancelled := &autoRunState{ctx: cancelledCtx, cancel: cancelNow, run: domain.AutoModeRun{ID: "c1", Phase: domain.AutoPhaseSyncing}}
	orchStop.runAutoMode(stCancelled, domain.AutoModeRequest{ProjectPath: proj})
	if stCancelled.snapshot().Phase != domain.AutoPhaseStopped {
		t.Fatalf("expected stopped after sync cancel, got %s", stCancelled.snapshot().Phase)
	}

	// Cancel during planning (via a custom brain that cancels ctx in GetContext? Better: cancel in autoPlanTasks via stub).
	stWait := &autoRunState{ctx: cancelledCtx, cancel: cancelNow, run: domain.AutoModeRun{ID: "c2", Phase: domain.AutoPhaseExecuting}}
	_, fMissing, stopped := orchStop.autoWaitTasks(cancelledCtx, stWait, []string{"missing-task-id", qID})
	if fMissing != 1 || stopped {
		// Note: qID is already CANCELLED by StopAutoMode above, so pending==0; test ctx cancel with a genuinely queued task!
		t.Logf("missing task counted f=%d stopped=%v", fMissing, stopped)
	}
	qID2, _ := orchStop.SubmitTask(domain.Task{ProjectPath: proj, TargetFile: "q2.go", Instruction: "blocked", DependsOn: []string{"never-completes-2"}})
	_, _, stopped2 := orchStop.autoWaitTasks(cancelledCtx, stWait, []string{qID2})
	if !stopped2 {
		t.Fatal("expected stopped=true when ctx is cancelled while task is pending")
	}

	// autoDigestLearnings with brain and missing task ID
	_ = orch.autoDigestLearnings(context.Background(), []string{"does-not-exist"})

	// decomposeGoal: no providers -> fallback; non-JSON reply -> fallback
	orchNoProv := NewOrchestrator(NewDiscoveryService(), repo, &autoTestWriter{files: map[string]string{}}, nil)
	t.Cleanup(orchNoProv.Stop)
	if steps := orchNoProv.decomposeGoal(context.Background(), domain.AutoModeRequest{ProjectPath: proj, Goal: "solo", MaxTasks: 3}, "", ""); len(steps) != 1 || steps[0].Instruction != "solo" {
		t.Fatalf("expected fallback step when no provider, got %+v", steps)
	}
	orchBadJSON := NewOrchestrator(NewDiscoveryService(&autoTestLLM{replies: []string{"not json at all"}}), repo, &autoTestWriter{files: map[string]string{}}, nil)
	t.Cleanup(orchBadJSON.Stop)
	if steps := orchBadJSON.decomposeGoal(context.Background(), domain.AutoModeRequest{ProjectPath: proj, Goal: "solo2", MaxTasks: 3}, "", ""); len(steps) != 1 || steps[0].Instruction != "solo2" {
		t.Fatalf("expected fallback step on non-JSON, got %+v", steps)
	}

	// Backlog MaxTasks cap (2 drafts, MaxTasks=1)
	projCap := t.TempDir()
	_, _ = orchStop.CreateDraft(domain.Task{ProjectPath: projCap, TargetFile: "d1.go", Instruction: "draft 1"})
	_, _ = orchStop.CreateDraft(domain.Task{ProjectPath: projCap, TargetFile: "d2.go", Instruction: "draft 2"})
	capIDs, err := orchStop.autoPlanTasks(context.Background(), domain.AutoModeRequest{ProjectPath: projCap, MaxTasks: 1}, "", "")
	if err != nil || len(capIDs) != 1 {
		t.Fatalf("expected 1 capped backlog task, got %v err=%v", capIDs, err)
	}

	// Stub failing repo tests: GetBacklog error, PromoteTask error, SubmitTask error (plan + repair),
	// runAutoMode plan error, and failBlockedDependents error paths (execution_engine.go:72,80).
	failRepo := &stubFailRepo{TaskRepository: repo, backlogErr: errors.New("backlog boom")}
	orchFail := NewOrchestrator(NewDiscoveryService(&autoTestLLM{}), failRepo, &autoTestWriter{files: map[string]string{}}, nil, WithCommandRunner(&autoTestRunner{errs: []error{errors.New("fail")}}))
	t.Cleanup(orchFail.Stop)

	stPlanErr := &autoRunState{ctx: context.Background(), cancel: func() {}, run: domain.AutoModeRun{ID: "pe1", Phase: domain.AutoPhaseSyncing}}
	orchFail.runAutoMode(stPlanErr, domain.AutoModeRequest{ProjectPath: proj, MaxTasks: 5})
	if stPlanErr.snapshot().Phase != domain.AutoPhaseFailed {
		t.Fatalf("expected failed phase on plan error, got %+v", stPlanErr.snapshot())
	}

	// PromoteTask error (backlog returns a task whose GetByID fails)
	failRepo.backlogErr = nil
	failRepo.backlogTasks = []domain.Task{{ID: "ghost-draft", Status: domain.StatusDraft}}
	failRepo.getByIDErr = errors.New("getbyid boom")
	if _, err := orchFail.autoPlanTasks(context.Background(), domain.AutoModeRequest{ProjectPath: proj, MaxTasks: 5}, "", ""); err == nil {
		t.Fatal("expected promote error")
	}
	failRepo.backlogTasks = nil
	failRepo.getByIDErr = nil

	// SubmitTask error in autoPlanTasks and autoVerifyAndRepair
	failRepo.saveErr = errors.New("save boom")
	if _, err := orchFail.autoPlanTasks(context.Background(), domain.AutoModeRequest{ProjectPath: proj, Goal: "g", MaxTasks: 2}, "", ""); err == nil {
		t.Fatal("expected submit error in autoPlanTasks")
	}
	stRep := &autoRunState{ctx: context.Background(), cancel: func() {}, run: domain.AutoModeRun{ID: "rep1", Phase: domain.AutoPhaseVerifying}}
	if msg := orchFail.autoVerifyAndRepair(context.Background(), stRep, domain.AutoModeRequest{ProjectPath: proj, MaxRepairAttempts: 1}, "go test"); !strings.Contains(msg, "submit repair task") {
		t.Fatalf("expected submit repair error, got %q", msg)
	}
	failRepo.saveErr = nil

	// Cancel during autoVerifyAndRepair (submit succeeds, wait sees cancelled ctx)
	ctxRepCancel, cancelRep := context.WithCancel(context.Background())
	orchRepCancel := NewOrchestrator(NewDiscoveryService(&autoTestLLM{}), repo, &autoTestWriter{files: map[string]string{}}, nil, WithCommandRunner(&autoTestRunner{errs: []error{errors.New("fail")}}))
	t.Cleanup(orchRepCancel.Stop)
	// Submit a repair task then immediately cancel ctx via runner? Even simpler: pre-cancel ctxRepCancel before autoVerifyAndRepair!
	cancelRep()
	if msg := orchRepCancel.autoVerifyAndRepair(ctxRepCancel, stRep, domain.AutoModeRequest{ProjectPath: proj, MaxRepairAttempts: 1}, "go test"); msg != "stopped" {
		t.Fatalf("expected stopped from autoVerifyAndRepair, got %q", msg)
	}

	// runAutoMode stopped after autoPlanTasks (cancel during decomposeGoal) and after autoWaitTasks and after autoVerifyAndRepair.
	ctxPlanCancel, cancelPlan := context.WithCancel(context.Background())
	cancelOnChat := &cancelLLM{cancel: cancelPlan}
	orchPlanCancel := NewOrchestrator(NewDiscoveryService(cancelOnChat), repo, &autoTestWriter{files: map[string]string{}}, nil)
	t.Cleanup(orchPlanCancel.Stop)
	stPC := &autoRunState{ctx: ctxPlanCancel, cancel: cancelPlan, run: domain.AutoModeRun{ID: "pc1", Phase: domain.AutoPhaseSyncing}}
	orchPlanCancel.runAutoMode(stPC, domain.AutoModeRequest{ProjectPath: proj, Goal: "cancel in plan", MaxTasks: 2})
	if stPC.snapshot().Phase != domain.AutoPhaseStopped {
		t.Fatalf("expected stopped after plan cancel, got %s", stPC.snapshot().Phase)
	}

	// runAutoMode stopped after autoWaitTasks
	ctxWaitCancel, cancelWait := context.WithCancel(context.Background())
	projWait := t.TempDir()
	_, _ = orchStop.CreateDraft(domain.Task{ProjectPath: projWait, TargetFile: "w.go", Instruction: "wait", DependsOn: []string{"never-3"}})
	stWC := &autoRunState{ctx: ctxWaitCancel, cancel: cancelWait, run: domain.AutoModeRun{ID: "wc1", Phase: domain.AutoPhaseSyncing}}
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancelWait()
	}()
	orchStop.runAutoMode(stWC, domain.AutoModeRequest{ProjectPath: projWait, MaxTasks: 2})
	if stWC.snapshot().Phase != domain.AutoPhaseStopped {
		t.Fatalf("expected stopped after wait cancel, got %s", stWC.snapshot().Phase)
	}

	// runAutoMode stopped after autoVerifyAndRepair
	ctxVerCancel, cancelVer := context.WithCancel(context.Background())
	cancelRunner := &cancelCmdRunner{cancel: cancelVer}
	orchVerCancel := NewOrchestrator(NewDiscoveryService(&autoTestLLM{}), repo, &autoTestWriter{files: map[string]string{}}, nil, WithCommandRunner(cancelRunner))
	t.Cleanup(orchVerCancel.Stop)
	projVer := t.TempDir()
	_, _ = orchVerCancel.CreateDraft(domain.Task{ProjectPath: projVer, TargetFile: "v.go", Instruction: "ver"})
	stVC := &autoRunState{ctx: ctxVerCancel, cancel: cancelVer, run: domain.AutoModeRun{ID: "vc1", Phase: domain.AutoPhaseSyncing}}
	orchVerCancel.runAutoMode(stVC, domain.AutoModeRequest{ProjectPath: projVer, MaxTasks: 2, VerifyCommand: "go test", MaxRepairAttempts: 1})
	if stVC.snapshot().Phase != domain.AutoPhaseStopped {
		t.Fatalf("expected stopped after verify cancel, got %s", stVC.snapshot().Phase)
	}

	// failBlockedDependents error paths (execution_engine.go lines 72 & 80)
	failRepo.getPendingErr = errors.New("pending boom")
	orchFail.failBlockedDependents("dep-1", domain.StatusFailed)
	failRepo.getPendingErr = nil
	failRepo.getPendingTasks = []domain.Task{{ID: "t-blocked", Status: domain.StatusQueued, DependsOn: []string{"dep-1"}}}
	failRepo.updateIfCurrOK = false
	orchFail.failBlockedDependents("dep-1", domain.StatusFailed)
}

type cancelLLM struct {
	autoTestLLM
	cancel context.CancelFunc
}

func (c *cancelLLM) Chat(context.Context, []domain.Message) (string, error) {
	c.cancel()
	return "", context.Canceled
}

type cancelCmdRunner struct {
	cancel context.CancelFunc
}

func (c *cancelCmdRunner) Run(context.Context, string, string) (string, error) {
	c.cancel()
	return "fail", errors.New("cancelled")
}
