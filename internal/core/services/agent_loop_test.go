package services_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

func tool(call string) string { return "```nexus-tool\n" + call + "\n```" }

// memFS is an in-memory project: the sandbox rules live in the real writer, here
// we only need to see what the loop reads and writes.
type memFS struct {
	mu    sync.Mutex
	files map[string]string
	order []string
}

func newMemFS(files map[string]string) *memFS { return &memFS{files: files} }

func (m *memFS) WriteCodeToFile(_, target, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if strings.HasPrefix(target, "/") || strings.Contains(target, "..") {
		return fmt.Errorf("path traversal blocked: %q", target)
	}
	m.files[target] = code
	m.order = append(m.order, target)
	return nil
}

func (m *memFS) ReadContextFiles(_ string, files []string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var sb strings.Builder
	for _, f := range files {
		c, ok := m.files[f]
		if !ok {
			return "", fmt.Errorf("no such file %q", f)
		}
		sb.WriteString(c)
	}
	return sb.String(), nil
}

func agentTask(extra func(*domain.Task)) domain.Task {
	t := domain.Task{Instruction: "agent", Agentic: true, TargetFile: "main.go"}
	if extra != nil {
		extra(&t)
	}
	return t
}

func TestAgentLoop_EditsSeveralFilesThenFinishes(t *testing.T) {
	fs := newMemFS(map[string]string{"go.mod": "module x"})
	llm := &scriptLLM{name: "Anthropic", alive: true, limit: 100000, replies: []string{
		tool(`{"tool":"read_file","path":"go.mod"}`),
		tool(`{"tool":"write_file","path":"a.go","content":"package a"}`) + "\n" + tool(`{"tool":"write_file","path":"b.go","content":"package b"}`),
		tool(`{"tool":"finish","summary":"done"}`),
	}}
	e := newEngine(t, fs, nil, nil, llm)
	id := e.queue(agentTask(nil))
	got := e.runOne(id, domain.StatusCompleted)

	if fs.files["a.go"] != "package a" || fs.files["b.go"] != "package b" {
		t.Errorf("files = %v", fs.files)
	}
	if llm.calls != 3 {
		t.Errorf("model calls = %d, want 3", llm.calls)
	}
	// The second turn must have been shown the file content the model asked for.
	if h := llm.histories[1]; !strings.Contains(h[len(h)-1].Content, "module x") {
		t.Errorf("observation missing file content: %q", h[len(h)-1].Content)
	}
	if !strings.Contains(got.Logs, "2 file(s) written") {
		t.Errorf("logs = %q", got.Logs)
	}
}

func TestAgentLoop_FinishIsRefusedUntilVerificationPasses(t *testing.T) {
	fs := newMemFS(map[string]string{})
	llm := &scriptLLM{name: "Anthropic", alive: true, limit: 100000, replies: []string{
		tool(`{"tool":"write_file","path":"a.go","content":"broken"}`) + tool(`{"tool":"finish","summary":"early"}`),
		tool(`{"tool":"write_file","path":"a.go","content":"fixed"}`) + tool(`{"tool":"finish","summary":"ok"}`),
	}}
	runner := &scriptedRunner{results: []runResult{{"FAIL a_test", errors.New("exit 1")}, {"ok", nil}}}
	e := newEngine(t, fs, nil, runner, llm)
	id := e.queue(agentTask(func(t *domain.Task) { t.VerificationCommand = "go test ./..." }))
	e.runOne(id, domain.StatusCompleted)

	if fs.files["a.go"] != "fixed" || runner.calls != 2 {
		t.Errorf("a.go=%q verification runs=%d", fs.files["a.go"], runner.calls)
	}
	h := llm.histories[1]
	if !strings.Contains(h[len(h)-1].Content, "cannot finish yet") || !strings.Contains(h[len(h)-1].Content, "FAIL a_test") {
		t.Errorf("model was not told why finish failed: %q", h[len(h)-1].Content)
	}
}

