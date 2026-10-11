package services_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// blockingLLM models a slow provider: Chat/GenerateCode block until the caller's
// context is cancelled, exactly like an HTTP request bound to that context.
type blockingLLM struct {
	scriptLLM
	started chan struct{}
	once    sync.Once
	sawDone atomicBool
}

type atomicBool struct {
	mu sync.Mutex
	v  bool
}

func (a *atomicBool) set()      { a.mu.Lock(); a.v = true; a.mu.Unlock() }
func (a *atomicBool) get() bool { a.mu.Lock(); defer a.mu.Unlock(); return a.v }

func newBlockingLLM() *blockingLLM {
	return &blockingLLM{scriptLLM: scriptLLM{name: "Slow", alive: true, model: "m"}, started: make(chan struct{})}
}

func (b *blockingLLM) block(ctx context.Context) (string, error) {
	b.once.Do(func() { close(b.started) })
	<-ctx.Done()
	b.sawDone.set()
	return "", ctx.Err()
}
func (b *blockingLLM) GenerateCode(ctx context.Context, _ string) (string, error) {
	return b.block(ctx)
}
func (b *blockingLLM) Chat(ctx context.Context, _ []domain.Message) (string, error) {
	return b.block(ctx)
}

// runInBackground executes one task and returns a channel closed when ProcessNext returns.
func runInBackground(e *engineFixture) <-chan struct{} {
	done := make(chan struct{})
	go func() { defer close(done); e.orch.ProcessNext() }()
	return done
}

// lastEvent returns the type of the most recent task event, or "".
func lastEvent(e *engineFixture) string {
	evs := e.bcast.taskEvents()
	if len(evs) == 0 {
		return ""
	}
	return string(evs[len(evs)-1].Type)
}

