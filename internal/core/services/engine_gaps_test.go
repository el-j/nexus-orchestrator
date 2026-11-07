package services_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

// cancelledOnReadRepo reports every task as cancelled when the engine re-reads it, as if a user had
// cancelled it between the claim and the start of the run.
type cancelledOnReadRepo struct{ *faultyRepo }

func (r *cancelledOnReadRepo) GetByID(ctx context.Context, id string) (domain.Task, error) {
	t, err := r.faultyRepo.GetByID(ctx, id)
	t.Status = domain.StatusCancelled
	return t, err
}

func TestProcessNext_HonoursACancelThatLandedBetweenClaimAndStart(t *testing.T) {
	repo := &cancelledOnReadRepo{faultyRepo: newFaultyRepo()}
	llm := &scriptLLM{name: "p", alive: true, replies: []string{"never used"}}
	orch := services.NewOrchestrator(services.NewDiscoveryService(llm), repo, &noopWriter{}, nil, services.WithDisableBackgroundWorkers())
	t.Cleanup(orch.Stop)
	seedTask(t, repo.faultyRepo, "t1", domain.StatusQueued)
	if !orch.ProcessNext() {
		t.Fatal("the task was claimed, so ProcessNext reports true")
	}
	if llm.calls != 0 {
		t.Errorf("a cancelled task reached the model %d times", llm.calls)
	}
}

func TestProcessNext_ATaskWhoseProjectRulesCannotBeReadFailsEvenWhenTheStatusCannotBeSaved(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".nexus"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".nexus", "policy.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	llm := &scriptLLM{name: "p", alive: true, replies: []string{"x"}}
	e := newEngine(t, &noopWriter{}, nil, nil, llm)
	e.repo.set("UpdateStatus", true)
	id := e.queue(domain.Task{Instruction: "rules", ProjectPath: project})
	if !e.orch.ProcessNext() {
		t.Fatal("ProcessNext found no task")
	}
	if got := e.status(id); !strings.Contains(got.Logs, "not started") {
		t.Errorf("the refusal must be logged on the task: %q", got.Logs)
	}
	if llm.calls != 0 {
		t.Errorf("the model was called %d times for a task that could not start", llm.calls)
	}
}

// cancelsOnChat cancels the task it is serving while it answers, then fails.
type cancelsOnChat struct {
	*scriptLLM
	cancel func()
}

func (c *cancelsOnChat) Chat(ctx context.Context, h []domain.Message) (string, error) {
	c.cancel()
	return "", errInjected
}

func (c *cancelsOnChat) ChatStream(ctx context.Context, h []domain.Message, _ func(string)) (string, error) {
	return c.Chat(ctx, h)
}

func TestExecuteWithFallbackChain_StopsBeforeTheNextProviderWhenTheTaskWasCancelled(t *testing.T) {
	var e *engineFixture
	var id string
	first := &cancelsOnChat{scriptLLM: &scriptLLM{name: "first", alive: true}, cancel: func() { _ = e.orch.CancelTask(id) }}
	second := &scriptLLM{name: "second", alive: true, replies: []string{"never used"}}
	e = newEngine(t, &noopWriter{}, nil, nil, first, second)
	id = e.queue(domain.Task{Instruction: "cancelled mid-chain"})
	e.runOne(id, domain.StatusCancelled)
	if second.calls != 0 {
		t.Errorf("the next provider was asked %d times after the cancel", second.calls)
	}
}

// cancelsOnRun cancels the task when its verification command runs.
type cancelsOnRun struct{ cancel func() }

func (c *cancelsOnRun) Run(context.Context, string, string) (string, error) {
	c.cancel()
	return "ok", nil
}

var _ ports.CommandRunner = (*cancelsOnRun)(nil)

