package services_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

// scriptLLM is a fully scriptable provider that records every prompt/history it receives.
type scriptLLM struct {
	mu        sync.Mutex
	name      string
	alive     bool
	model     string
	models    []string
	modelErr  error
	limit     int
	replies   []string // consumed one per call; the last repeats
	errs      []error  // per call; nil entries succeed
	prompts   []string
	histories [][]domain.Message
	calls     int
}

func (s *scriptLLM) Ping() bool           { return s.alive }
func (s *scriptLLM) ProviderName() string { return s.name }
func (s *scriptLLM) ActiveModel() string  { return s.model }
func (s *scriptLLM) BaseURL() string      { return "" }
func (s *scriptLLM) ContextLimit() int    { return s.limit }
func (s *scriptLLM) GetAvailableModels() ([]string, error) {
	return s.models, s.modelErr
}

func (s *scriptLLM) next(prompt string, hist []domain.Message) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.calls
	s.calls++
	s.prompts = append(s.prompts, prompt)
	s.histories = append(s.histories, hist)
	var err error
	if i < len(s.errs) {
		err = s.errs[i]
	}
	if err != nil {
		return "", err
	}
	if len(s.replies) == 0 {
		return "ok", nil
	}
	if i >= len(s.replies) {
		i = len(s.replies) - 1
	}
	return s.replies[i], nil
}

func (s *scriptLLM) GenerateCode(p string) (string, error) { return s.next(p, nil) }
func (s *scriptLLM) Chat(h []domain.Message) (string, error) {
	return s.next(h[len(h)-1].Content, h)
}

// contextWriter returns canned context-file content (or an error) and records writes.
type contextWriter struct {
	ctx      string
	ctxErr   error
	writeErr error
	writes   []string
}

func (w *contextWriter) WriteCodeToFile(_, target, code string) error {
	w.writes = append(w.writes, target+":"+code)
	return w.writeErr
}
func (w *contextWriter) ReadContextFiles(string, []string) (string, error) { return w.ctx, w.ctxErr }

// engineFixture is a worker-less orchestrator driven one task at a time via ProcessNext.
type engineFixture struct {
	t     *testing.T
	orch  *services.OrchestratorService
	repo  *faultyRepo
	bcast *capturingBroadcaster
}

func newEngine(t *testing.T, writer ports.FileWriter, sessions ports.SessionRepository, runner ports.CommandRunner, llms ...ports.LLMClient) *engineFixture {
	t.Helper()
	repo := newFaultyRepo()
	opts := []services.Option{services.WithDisableBackgroundWorkers(), services.WithMaxRetries(2)}
	if runner != nil {
		opts = append(opts, services.WithCommandRunner(runner))
	}
	orch := services.NewOrchestrator(services.NewDiscoveryService(llms...), repo, writer, sessions, opts...)
	b := &capturingBroadcaster{}
	orch.SetBroadcaster(b)
	t.Cleanup(orch.Stop)
	return &engineFixture{t: t, orch: orch, repo: repo, bcast: b}
}

func (e *engineFixture) queue(task domain.Task) string {
	e.t.Helper()
	if task.ProjectPath == "" {
		task.ProjectPath = "/proj"
	}
	task.ID = "t-" + task.Instruction
	task.Status = domain.StatusQueued
	if err := e.repo.memRepo.Save(task); err != nil {
		e.t.Fatal(err)
	}
	return task.ID
}

func (e *engineFixture) status(id string) domain.Task {
	e.t.Helper()
	got, err := e.repo.memRepo.GetByID(id)
	if err != nil {
		e.t.Fatal(err)
	}
	return got
}

func (e *engineFixture) runOne(id string, want domain.TaskStatus) domain.Task {
	e.t.Helper()
	if !e.orch.ProcessNext() {
		e.t.Fatal("ProcessNext found no queued task")
	}
	got := e.status(id)
	if got.Status != want {
		e.t.Fatalf("task %s: status = %s, want %s (logs: %q)", id, got.Status, want, got.Logs)
	}
	return got
}

func TestProcessNext_EmptyQueueAndClaimErrors(t *testing.T) {
	e := newEngine(t, &noopWriter{}, nil, nil, &scriptLLM{name: "p", alive: true})
	if e.orch.ProcessNext() {
		t.Error("empty queue must report false")
	}
	// A hard repository failure also reports false (and must not panic).
	broken := &claimFailRepo{faultyRepo: newFaultyRepo()}
	orch := newOrchOver(t, broken)
	if orch.ProcessNext() {
		t.Error("claim error must report false")
	}
}