func waitFor(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

func TestCancelTask_AbortsAGenerationInFlight(t *testing.T) {
	llm := newBlockingLLM()
	e := newEngine(t, &contextWriter{}, nil, nil, llm)
	id := e.queue(domain.Task{Instruction: "slow", TargetFile: "a.go"})

	done := runInBackground(e)
	waitFor(t, llm.started, "the provider call to start")
	if got := e.status(id).Status; got != domain.StatusProcessing {
		t.Fatalf("status before cancel = %s", got)
	}

	if err := e.orch.CancelTask(id); err != nil {
		t.Fatalf("CancelTask(PROCESSING) = %v; the UI offers this, it must work", err)
	}
	waitFor(t, done, "the worker to settle")

	if !llm.sawDone.get() {
		t.Error("the provider call must observe the cancelled context")
	}
	got := e.status(id)
	if got.Status != domain.StatusCancelled {
		t.Fatalf("status = %s, want CANCELLED (logs: %s)", got.Status, got.Logs)
	}
	if got.RetryCount != 0 {
		t.Errorf("a cancelled task must not be retried (retryCount=%d)", got.RetryCount)
	}
	if got := lastEvent(e); got != "task.cancelled" {
		t.Errorf("last event = %q, want task.cancelled", got)
	}
}

func TestCancelTask_DoesNotFailOverToTheNextProviderWhenCancelled(t *testing.T) {
	slow := newBlockingLLM()
	spare := &scriptLLM{name: "Spare", alive: true, model: "m"}
	e := newEngine(t, &contextWriter{}, nil, nil, slow, spare)
	id := e.queue(domain.Task{Instruction: "slow", ProviderName: "Slow", TargetFile: "a.go"})

	done := runInBackground(e)
	waitFor(t, slow.started, "the provider call to start")
	if err := e.orch.CancelTask(id); err != nil {
		t.Fatal(err)
	}
	waitFor(t, done, "the worker to settle")
	if spare.calls != 0 {
		t.Errorf("a cancelled task fell through to the next provider (%d calls)", spare.calls)
	}
	if got := e.status(id).Status; got != domain.StatusCancelled {
		t.Errorf("status = %s", got)
	}
}

func TestStop_AbortsInFlightGenerationPromptlyAndRequeuesTheTask(t *testing.T) {
	llm := newBlockingLLM()
	e := newEngine(t, &contextWriter{}, nil, nil, llm)
	id := e.queue(domain.Task{Instruction: "slow", TargetFile: "a.go"})

	done := runInBackground(e)
	waitFor(t, llm.started, "the provider call to start")

	stopped := make(chan struct{})
	go func() { e.orch.Stop(); close(stopped) }()
	waitFor(t, stopped, "Stop to return (it used to wait out the whole provider call)")
	waitFor(t, done, "the worker to settle")

	if got := e.status(id).Status; got != domain.StatusQueued {
		t.Fatalf("a task interrupted by shutdown must go back to the queue, got %s", got)
	}
}

func TestCancelTask_PROCESSINGTaskNotRunByThisEngineIsCancelledDirectly(t *testing.T) {
	e := newEngine(t, &contextWriter{}, nil, nil)
	if err := e.repo.memRepo.Save(context.Background(), domain.Task{ID: "ext", ProjectPath: "/p", Instruction: "x", Status: domain.StatusProcessing}); err != nil {
		t.Fatal(err)
	}
	if err := e.orch.CancelTask("ext"); err != nil {
		t.Fatalf("an externally claimed task must be cancellable: %v", err)
	}
	if got := e.status("ext").Status; got != domain.StatusCancelled {
		t.Errorf("status = %s", got)
	}
	if got := lastEvent(e); got != "task.cancelled" {
		t.Errorf("last event = %q", got)
	}
}

func TestCancelTask_ErrorsAreStillReportedForFinishedAndUnknownTasks(t *testing.T) {
	e := newEngine(t, &contextWriter{}, nil, nil)
	_ = e.repo.memRepo.Save(context.Background(), domain.Task{ID: "done", ProjectPath: "/p", Instruction: "x", Status: domain.StatusCompleted})
	if err := e.orch.CancelTask("done"); err == nil {
		t.Error("cancelling a COMPLETED task must fail")
	}
	if err := e.orch.CancelTask("ghost"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("unknown task: %v", err)
	}
	e.repo.set("UpdateStatusIfCurrent", true)
	if err := e.orch.CancelTask("done"); err == nil {
		t.Error("a repo failure must surface")
	}
}

func TestCancelTask_AfterTheRunHasFinishedThereIsNothingLeftToAbort(t *testing.T) {
	llm := &scriptLLM{name: "Fast", alive: true, model: "m", replies: []string{"ok"}}
	e := newEngine(t, &contextWriter{}, nil, nil, llm)
	id := e.queue(domain.Task{Instruction: "quick", TargetFile: "a.go"})
	e.orch.ProcessNext()
	if err := e.orch.CancelTask(id); err == nil {
		t.Error("a finished task cannot be cancelled")
	}
}

func TestCancelledSelfHealingGenerationIsNotRetriedOrFailed(t *testing.T) {
	// First generation succeeds, verification fails, and the correction turn blocks.
	llm := newBlockingLLM()
	first := &switchLLM{blockingLLM: llm}
	runner := &mockCommandRunner{runFunc: func(context.Context, string, string) (string, error) {
		return "boom", errors.New("tests failed")
	}}
	e := newEngine(t, &contextWriter{}, nil, runner, first)
	id := e.queue(domain.Task{Instruction: "heal", TargetFile: "a.go", VerificationCommand: "go test ./..."})

	done := runInBackground(e)
	waitFor(t, llm.started, "the correction turn to start")
	if err := e.orch.CancelTask(id); err != nil {
		t.Fatal(err)
	}
	waitFor(t, done, "the worker to settle")
	if got := e.status(id).Status; got != domain.StatusCancelled {
		t.Fatalf("status = %s, want CANCELLED", got)
	}
}

// switchLLM answers the first call normally and blocks every later one.
type switchLLM struct {
	*blockingLLM
	mu sync.Mutex
	n  int
}

func (s *switchLLM) take() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.n++
	return s.n == 1
}
func (s *switchLLM) GenerateCode(ctx context.Context, p string) (string, error) {
	if s.take() {
		return "first draft", nil
	}
	return s.blockingLLM.GenerateCode(ctx, p)
}
func (s *switchLLM) Chat(ctx context.Context, h []domain.Message) (string, error) {
	if s.take() {
		return "first draft", nil
	}
	return s.blockingLLM.Chat(ctx, h)
}