func TestAgentLoop_VerifyToolAndMissingCommand(t *testing.T) {
	llm := &scriptLLM{name: "Anthropic", alive: true, limit: 100000, replies: []string{
		tool(`{"tool":"verify"}`),
		tool(`{"tool":"finish","summary":"x"}`),
	}}
	e := newEngine(t, newMemFS(map[string]string{}), nil, nil, llm)
	id := e.queue(agentTask(nil))
	e.runOne(id, domain.StatusCompleted)
	h := llm.histories[1]
	if !strings.Contains(h[len(h)-1].Content, "no verification command") {
		t.Errorf("observation = %q", h[len(h)-1].Content)
	}

	runner := &scriptedRunner{results: []runResult{{"all green", nil}}}
	llm2 := &scriptLLM{name: "Anthropic", alive: true, limit: 100000, replies: []string{
		tool(`{"tool":"verify"}`), tool(`{"tool":"finish"}`),
	}}
	e2 := newEngine(t, newMemFS(map[string]string{}), nil, runner, llm2)
	id2 := e2.queue(agentTask(func(t *domain.Task) { t.VerificationCommand = "make test" }))
	e2.runOne(id2, domain.StatusCompleted)
	if h := llm2.histories[1]; !strings.Contains(h[len(h)-1].Content, "verification passed") {
		t.Errorf("observation = %q", h[len(h)-1].Content)
	}
}

func TestAgentLoop_SandboxErrorsAreReportedToTheModelNotExecuted(t *testing.T) {
	fs := newMemFS(map[string]string{"secret": "s"})
	llm := &scriptLLM{name: "Anthropic", alive: true, limit: 100000, replies: []string{
		tool(`{"tool":"write_file","path":"../../etc/passwd","content":"x"}`) +
			tool(`{"tool":"write_file","path":"/etc/passwd","content":"x"}`) +
			tool(`{"tool":"read_file","path":"missing"}`) +
			tool(`{"tool":"write_file","content":"x"}`) +
			tool(`{"tool":"read_file"}`) +
			tool(`{"tool":"rm_rf","path":"/"}`) +
			"```nexus-tool\n{not json}\n```",
		tool(`{"tool":"finish"}`),
	}}
	e := newEngine(t, fs, nil, nil, llm)
	id := e.queue(agentTask(nil))
	e.runOne(id, domain.StatusCompleted)

	if len(fs.order) != 0 {
		t.Fatalf("a sandbox-violating write went through: %v", fs.order)
	}
	obs := llm.histories[1][len(llm.histories[1])-1].Content
	for _, want := range []string{"path traversal blocked", "no such file", "path is required", `unknown tool "rm_rf"`, "invalid tool call JSON"} {
		if !strings.Contains(obs, want) {
			t.Errorf("observation lacks %q:\n%s", want, obs)
		}
	}
}

func TestAgentLoop_NudgesWhenNoToolCallAndFailsAtTheTurnLimit(t *testing.T) {
	llm := &scriptLLM{name: "Anthropic", alive: true, limit: 100000, replies: []string{"I think the answer is 42."}}
	e := newEngine(t, newMemFS(map[string]string{}), nil, nil, llm)
	id := e.queue(agentTask(func(t *domain.Task) { t.MaxAgentTurns = 3 }))
	got := e.runOne(id, domain.StatusFailed)
	if llm.calls != 3 || !strings.Contains(got.Logs, "stopped after 3 turns") {
		t.Errorf("calls=%d logs=%q", llm.calls, got.Logs)
	}
	if h := llm.histories[1]; !strings.Contains(h[len(h)-1].Content, "No tool call found") {
		t.Errorf("no nudge: %q", h[len(h)-1].Content)
	}
}

func TestAgentLoop_TurnLimitIsClamped(t *testing.T) {
	llm := &scriptLLM{name: "Anthropic", alive: true, limit: 100000, replies: []string{"no tools"}}
	e := newEngine(t, newMemFS(map[string]string{}), nil, nil, llm)
	id := e.queue(agentTask(func(t *domain.Task) { t.MaxAgentTurns = 1000 }))
	e.runOne(id, domain.StatusFailed)
	if llm.calls != 25 {
		t.Errorf("calls = %d, want the hard cap of 25", llm.calls)
	}

	llm2 := &scriptLLM{name: "Anthropic", alive: true, limit: 100000, replies: []string{"no tools"}}
	e2 := newEngine(t, newMemFS(map[string]string{}), nil, nil, llm2)
	id2 := e2.queue(agentTask(nil))
	e2.runOne(id2, domain.StatusFailed)
	if llm2.calls != 8 {
		t.Errorf("calls = %d, want the default of 8", llm2.calls)
	}
}