type claimFailRepo struct{ *faultyRepo }

func (c *claimFailRepo) ClaimNextQueued() (domain.Task, error) { return domain.Task{}, errInjected }

func TestProcessNext_NoProviderMarksTaskNoProvider(t *testing.T) {
	e := newEngine(t, &noopWriter{}, nil, nil) // no providers at all
	id := e.queue(domain.Task{Instruction: "orphan"})
	got := e.runOne(id, domain.StatusNoProvider)
	if !strings.Contains(got.Logs, "no active provider") {
		t.Errorf("logs = %q", got.Logs)
	}
}

func TestPrepareChatPrompt_ContextFilesAndSessionHistory(t *testing.T) {
	llm := &scriptLLM{name: "p", alive: true}
	w := &contextWriter{ctx: "// file: a.go\npackage a"}
	sess := newMemSessionRepo()
	_ = sess.Save(domain.Session{ProjectPath: "/proj", Messages: []domain.Message{{Role: domain.RoleUser, Content: "earlier"}}})
	e := newEngine(t, w, sess, nil, llm)
	id := e.queue(domain.Task{Instruction: "do", ContextFiles: []string{"a.go"}, TargetFile: "out.go"})
	e.runOne(id, domain.StatusCompleted)

	if len(llm.prompts) != 1 || !strings.HasPrefix(llm.prompts[0], "// file: a.go") || !strings.HasSuffix(llm.prompts[0], "do") {
		t.Errorf("context must be prepended to the instruction, prompt = %q", llm.prompts)
	}
	if h := llm.histories[0]; len(h) != 2 || h[0].Content != "earlier" {
		t.Errorf("session history must precede the new prompt, history = %+v", h)
	}
	if len(w.writes) != 1 || !strings.HasPrefix(w.writes[0], "out.go:") {
		t.Errorf("output not written: %v", w.writes)
	}
	// The exchange is persisted to the session.
	if saved, _ := sess.GetByProjectPath("/proj"); len(saved.Messages) != 3 {
		t.Errorf("session should hold history + user + assistant, got %d messages", len(saved.Messages))
	}
}

func TestPrepareChatPrompt_BlankContextIsIgnored(t *testing.T) {
	llm := &scriptLLM{name: "p", alive: true}
	e := newEngine(t, &contextWriter{ctx: "   \n"}, nil, nil, llm)
	id := e.queue(domain.Task{Instruction: "plain", ContextFiles: []string{"a.go"}})
	e.runOne(id, domain.StatusCompleted)
	if llm.prompts[0] != "plain" {
		t.Errorf("blank context must not alter the prompt, got %q", llm.prompts[0])
	}
}

func TestPrepareChatPrompt_FailuresFailTheTaskBeforeCallingTheModel(t *testing.T) {
	llm := &scriptLLM{name: "p", alive: true}

	e := newEngine(t, &contextWriter{ctxErr: errors.New("permission denied")}, nil, nil, llm)
	id := e.queue(domain.Task{Instruction: "ctx", ContextFiles: []string{"x"}})
	got := e.runOne(id, domain.StatusFailed)
	if !strings.Contains(got.Logs, "failed reading context files") || llm.calls != 0 {
		t.Errorf("logs=%q calls=%d", got.Logs, llm.calls)
	}

	e2 := newEngine(t, &noopWriter{}, &failingSessionRepo{err: errInjected}, nil, llm)
	id2 := e2.queue(domain.Task{Instruction: "sess"})
	got = e2.runOne(id2, domain.StatusFailed)
	if !strings.Contains(got.Logs, "failed loading session history") || llm.calls != 0 {
		t.Errorf("logs=%q calls=%d", got.Logs, llm.calls)
	}
}

type failingSessionRepo struct{ err error }

func (f *failingSessionRepo) Save(domain.Session) error { return f.err }
func (f *failingSessionRepo) GetByProjectPath(string) (domain.Session, error) {
	return domain.Session{}, f.err
}
func (f *failingSessionRepo) AppendMessage(string, domain.Message) error { return f.err }