func TestAgentLoop_ACancelDuringAToolCallEndsTheRunBeforeAnotherTurn(t *testing.T) {
	var e *engineFixture
	var id string
	fs := newMemFS(map[string]string{})
	llm := &scriptLLM{name: "p", alive: true, limit: 100000, replies: []string{
		tool(`{"tool":"verify"}`),
		tool(`{"tool":"finish","summary":"never reached"}`),
	}}
	e = newEngine(t, fs, nil, &cancelsOnRun{cancel: func() { _ = e.orch.CancelTask(id) }}, llm)
	id = e.queue(agentTask(func(t *domain.Task) { t.VerificationCommand = "go test ./..." }))
	e.runOne(id, domain.StatusCancelled)
	if llm.calls != 1 {
		t.Errorf("the model was asked %d times; the cancelled task must not get a second turn", llm.calls)
	}
}

func TestAgentLoop_WaitsForTheResetWhenTheOnlyProviderIsCoolingDownAtTheStartOfATurn(t *testing.T) {
	limited := &domain.ProviderError{Provider: "p", Kind: domain.ErrKindRateLimited, Status: 429, ResetAt: time.Now().Add(time.Hour), Msg: "p: rate limited"}
	llm := &scriptLLM{name: "p", alive: true, limit: 100000, errs: []error{limited}, replies: []string{"x"}}
	e := newEngine(t, newMemFS(map[string]string{}), nil, nil, llm)
	e.runOne(e.queue(domain.Task{Instruction: "plain"}), domain.StatusWaiting) // a first task cools the provider down
	got := e.runOne(e.queue(agentTask(nil)), domain.StatusWaiting)
	if !strings.Contains(got.Logs, "cooling down") {
		t.Errorf("logs = %q", got.Logs)
	}
	if llm.calls != 1 {
		t.Errorf("the cooling provider was called %d times", llm.calls)
	}
}

func TestRules_AFailedTaskMemoryNamesTheVerificationCommandAndTheLastLog(t *testing.T) {
	root := projectWith(t, `{"remember":true}`)
	runner := &recordingRunner{}
	first := &scriptLLM{name: "first", alive: true, errs: []error{errInjected, errInjected, errInjected}}
	second := &scriptLLM{name: "second", alive: true, errs: []error{errInjected, errInjected, errInjected}}
	know := &mockKnowledgeRepo{}
	e := newEngine(t, &noopWriter{}, nil, runner, first, second)
	e.orch.WithBrain(services.NewBrainService(know, nil))
	id := e.queue(domain.Task{Instruction: "doomed twice", ProjectPath: root, TargetFile: "a.go", VerificationCommand: "go test ./..."})
	for e.status(id).Status != domain.StatusFailed {
		e.orch.ProcessNext()
	}
	var memory string
	for _, k := range know.entries {
		if k.Source == services.TaskMemorySource {
			memory = k.Content
		}
	}
	if !strings.Contains(memory, "verified with `go test ./...`") || !strings.Contains(memory, "last log:") {
		t.Errorf("memory = %q", memory)
	}
}

func TestRecordOutcome_IgnoresATaskItCannotFind(t *testing.T) {
	know := &mockKnowledgeRepo{}
	e := newEngine(t, &noopWriter{}, nil, nil, &scriptLLM{name: "p", alive: true})
	e.orch.WithBrain(services.NewBrainService(know, nil))
	e.orch.RecordOutcome("no-such-task", domain.StatusCompleted)
	if len(know.entries) != 0 {
		t.Errorf("a memory was stored for a task that does not exist: %+v", know.entries)
	}
}

func TestHoldReleased_AWaitingTaskWhoseProvidersCannotBeResolvedStaysPut(t *testing.T) {
	orch := newOrchOver(t, newFaultyRepo()) // no provider is registered at all
	if orch.HoldReleased(bg, domain.Task{ID: "w", Status: domain.StatusWaiting}) {
		t.Error("without a provider there is nothing to release the task to")
	}
}

// flipLLM answers the first liveness probe and then goes down, as a provider that stops between the
// router's cached check and the moment the engine asks.
type flipLLM struct {
	*scriptLLM
	pings int
}

func (f *flipLLM) Ping(context.Context) bool {
	f.pings++
	return f.pings == 1
}