func TestAgentLoop_OnlyTheFirstCallsOfAReplyRun(t *testing.T) {
	var many strings.Builder
	for i := 0; i < 12; i++ {
		many.WriteString(tool(fmt.Sprintf(`{"tool":"write_file","path":"f%d.go","content":"x"}`, i)))
	}
	fs := newMemFS(map[string]string{})
	llm := &scriptLLM{name: "Anthropic", alive: true, limit: 100000, replies: []string{many.String(), tool(`{"tool":"finish"}`)}}
	e := newEngine(t, fs, nil, nil, llm)
	e.runOne(e.queue(agentTask(nil)), domain.StatusCompleted)
	if len(fs.order) != 8 {
		t.Errorf("ran %d writes, want 8", len(fs.order))
	}
}

func TestAgentLoop_LargeObservationsAreTruncated(t *testing.T) {
	fs := newMemFS(map[string]string{"big": strings.Repeat("x", 40<<10)})
	llm := &scriptLLM{name: "Anthropic", alive: true, limit: 1000000, replies: []string{tool(`{"tool":"read_file","path":"big"}`), tool(`{"tool":"finish"}`)}}
	e := newEngine(t, fs, nil, nil, llm)
	e.runOne(e.queue(agentTask(nil)), domain.StatusCompleted)
	obs := llm.histories[1][len(llm.histories[1])-1].Content
	if len(obs) > 17<<10 || !strings.Contains(obs, "truncated") {
		t.Errorf("observation is %d bytes, not truncated", len(obs))
	}
}

func TestAgentLoop_FailsOverToTheNextProviderMidLoop(t *testing.T) {
	bad := &scriptLLM{name: "Anthropic", alive: true, limit: 100000, errs: []error{errors.New("503")}}
	good := &scriptLLM{name: "Ollama", alive: true, limit: 100000, replies: []string{tool(`{"tool":"finish","summary":"ok"}`)}}
	e := newEngine(t, newMemFS(map[string]string{}), nil, nil, bad, good)
	got := e.runOne(e.queue(agentTask(nil)), domain.StatusCompleted)
	if !strings.Contains(got.Logs, "provider Anthropic failed") || !strings.Contains(got.Logs, "via Ollama") {
		t.Errorf("logs = %q", got.Logs)
	}
}

func TestAgentLoop_AllProvidersFailing(t *testing.T) {
	bad := &scriptLLM{name: "Anthropic", alive: true, limit: 100000, errs: []error{errors.New("503"), errors.New("503")}}
	e := newEngine(t, newMemFS(map[string]string{}), nil, nil, bad)
	got := e.runOne(e.queue(agentTask(nil)), domain.StatusFailed)
	if !strings.Contains(got.Logs, "all providers failed") {
		t.Errorf("logs = %q", got.Logs)
	}
}

func TestAgentLoop_TranscriptOutgrowingTheContextWindow(t *testing.T) {
	// A tiny window: the first turn already does not fit, and there is no fallback.
	llm := &scriptLLM{name: "Anthropic", alive: true, limit: 10, replies: []string{"x"}}
	e := newEngine(t, newMemFS(map[string]string{}), nil, nil, llm)
	got := e.runOne(e.queue(agentTask(func(t *domain.Task) { t.Instruction = strings.Repeat("word ", 5000) })), domain.StatusTooLarge)
	if !strings.Contains(got.Logs, "context window") || llm.calls != 0 {
		t.Errorf("calls=%d logs=%q", llm.calls, got.Logs)
	}

	// With a roomier fallback the loop moves on to it.
	small := &scriptLLM{name: "Anthropic", alive: true, limit: 10, replies: []string{"x"}}
	big := &scriptLLM{name: "Ollama", alive: true, limit: 1000000, replies: []string{tool(`{"tool":"finish"}`)}}
	e2 := newEngine(t, newMemFS(map[string]string{}), nil, nil, small, big)
	e2.runOne(e2.queue(agentTask(func(t *domain.Task) { t.Instruction = strings.Repeat("word ", 5000) })), domain.StatusCompleted)
	if small.calls != 0 || big.calls != 1 {
		t.Errorf("small=%d big=%d", small.calls, big.calls)
	}
}