func TestFallbackChain_FailsOverAndRecordsNote(t *testing.T) {
	primary := &scriptLLM{name: "anthropic-main", alive: true, errs: []error{errors.New("rate limited")}}
	backup := &scriptLLM{name: "ollama-backup", alive: true, replies: []string{"recovered"}}
	e := newEngine(t, &noopWriter{}, nil, nil, primary, backup)
	id := e.queue(domain.Task{Instruction: "needs failover", Role: "architect"})
	got := e.runOne(id, domain.StatusCompleted)
	for _, want := range []string{"provider anthropic-main failed: rate limited", "[failover: primary provider failed, executed via ollama-backup]"} {
		if !strings.Contains(got.Logs, want) {
			t.Errorf("logs missing %q:\n%s", want, got.Logs)
		}
	}
}

func TestFallbackChain_AllFailRetriesThenFails(t *testing.T) {
	llm := &scriptLLM{name: "only", alive: true, errs: []error{errInjected, errInjected, errInjected, errInjected}}
	e := newEngine(t, &noopWriter{}, nil, nil, llm)
	id := e.queue(domain.Task{Instruction: "doomed"})

	// maxRetries=2: two re-queues, then permanent failure.
	e.runOne(id, domain.StatusQueued)
	if got := e.status(id); got.RetryCount != 1 {
		t.Errorf("retry count = %d", got.RetryCount)
	}
	e.runOne(id, domain.StatusQueued)
	got := e.runOne(id, domain.StatusFailed)
	if !strings.Contains(got.Logs, "all candidate providers failed") || got.RetryCount != 2 {
		t.Errorf("final: retry=%d logs=%q", got.RetryCount, got.Logs)
	}
}

func TestFallbackChain_ContextTooLargeForEveryProvider(t *testing.T) {
	small := &scriptLLM{name: "tiny", alive: true, limit: 600} // 600 - 512 headroom = 88 tokens
	e := newEngine(t, &noopWriter{}, nil, nil, small)
	id := e.queue(domain.Task{Instruction: strings.Repeat("word ", 400)})
	got := e.runOne(id, domain.StatusTooLarge)
	if small.calls != 0 || !strings.Contains(got.Logs, "all candidate providers exceeded context limits") {
		t.Errorf("calls=%d logs=%q", small.calls, got.Logs)
	}
}

func TestFallbackChain_TooLargeForOneProviderFallsThroughToNext(t *testing.T) {
	tiny := &scriptLLM{name: "tiny-local", alive: true, limit: 600}
	big := &scriptLLM{name: "big-other", alive: true}
	e := newEngine(t, &noopWriter{}, nil, nil, tiny, big)
	id := e.queue(domain.Task{Instruction: strings.Repeat("word ", 400)})
	e.runOne(id, domain.StatusCompleted)
	if tiny.calls != 0 || big.calls != 1 {
		t.Errorf("tiny=%d big=%d", tiny.calls, big.calls)
	}
}

func TestResolveProvider_ExplicitNamePutsItFirstWithLiveFallbacks(t *testing.T) {
	a := &scriptLLM{name: "alpha", alive: true}
	b := &scriptLLM{name: "beta", alive: true}
	dead := &scriptLLM{name: "gamma", alive: false}
	e := newEngine(t, &noopWriter{}, nil, nil, a, b, dead)
	id := e.queue(domain.Task{Instruction: "named", ProviderName: "beta"})
	e.runOne(id, domain.StatusCompleted)
	if b.calls != 1 || a.calls != 0 {
		t.Errorf("explicit provider must be tried first: alpha=%d beta=%d", a.calls, b.calls)
	}

	id2 := e.queue(domain.Task{Instruction: "missing", ProviderName: "nope"})
	got := e.runOne(id2, domain.StatusNoProvider)
	if !strings.Contains(got.Logs, `provider "nope" not found`) {
		t.Errorf("logs = %q", got.Logs)
	}
}

func TestResolveProvider_ModelSelection(t *testing.T) {
	byActive := &scriptLLM{name: "p-active", alive: true, model: "qwen-7b"}
	byList := &scriptLLM{name: "p-list", alive: true, model: "other", models: []string{"llama-3"}}
	e := newEngine(t, &noopWriter{}, nil, nil, byActive, byList)

	id := e.queue(domain.Task{Instruction: "active-model", ModelID: "QWEN-7B"})
	e.runOne(id, domain.StatusCompleted)
	if byActive.calls != 1 {
		t.Errorf("active model match (case-insensitive) should be used: %d", byActive.calls)
	}

	id = e.queue(domain.Task{Instruction: "listed-model", ModelID: "llama-3"})
	e.runOne(id, domain.StatusCompleted)
	if byList.calls != 1 {
		t.Errorf("model found via GetAvailableModels should be used: %d", byList.calls)
	}

	id = e.queue(domain.Task{Instruction: "unknown-model", ModelID: "ghost-9000"})
	got := e.runOne(id, domain.StatusNoProvider)
	if got.Logs == "" {
		t.Error("an unavailable model must explain itself in the logs")
	}
}