func TestResolveChain_NamesTheModelWhenTheOnlyProviderServingItWentDown(t *testing.T) {
	llm := &flipLLM{scriptLLM: &scriptLLM{name: "p", alive: true, model: "big", models: []string{"big"}}}
	orch := services.NewOrchestrator(services.NewDiscoveryService(llm), newFaultyRepo(), &noopWriter{}, nil, services.WithDisableBackgroundWorkers())
	t.Cleanup(orch.Stop)
	_, err := orch.ResolveChain(domain.Task{ModelID: "big"})
	if err == nil || !strings.Contains(err.Error(), `model "big" not available`) {
		t.Errorf("err = %v", err)
	}
}

func TestTerminateAISession_AProcessThatCannotBeLookedUpIsLoggedNotReturned(t *testing.T) {
	defer services.SetFindProcess(func(int) (*os.Process, error) { return nil, errInjected })()
	orch, repo, _ := newSessionOrch(t)
	orch.SetDiscoveredAgentRepo(&stubAgentStore{list: []domain.DiscoveredAgent{runningAgent(4242)}})
	_ = repo.memAISessionRepo.SaveAISession(bg, domain.AISession{ID: "s", Status: domain.SessionStatusActive, PID: 4242})
	if err := orch.TerminateAISession(bg, "s", false); err != nil {
		t.Fatalf("a lookup failure must not fail the terminate: %v", err)
	}
}

// hold parks a task for the cooldown of llm, the way the engine does when every provider is limited.
func hold(t *testing.T, orch *services.OrchestratorService, llm ports.LLMClient, id string, reset time.Duration) {
	t.Helper()
	orch.NoteProviderError(llm, &domain.ProviderError{Provider: llm.ProviderName(), Kind: domain.ErrKindRateLimited, Status: 429, ResetAt: time.Now().Add(reset), Msg: "limited"})
	if !orch.HoldUntilReset(domain.Task{ID: id, ProjectPath: "/proj", Status: domain.StatusProcessing}, []ports.LLMClient{llm}) {
		t.Fatal("the task was not held although its only provider is cooling down")
	}
}

func TestHoldUntilReset_HoldingATaskAgainReplacesItsTimer(t *testing.T) {
	defer services.SetWaitGrace(0)()
	llm := &scriptLLM{name: "p", alive: true}
	repo := newFaultyRepo()
	orch := services.NewOrchestrator(services.NewDiscoveryService(llm), repo, &noopWriter{}, nil, services.WithDisableBackgroundWorkers())
	t.Cleanup(orch.Stop)
	seedTask(t, repo, "w", domain.StatusProcessing)
	hold(t, orch, llm, "w", time.Hour)
	hold(t, orch, llm, "w", time.Hour) // the first timer is stopped and replaced, not leaked
}

func TestHoldUntilReset_ATimerThatFiresAfterShutdownResumesNothing(t *testing.T) {
	defer services.SetWaitGrace(0)()
	llm := &scriptLLM{name: "p", alive: true}
	repo := newFaultyRepo()
	orch := services.NewOrchestrator(services.NewDiscoveryService(llm), repo, &noopWriter{}, nil, services.WithDisableBackgroundWorkers())
	seedTask(t, repo, "w", domain.StatusProcessing)
	hold(t, orch, llm, "w", 60*time.Millisecond)
	orch.Stop()
	time.Sleep(200 * time.Millisecond) // the timer fires after Stop
	if got, _ := repo.memRepo.GetByID(bg, "w"); got.Status != domain.StatusWaiting {
		t.Errorf("status = %s: a stopped engine must not queue the task again", got.Status)
	}
}

func TestHoldUntilReset_ATimerThatCannotReadTheTasksLogsAndLeavesThemWaiting(t *testing.T) {
	defer services.SetWaitGrace(0)()
	llm := &scriptLLM{name: "p", alive: true}
	repo := newFaultyRepo()
	orch := services.NewOrchestrator(services.NewDiscoveryService(llm), repo, &noopWriter{}, nil, services.WithDisableBackgroundWorkers())
	t.Cleanup(orch.Stop)
	seedTask(t, repo, "w", domain.StatusProcessing)
	repo.set("GetAll", true)
	hold(t, orch, llm, "w", 40*time.Millisecond)
	time.Sleep(200 * time.Millisecond)
	if got, _ := repo.memRepo.GetByID(bg, "w"); got.Status != domain.StatusWaiting {
		t.Errorf("status = %s, want it left WAITING when the resume failed", got.Status)
	}
}