// blockOnChat lets a test cancel a task while the model call is in flight.
type blockOnChat struct {
	scriptLLM
	entered chan struct{}
}

func (b *blockOnChat) Chat(ctx context.Context, _ []domain.Message) (string, error) {
	close(b.entered)
	<-ctx.Done()
	return "", ctx.Err()
}

func TestAgentLoop_CancellingAbortsAnInFlightTurn(t *testing.T) {
	llm := &blockOnChat{scriptLLM: scriptLLM{name: "Anthropic", alive: true, limit: 100000}, entered: make(chan struct{})}
	e := newEngine(t, newMemFS(map[string]string{}), nil, nil, llm)
	id := e.queue(agentTask(nil))
	done := make(chan bool)
	go func() { done <- e.orch.ProcessNext() }()
	<-llm.entered
	if err := e.orch.CancelTask(id); err != nil {
		t.Fatal(err)
	}
	<-done
	if got := e.status(id); got.Status != domain.StatusCancelled {
		t.Errorf("status = %s, want CANCELLED", got.Status)
	}
}

func TestAgentLoop_FilesAreUnavailableWithoutAWriter(t *testing.T) {
	llm := &scriptLLM{name: "Anthropic", alive: true, limit: 100000, replies: []string{
		tool(`{"tool":"read_file","path":"a"}`) + tool(`{"tool":"write_file","path":"a","content":"x"}`),
		tool(`{"tool":"finish"}`),
	}}
	e := newEngine(t, &contextWriter{writeErr: errors.New("disk full"), ctxErr: errors.New("nope")}, nil, nil, llm)
	e.runOne(e.queue(agentTask(nil)), domain.StatusCompleted)
	obs := llm.histories[1][len(llm.histories[1])-1].Content
	if !strings.Contains(obs, "read_file failed: nope") || !strings.Contains(obs, "write_file failed: disk full") {
		t.Errorf("observation = %q", obs)
	}
}

func TestAgentLoop_EditFileReplacesTheOneOccurrence(t *testing.T) {
	fs := newMemFS(map[string]string{"a.go": "package a\nvar x = 1\nvar y = 1\n"})
	llm := &scriptLLM{name: "Anthropic", alive: true, limit: 100000, replies: []string{
		tool(`{"tool":"edit_file","path":"a.go","old":"var x = 1","new":"var x = 2"}`),
		tool(`{"tool":"edit_file","path":"a.go","old":"nope","new":"z"}`) + "\n" +
			tool(`{"tool":"edit_file","path":"missing.go","old":"a","new":"b"}`) + "\n" +
			tool(`{"tool":"edit_file","path":"a.go"}`),
		tool(`{"tool":"finish","summary":"done"}`),
	}}
	e := newEngine(t, fs, nil, nil, llm)
	e.runOne(e.queue(agentTask(nil)), domain.StatusCompleted)

	if got := fs.files["a.go"]; got != "package a\nvar x = 2\nvar y = 1\n" {
		t.Errorf("a.go = %q", got)
	}
	h := llm.histories[2]
	obs := h[len(h)-1].Content
	for _, want := range []string{"not found", "no such file", "path and old are required"} {
		if !strings.Contains(obs, want) {
			t.Errorf("observation missing %q: %q", want, obs)
		}
	}
}