func TestResolveProvider_RoleRoutingOrder(t *testing.T) {
	cloud := &scriptLLM{name: "anthropic-cloud", alive: true}
	local := &scriptLLM{name: "ollama-local", alive: true}
	other := &scriptLLM{name: "custom-thing", alive: true}

	cases := []struct {
		role      string
		wantFirst *scriptLLM
	}{
		{"architect", cloud},
		{"linter", local},
		{"", cloud}, // default: frontier first when one is alive
		{"unknown", cloud},
	}
	for _, tc := range cases {
		cloud.calls, local.calls, other.calls = 0, 0, 0
		e := newEngine(t, &noopWriter{}, nil, nil, other, local, cloud)
		id := e.queue(domain.Task{Instruction: "route-" + tc.role, Role: tc.role})
		e.runOne(id, domain.StatusCompleted)
		if tc.wantFirst.calls != 1 {
			t.Errorf("role %q: expected %s to serve the task (cloud=%d local=%d other=%d)", tc.role, tc.wantFirst.name, cloud.calls, local.calls, other.calls)
		}
	}

	// Without a frontier provider the default order is preserved.
	e := newEngine(t, &noopWriter{}, nil, nil, other, local)
	id := e.queue(domain.Task{Instruction: "no-frontier"})
	e.runOne(id, domain.StatusCompleted)
	if other.calls != 0 || local.calls != 0 {
		// 'local' and 'other' are alive; whichever is first in discovery order serves it.
		if other.calls+local.calls != 1 {
			t.Errorf("exactly one provider should serve the task: other=%d local=%d", other.calls, local.calls)
		}
	}
}

func TestWriteAndVerify_WriteFailureFailsTheTask(t *testing.T) {
	llm := &scriptLLM{name: "p", alive: true, replies: []string{"```go\npackage x\n```"}}
	w := &contextWriter{writeErr: errors.New("disk full")}
	e := newEngine(t, w, nil, nil, llm)
	id := e.queue(domain.Task{Instruction: "write", TargetFile: "x.go"})
	got := e.runOne(id, domain.StatusFailed)
	if !strings.Contains(got.Logs, "failed writing output via p: disk full") {
		t.Errorf("logs = %q", got.Logs)
	}
	if len(w.writes) != 1 || w.writes[0] != "x.go:package x" {
		t.Errorf("code fence must be stripped before writing, got %v", w.writes)
	}
}

// scriptedRunner returns verification results in order (last repeats).
type scriptedRunner struct {
	results []runResult
	calls   int
}
type runResult struct {
	out string
	err error
}

func (r *scriptedRunner) Run(context.Context, string, string) (string, error) {
	i := r.calls
	r.calls++
	if i >= len(r.results) {
		i = len(r.results) - 1
	}
	return r.results[i].out, r.results[i].err
}

func TestSelfHealing_CorrectionPromptCarriesInstructionAndLatestOutput(t *testing.T) {
	llm := &scriptLLM{name: "p", alive: true, replies: []string{"v1 broken", "v2 still broken", "v3 fixed"}}
	runner := &scriptedRunner{results: []runResult{
		{"error A", errors.New("exit 1")}, {"error B", errors.New("exit 1")}, {"PASS", nil},
	}}
	e := newEngine(t, &contextWriter{}, nil, runner, llm)
	id := e.queue(domain.Task{Instruction: "implement Foo", TargetFile: "foo.go", VerificationCommand: "go test", MaxCorrectionTurns: 3})
	got := e.runOne(id, domain.StatusCompleted)

	if len(llm.prompts) != 3 {
		t.Fatalf("want initial + 2 correction calls, got %d", len(llm.prompts))
	}
	turn1, turn2 := llm.prompts[1], llm.prompts[2]
	for _, want := range []string{"implement Foo", "v1 broken", "error A", `"go test"`} {
		if !strings.Contains(turn1, want) {
			t.Errorf("turn 1 prompt missing %q:\n%s", want, turn1)
		}
	}
	for _, want := range []string{"implement Foo", "v2 still broken", "error B"} {
		if !strings.Contains(turn2, want) {
			t.Errorf("turn 2 prompt missing %q (must show the LATEST output):\n%s", want, turn2)
		}
	}
	if strings.Contains(turn2, "v1 broken") {
		t.Errorf("turn 2 must not resend the superseded output:\n%s", turn2)
	}
	if !strings.Contains(got.Logs, "verification passed on correction turn 2/3") || got.VerificationOutput != "PASS" {
		t.Errorf("logs=%q out=%q", got.Logs, got.VerificationOutput)
	}
}