func TestWatchdog_LogsAFailedResumeAndLeavesATaskThatIsStillRunningAlone(t *testing.T) {
	repo := newFaultyRepo("GetAll")
	blocking := newBlockingLLM()
	orch := services.NewOrchestrator(services.NewDiscoveryService(blocking), repo, &noopWriter{}, nil,
		services.WithWatchdogInterval(10*time.Millisecond), services.WithCleanupInterval(time.Hour))
	t.Cleanup(orch.Stop)
	time.Sleep(60 * time.Millisecond) // ticks whose resume step fails
	repo.set("GetAll", false)

	id, err := orch.SubmitTask(domain.Task{ProjectPath: "/p", Instruction: "i"})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, blocking.started, "the worker to start the task")
	repo.setStale(domain.Task{ID: id, ProjectPath: "/p", Status: domain.StatusProcessing})
	time.Sleep(100 * time.Millisecond) // the watchdog sees the stale entry and must skip it: it is running here
	if got, _ := repo.memRepo.GetByID(bg, id); got.Status != domain.StatusProcessing {
		t.Errorf("status = %s: the watchdog failed a task this engine is still running", got.Status)
	}
	_ = orch.CancelTask(id)
}

// cancelOnSession cancels the task being prepared when its conversation is loaded.
type cancelOnSession struct {
	*memSessionRepo
	cancel func()
}

func (c *cancelOnSession) GetByProjectPath(ctx context.Context, p string) (domain.Session, error) {
	c.cancel()
	return c.memSessionRepo.GetByProjectPath(ctx, p)
}

func TestExecuteWithFallbackChain_DoesNotStartWhenTheTaskWasCancelledWhileItWasBeingPrepared(t *testing.T) {
	var e *engineFixture
	var id string
	sessions := &cancelOnSession{memSessionRepo: newMemSessionRepo(), cancel: func() { _ = e.orch.CancelTask(id) }}
	llm := &scriptLLM{name: "p", alive: true, replies: []string{"never used"}}
	e = newEngine(t, &noopWriter{}, sessions, nil, llm)
	id = e.queue(domain.Task{Instruction: "cancelled while preparing"})
	e.runOne(id, domain.StatusCancelled)
	if llm.calls != 0 {
		t.Errorf("the model was asked %d times", llm.calls)
	}
}

func TestWithBrainOption_GivesTheEngineTheKnowledgeStoreItRemembersFinishedTasksIn(t *testing.T) {
	root := projectWith(t, `{"remember":true}`)
	know := &mockKnowledgeRepo{}
	repo := newFaultyRepo()
	orch := services.NewOrchestrator(services.NewDiscoveryService(), repo, &noopWriter{}, nil,
		services.WithDisableBackgroundWorkers(), services.WithBrain(services.NewBrainService(know, nil)))
	t.Cleanup(orch.Stop)
	if err := repo.memRepo.Save(bg, domain.Task{ID: "done1", ProjectPath: root, Instruction: "ship", Status: domain.StatusCompleted, CreatedAt: time.Now(), UpdatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	orch.RecordOutcome("done1", domain.StatusCompleted)
	if len(know.entries) != 1 {
		t.Errorf("entries = %+v, want one memory of the finished task", know.entries)
	}
}

func TestResolveChain_ALiveProviderWhoseModelListFailsIsLeftOutOfAModelRequest(t *testing.T) {
	broken := &scriptLLM{name: "broken", alive: true, model: "small", modelErr: errInjected}
	serving := &scriptLLM{name: "serving", alive: true, model: "big", models: []string{"big"}}
	orch := services.NewOrchestrator(services.NewDiscoveryService(broken, serving), newFaultyRepo(), &noopWriter{}, nil, services.WithDisableBackgroundWorkers())
	t.Cleanup(orch.Stop)
	got, err := orch.ResolveChain(domain.Task{ModelID: "big"})
	if err != nil || len(got) != 1 || got[0] != "serving" {
		t.Errorf("chain = %v, err = %v; want only the provider that serves the model", got, err)
	}
}
