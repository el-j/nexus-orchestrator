package services_test

// Store failures during task execution are logged and never abort the run:
// the engine still ends in a terminal state.

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

var errStore = errors.New("store unavailable")

// flakyRepo fails log and status writes while counting the attempts.
type flakyRepo struct {
	*memRepo
	statusCalls atomic.Int32
}

func (r *flakyRepo) UpdateLogs(context.Context, string, string) error { return errStore }
func (r *flakyRepo) UpdateStatus(context.Context, string, domain.TaskStatus) error {
	r.statusCalls.Add(1)
	return errStore
}

// flakySession succeeds on reads but fails every append.
type flakySession struct {
	*memSessionRepo
	appends atomic.Int32
}

func (s *flakySession) AppendMessage(context.Context, string, domain.Message) error {
	s.appends.Add(1)
	return errStore
}

func TestExecution_FailedGenerationSurvivesLogAndStatusWriteErrors(t *testing.T) {
	for name, sess := range map[string]ports.SessionRepository{"chat": newMemSessionRepo(), "generate": nil} {
		t.Run(name, func(t *testing.T) {
			repo := &flakyRepo{memRepo: newMemRepo()}
			llm := &mockLLMClient{alive: true, name: "mock", codeErr: errors.New("llm down")}
			orch := services.NewOrchestrator(services.NewDiscoveryService(llm), repo, &noopWriter{}, sess,
				services.WithMaxRetries(0))
			defer orch.Stop()

			if _, err := orch.SubmitTask(domain.Task{ProjectPath: "/p", Instruction: "x"}); err != nil {
				t.Fatal(err)
			}
			eventually(t, "failure status writes", func() bool { return repo.statusCalls.Load() >= 1 })
		})
	}
}

func TestExecution_SuccessSurvivesSessionAppendErrors(t *testing.T) {
	repo := newMemRepo()
	sess := &flakySession{memSessionRepo: newMemSessionRepo()}
	llm := &mockLLMClient{alive: true, name: "mock", code: "ok"}
	orch := services.NewOrchestrator(services.NewDiscoveryService(llm), repo, &noopWriter{}, sess)
	defer orch.Stop()

	id, _ := orch.SubmitTask(domain.Task{ProjectPath: "/p", Instruction: "x"})
	waitStatus(t, repo, id, domain.StatusCompleted, 10*time.Second)
	if sess.appends.Load() < 2 {
		t.Errorf("expected user and assistant appends to be attempted, got %d", sess.appends.Load())
	}
}

// scriptedLLM answers each call from a script; calls past the end repeat the last entry.
type scriptedLLM struct {
	mockLLMClient
	calls  atomic.Int32
	script []func() (string, error)
}

func newScriptedLLM(script ...func() (string, error)) *scriptedLLM {
	return &scriptedLLM{mockLLMClient: mockLLMClient{alive: true, name: "scripted"}, script: script}
}

func (l *scriptedLLM) next() (string, error) {
	i := int(l.calls.Add(1)) - 1
	if i >= len(l.script) {
		i = len(l.script) - 1
	}
	return l.script[i]()
}
func (l *scriptedLLM) GenerateCode(context.Context, string) (string, error)   { return l.next() }
func (l *scriptedLLM) Chat(context.Context, []domain.Message) (string, error) { return l.next() }

func answer(code string) func() (string, error) { return func() (string, error) { return code, nil } }
func fail(msg string) func() (string, error) {
	return func() (string, error) { return "", errors.New(msg) }
}

// failThenPass fails the first verification run and passes the rest.
func failThenPass() *mockCommandRunner {
	r := &mockCommandRunner{}
	r.runFunc = func(context.Context, string, string) (string, error) {
		if r.calls.Load() == 1 {
			return "boom", errors.New("exit status 1")
		}
		return "PASS", nil
	}
	return r
}

func verifiedTask() domain.Task {
	return domain.Task{
		ProjectPath: "/p", TargetFile: "f.go", Instruction: "x",
		VerificationCommand: "go test ./...", MaxCorrectionTurns: 2,
	}
}

// A correction turn that fails leaves the task FAILED even when the store rejects the log and
// status writes.
func TestSelfHealing_FailedCorrectionTurnSurvivesStoreErrors(t *testing.T) {
	for name, sess := range map[string]ports.SessionRepository{"chat": newMemSessionRepo(), "generate": nil} {
		t.Run(name, func(t *testing.T) {
			repo := &flakyRepo{memRepo: newMemRepo()}
			llm := newScriptedLLM(answer("draft"), fail("llm down"))
			orch := services.NewOrchestrator(services.NewDiscoveryService(llm), repo, &noopWriter{}, sess,
				services.WithMaxRetries(0), services.WithCommandRunner(failThenPass()))
			defer orch.Stop()

			orch.SubmitTask(verifiedTask())
			eventually(t, "failed status write", func() bool { return repo.statusCalls.Load() >= 1 })
		})
	}
}

// A failing correction turn is retried when attempts remain.
func TestSelfHealing_FailedCorrectionTurnRequeuesWhenRetriesRemain(t *testing.T) {
	for name, sess := range map[string]ports.SessionRepository{"chat": newMemSessionRepo(), "generate": nil} {
		t.Run(name, func(t *testing.T) {
			repo := newMemRepo()
			llm := newScriptedLLM(answer("draft"), fail("llm down"))
			orch := services.NewOrchestrator(services.NewDiscoveryService(llm), repo, &noopWriter{}, sess,
				services.WithMaxRetries(1), services.WithCommandRunner(failThenPass()))
			defer orch.Stop()

			id, _ := orch.SubmitTask(verifiedTask())
			eventually(t, "a retry", func() bool {
				got, _ := repo.GetByID(context.Background(), id)
				return got.RetryCount >= 1
			})
		})
	}
}

// Session append errors never block a corrected result.
func TestSelfHealing_SucceedsDespiteSessionAppendErrors(t *testing.T) {
	repo := newMemRepo()
	sess := &flakySession{memSessionRepo: newMemSessionRepo()}
	llm := newScriptedLLM(answer("draft"), answer("fixed"))
	orch := services.NewOrchestrator(services.NewDiscoveryService(llm), repo, &noopWriter{}, sess,
		services.WithCommandRunner(failThenPass()))
	defer orch.Stop()

	id, _ := orch.SubmitTask(verifiedTask())
	waitStatus(t, repo, id, domain.StatusCompleted, 10*time.Second)
	if sess.appends.Load() < 2 {
		t.Errorf("expected appends for the final prompt and corrected reply, got %d", sess.appends.Load())
	}
}

// Cancelling during a correction turn settles the task as CANCELLED even when the status write
// fails.
func TestSelfHealing_CancelDuringCorrectionTurn(t *testing.T) {
	repo := &flakyRepo{memRepo: newMemRepo()}
	var orch atomic.Pointer[services.OrchestratorService]
	var taskID atomic.Value
	ready := make(chan struct{})
	llm := newScriptedLLM(answer("draft"), func() (string, error) {
		<-ready
		_ = orch.Load().CancelTask(taskID.Load().(string))
		time.Sleep(50 * time.Millisecond)
		return "", errors.New("aborted")
	})
	o := services.NewOrchestrator(services.NewDiscoveryService(llm), repo, &noopWriter{}, newMemSessionRepo(),
		services.WithCommandRunner(failThenPass()))
	orch.Store(o)
	defer o.Stop()

	id, _ := o.SubmitTask(verifiedTask())
	taskID.Store(id)
	close(ready)
	eventually(t, "cancelled status write", func() bool { return repo.statusCalls.Load() >= 1 })
}