func TestAgentLoop_EditFileRefusesAnAmbiguousMatchAndNeedsAWriter(t *testing.T) {
	fs := newMemFS(map[string]string{"a.go": "x x"})
	llm := &scriptLLM{name: "Anthropic", alive: true, limit: 100000, replies: []string{
		tool(`{"tool":"edit_file","path":"a.go","old":"x","new":"y"}`),
		tool(`{"tool":"finish","summary":"done"}`),
	}}
	e := newEngine(t, fs, nil, nil, llm)
	e.runOne(e.queue(agentTask(nil)), domain.StatusCompleted)
	if fs.files["a.go"] != "x x" {
		t.Errorf("ambiguous edit must not write: %q", fs.files["a.go"])
	}
	if h := llm.histories[1]; !strings.Contains(h[len(h)-1].Content, "occurs 2 times") {
		t.Errorf("observation = %q", h[len(h)-1].Content)
	}
}

func TestAgentLoop_EditFileReportsWriteFailures(t *testing.T) {
	llm := &scriptLLM{name: "Anthropic", alive: true, limit: 100000, replies: []string{
		tool(`{"tool":"edit_file","path":"a.go","old":"x","new":"y"}`), tool(`{"tool":"finish"}`)}}
	e := newEngine(t, &contextWriter{ctx: "// --- a.go ---\nx\n", writeErr: errors.New("disk full")}, nil, nil, llm)
	e.runOne(e.queue(agentTask(nil)), domain.StatusCompleted)
	if h := llm.histories[1]; !strings.Contains(h[len(h)-1].Content, "edit_file failed: disk full") {
		t.Errorf("observation = %q", h[len(h)-1].Content)
	}
}

func TestAgentLoop_ALimitedProviderIsCooledDownAndEveryoneLimitedMakesTheTaskWait(t *testing.T) {
	reset := time.Now().Add(time.Hour)
	only := &scriptLLM{name: "Anthropic", alive: true, limit: 100000, errs: []error{limit("Anthropic", reset)}}
	e := newEngine(t, newMemFS(map[string]string{}), nil, nil, only)
	got := e.runOne(e.queue(agentTask(nil)), domain.StatusWaiting)
	if !strings.Contains(got.Logs, "all providers are cooling down") || got.WaitUntil.Before(reset) {
		t.Errorf("an agentic task waits too: wait=%v logs=%q", got.WaitUntil, got.Logs)
	}
}

func TestAgentLoop_ACooledDownProviderIsSkippedForTheNextOne(t *testing.T) {
	reset := time.Now().Add(time.Hour)
	limited := &scriptLLM{name: "Anthropic", alive: true, limit: 100000, errs: []error{limit("Anthropic", reset)}}
	good := &scriptLLM{name: "Ollama", alive: true, limit: 100000, replies: []string{tool(`{"tool":"finish","summary":"ok"}`)}}
	e := newEngine(t, newMemFS(map[string]string{}), nil, nil, limited, good)
	e.runOne(e.queue(agentTask(nil)), domain.StatusCompleted)
	// The next agentic task does not even try the limited provider.
	e.runOne(e.queue(agentTask(func(t *domain.Task) { t.Instruction = "second" })), domain.StatusCompleted)
	if limited.calls != 1 {
		t.Errorf("a provider in cooldown is not called again: %d", limited.calls)
	}
}

func TestAgentLoop_StepsUpLadderWhenVerifyFails(t *testing.T) {
	opus, sonnet, local := ladderProviders()
	local.limit = 100000
	sonnet.limit = 100000
	opus.limit = 100000
	local.replies = []string{tool(`{"tool":"finish","summary":"tried local"}`)}
	sonnet.replies = []string{tool(`{"tool":"finish","summary":"fixed on sonnet"}`)}
	runner := &scriptedRunner{results: []runResult{
		{"compile error", errors.New("exit 1")},
		{"PASS", nil},
	}}
	e := newEngine(t, newMemFS(map[string]string{}), nil, runner, opus, sonnet, local)
	dir := ladderProject(t, threeRungs)
	id := e.queue(agentTask(func(tk *domain.Task) {
		tk.ProjectPath = dir
		tk.Role = "coder"
		tk.VerificationCommand = "go test"
	}))
	e.runOne(id, domain.StatusCompleted)
	if local.calls != 1 || sonnet.calls != 1 {
		t.Errorf("failed finish verification in agent loop must step up ladder: local=%d sonnet=%d opus=%d", local.calls, sonnet.calls, opus.calls)
	}
}