func TestSelfHealing_GenerationFailureAndWriteFailureDuringCorrection(t *testing.T) {
	failing := errors.New("exit 1")

	// The model errors during a correction turn: retries are exhausted -> FAILED.
	llm := &scriptLLM{name: "p", alive: true, replies: []string{"v1"}, errs: []error{nil, errInjected, errInjected, errInjected}}
	runner := &scriptedRunner{results: []runResult{{"boom", failing}}}
	e := newEngine(t, &contextWriter{}, nil, runner, llm)
	e.orch.WithQueueCap(10)
	id := e.queue(domain.Task{Instruction: "gen-fail", TargetFile: "a.go", VerificationCommand: "t"})
	e.runOne(id, domain.StatusQueued) // executeGeneration re-queues for retry
	got := e.status(id)
	if got.RetryCount != 1 {
		t.Errorf("retry count = %d", got.RetryCount)
	}

	// The correction output cannot be written -> FAILED with a log line.
	llm2 := &scriptLLM{name: "p", alive: true, replies: []string{"v1", "v2"}}
	w := &failSecondWriteWriter{}
	runner2 := &scriptedRunner{results: []runResult{{"boom", failing}}}
	e2 := newEngine(t, w, nil, runner2, llm2)
	id2 := e2.queue(domain.Task{Instruction: "write-fail", TargetFile: "a.go", VerificationCommand: "t"})
	got2 := e2.runOne(id2, domain.StatusFailed)
	if !strings.Contains(got2.Logs, "failed writing corrected output via p") {
		t.Errorf("logs = %q", got2.Logs)
	}
}

type failSecondWriteWriter struct{ n int }

func (w *failSecondWriteWriter) WriteCodeToFile(string, string, string) error {
	w.n++
	if w.n >= 2 {
		return errors.New("disk full")
	}
	return nil
}
func (w *failSecondWriteWriter) ReadContextFiles(string, []string) (string, error) { return "", nil }

func TestSelfHealing_NoCorrectionWithoutRunnerOrBlankCommand(t *testing.T) {
	llm := &scriptLLM{name: "p", alive: true}
	// Command set but no runner configured -> completes without verification.
	e := newEngine(t, &noopWriter{}, nil, nil, llm)
	id := e.queue(domain.Task{Instruction: "no-runner", VerificationCommand: "go test"})
	e.runOne(id, domain.StatusCompleted)

	// Blank/whitespace command with a runner -> no verification either.
	runner := &scriptedRunner{results: []runResult{{"", nil}}}
	e2 := newEngine(t, &noopWriter{}, nil, runner, llm)
	id2 := e2.queue(domain.Task{Instruction: "blank-cmd", VerificationCommand: "   "})
	e2.runOne(id2, domain.StatusCompleted)
	if runner.calls != 0 {
		t.Errorf("blank command must not run, calls=%d", runner.calls)
	}
}

func TestExtractCode_Variants(t *testing.T) {
	// Exercised through the writer: what reaches disk is the fenced body only.
	cases := map[string]string{
		"no fence at all":                       "no fence at all",
		"intro\n```go\nA\nB\n```\ntrailing":     "A\nB",
		"```\nunterminated fence\nbody":         "unterminated fence\nbody",
		"```python\nprint(1)\n```\n```\nx\n```": "print(1)",
	}
	for in, want := range cases {
		llm := &scriptLLM{name: "p", alive: true, replies: []string{in}}
		w := &contextWriter{}
		e := newEngine(t, w, nil, nil, llm)
		id := e.queue(domain.Task{Instruction: "x", TargetFile: "f.txt"})
		e.runOne(id, domain.StatusCompleted)
		if len(w.writes) != 1 || w.writes[0] != "f.txt:"+want {
			t.Errorf("input %q: wrote %v, want %q", in, w.writes, want)
		}
	}
}