// blockingRunner is a verification command that runs until its context ends.
type blockingRunner struct {
	started chan struct{}
	once    sync.Once
}

func (b *blockingRunner) Run(ctx context.Context, _, _ string) (string, error) {
	b.once.Do(func() { close(b.started) })
	<-ctx.Done()
	return "", ctx.Err()
}

func TestCancelTask_AbortsWhateverStageTheTaskIsIn(t *testing.T) {
	code := "```go\npackage a\n```"
	finishCall := tool(`{"tool":"verify"}`)
	for name, tc := range map[string]struct {
		task    domain.Task
		runner  bool
		startOn func(llm *blockingLLM, r *blockingRunner) <-chan struct{}
	}{
		"agent model call": {
			task:    agentTask(func(t *domain.Task) { t.Instruction = "m" }),
			startOn: func(l *blockingLLM, _ *blockingRunner) <-chan struct{} { return l.started },
		},
		"agent verify tool": {
			task:    agentTask(func(t *domain.Task) { t.Instruction = "v"; t.VerificationCommand = "t" }),
			runner:  true,
			startOn: func(_ *blockingLLM, r *blockingRunner) <-chan struct{} { return r.started },
		},
		"first verification": {
			task:    domain.Task{Instruction: "p", TargetFile: "a.go", VerificationCommand: "t"},
			runner:  true,
			startOn: func(_ *blockingLLM, r *blockingRunner) <-chan struct{} { return r.started },
		},
	} {
		t.Run(name, func(t *testing.T) {
			runner := &blockingRunner{started: make(chan struct{})}
			var llm ports.LLMClient
			var blocking *blockingLLM
			if tc.runner {
				llm = &scriptLLM{name: "p", alive: true, limit: 100000, replies: []string{code, finishCall}}
				if tc.task.Agentic {
					llm = &scriptLLM{name: "p", alive: true, limit: 100000, replies: []string{finishCall}}
				}
				blocking = newBlockingLLM()
			} else {
				blocking = newBlockingLLM()
				blocking.limit = 100000
				llm = blocking
			}
			var e *engineFixture
			if tc.runner {
				e = newEngine(t, &contextWriter{}, nil, runner, llm)
			} else {
				e = newEngine(t, &contextWriter{}, nil, nil, llm)
			}
			id := e.queue(tc.task)
			done := runInBackground(e)
			waitFor(t, tc.startOn(blocking, runner), "the stage to start")
			if err := e.orch.CancelTask(id); err != nil {
				t.Fatal(err)
			}
			waitFor(t, done, "the worker to settle")
			if got := e.status(id).Status; got != domain.StatusCancelled {
				t.Errorf("status = %s, want CANCELLED", got)
			}
		})
	}
}

// A daemon shutdown (not a user cancel) puts the task back in the queue, and a
// failing re-queue write is only logged.
func TestStop_RequeuesAnInterruptedTaskEvenWhenTheWriteFails(t *testing.T) {
	for _, fail := range []bool{false, true} {
		llm := newBlockingLLM()
		e := newEngine(t, &contextWriter{}, nil, nil, llm)
		id := e.queue(domain.Task{Instruction: "slow", TargetFile: "a.go"})
		e.repo.set("UpdateStatusIfCurrent", fail)
		done := runInBackground(e)
		waitFor(t, llm.started, "the provider call to start")
		e.orch.Stop()
		waitFor(t, done, "the worker to settle")
		want := domain.StatusQueued
		if fail {
			want = domain.StatusProcessing
		}
		if got := e.status(id).Status; got != want {
			t.Errorf("fail=%v: status = %s, want %s", fail, got, want)
		}
	}
}
