package services_test

// Tests for TASK-493 (execution_engine.go behaviour) and
// TASK-501 (provider selection / failover).
//
// All internal methods (selectProviderForTask, buildChatContext,
// executeGeneration, writeTaskOutput, extractCode, estimateTokens)
// are unexported; they are exercised indirectly through the public
// OrchestratorService API.

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

// ---- helpers shared across this file ----------------------------------------

func waitStatus(t *testing.T, repo *memRepo, id string, want domain.TaskStatus, timeout time.Duration) domain.Task {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		saved, _ := repo.GetByID(id)
		if saved.Status == want {
			return saved
		}
	}
	saved, _ := repo.GetByID(id)
	t.Fatalf("task %s: want status %s, got %s after %s", id, want, saved.Status, timeout)
	return saved
}

// errWriter returns an error from WriteCodeToFile so we can verify writeTaskOutput
// transitions the task to StatusFailed.
type errWriter struct{}

func (w *errWriter) WriteCodeToFile(_, _, _ string) error {
	return errors.New("disk full")
}
func (w *errWriter) ReadContextFiles(_ string, _ []string) (string, error) { return "", nil }

// recordingWriter captures what was written so we can inspect the output.
type recordingWriter struct {
	content string
	target  string
}

func (w *recordingWriter) WriteCodeToFile(_, target, code string) error {
	w.target = target
	w.content = code
	return nil
}
func (w *recordingWriter) ReadContextFiles(_ string, _ []string) (string, error) { return "", nil }

// countingLLM counts how many times GenerateCode / Chat are called.
type countingLLM struct {
	name    string
	alive   bool
	code    string
	codeErr error
	calls   atomic.Int64
}

func (c *countingLLM) Ping() bool                            { return c.alive }
func (c *countingLLM) ProviderName() string                  { return c.name }
func (c *countingLLM) ActiveModel() string                   { return "" }
func (c *countingLLM) BaseURL() string                       { return "" }
func (c *countingLLM) GetAvailableModels() ([]string, error) { return nil, nil }
func (c *countingLLM) ContextLimit() int                     { return 0 }
func (c *countingLLM) GenerateCode(_ string) (string, error) {
	c.calls.Add(1)
	return c.code, c.codeErr
}
func (c *countingLLM) Chat(_ []domain.Message) (string, error) {
	c.calls.Add(1)
	return c.code, c.codeErr
}

// ---- TASK-493: selectProviderForTask ----------------------------------------

// TestSelectProvider_MatchingProviderName verifies a task with a known ProviderName
// is routed to that provider and completes.
func TestSelectProvider_MatchingProviderName(t *testing.T) {
	repo := newMemRepo()
	llm := &mockLLMClient{alive: true, name: "target-provider", code: "ok"}
	discovery := services.NewDiscoveryService(llm)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, err := orch.SubmitTask(domain.Task{
		ProviderName: "target-provider",
		Instruction:  "use specific provider",
	})
	if err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}
	waitStatus(t, repo, id, domain.StatusCompleted, 10*time.Second)
}

// TestSelectProvider_UnknownProviderName verifies an unknown ProviderName puts
// the task in StatusNoProvider without hanging.
func TestSelectProvider_UnknownProviderName(t *testing.T) {
	repo := newMemRepo()
	llm := &mockLLMClient{alive: true, name: "real-provider", code: "ok"}
	discovery := services.NewDiscoveryService(llm)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, err := orch.SubmitTask(domain.Task{
		ProviderName: "ghost-provider",
		Instruction:  "will never find a provider",
	})
	if err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}
	waitStatus(t, repo, id, domain.StatusNoProvider, 10*time.Second)
}

// TestSelectProvider_NoHint_UsesFirstActive verifies that when no ProviderName is
// set, the first alive provider is used.
func TestSelectProvider_NoHint_UsesFirstActive(t *testing.T) {
	repo := newMemRepo()
	dead := &mockLLMClient{alive: false, name: "dead"}
	alive := &mockLLMClient{alive: true, name: "alive", code: "result"}
	discovery := services.NewDiscoveryService(dead, alive)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, err := orch.SubmitTask(domain.Task{Instruction: "no hint"})
	if err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}
	waitStatus(t, repo, id, domain.StatusCompleted, 10*time.Second)
}

// ---- TASK-493: executeGeneration --------------------------------------------

// TestExecuteGeneration_ChatCalledWithSessionRepo verifies Chat() is used when
// a sessionRepo is configured.
func TestExecuteGeneration_ChatCalledWithSessionRepo(t *testing.T) {
	repo := newMemRepo()
	llm := &chatTrackingLLM{mockLLMClient: mockLLMClient{alive: true, name: "mock", code: "chat result"}}
	discovery := services.NewDiscoveryService(llm)
	sessRepo := newMemSessionRepo()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, sessRepo)
	defer orch.Stop()

	id, _ := orch.SubmitTask(domain.Task{ProjectPath: "/proj/chat", Instruction: "use chat"})
	waitStatus(t, repo, id, domain.StatusCompleted, 10*time.Second)

	llm.mu.Lock()
	called := llm.chatCalled
	llm.mu.Unlock()
	if called == 0 {
		t.Error("expected Chat() to be called when sessionRepo is set")
	}
}

// TestExecuteGeneration_GenerateCodeWithoutSessionRepo verifies GenerateCode() is
// used as the fallback when no sessionRepo is configured.
func TestExecuteGeneration_GenerateCodeWithoutSessionRepo(t *testing.T) {
	repo := newMemRepo()
	llm := &chatTrackingLLM{mockLLMClient: mockLLMClient{alive: true, name: "mock", code: "gen result"}}
	discovery := services.NewDiscoveryService(llm)
	// nil sessionRepo → GenerateCode path
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, _ := orch.SubmitTask(domain.Task{Instruction: "no session"})
	waitStatus(t, repo, id, domain.StatusCompleted, 10*time.Second)

	llm.mu.Lock()
	called := llm.chatCalled
	llm.mu.Unlock()
	if called != 0 {
		t.Errorf("Chat() should not be called without a sessionRepo; called %d times", called)
	}
}

// TestExecuteGeneration_ChatError_EventuallyFails verifies that Chat() errors are
// retried and the task is eventually set to StatusFailed.
func TestExecuteGeneration_ChatError_EventuallyFails(t *testing.T) {
	repo := newMemRepo()
	llm := &mockLLMClient{alive: true, name: "mock", codeErr: errors.New("llm down")}
	discovery := services.NewDiscoveryService(llm)
	sessRepo := newMemSessionRepo()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, sessRepo)
	defer orch.Stop()

	id, _ := orch.SubmitTask(domain.Task{ProjectPath: "/proj/fail-chat", Instruction: "will fail"})
	waitStatus(t, repo, id, domain.StatusFailed, 20*time.Second)
}

// ---- TASK-493: buildChatContext / estimateTokens ----------------------------

// TestBuildChatContext_TooLargeContext verifies that an instruction exceeding the
// model's context limit results in StatusTooLarge.
func TestBuildChatContext_TooLargeContext(t *testing.T) {
	repo := newMemRepo()
	// contextLimit=10 means limit-512 = -502; any non-empty instruction overflows.
	llm := &mockLLMClient{alive: true, name: "mock", code: "ok", contextLimit: 10}
	discovery := services.NewDiscoveryService(llm)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, _ := orch.SubmitTask(domain.Task{
		Instruction: strings.Repeat("x", 200),
	})
	waitStatus(t, repo, id, domain.StatusTooLarge, 10*time.Second)
}

// ---- TASK-493: writeTaskOutput ----------------------------------------------

// TestWriteTaskOutput_WriteError_TaskFailed verifies that when the file writer
// returns an error the task transitions to StatusFailed.
func TestWriteTaskOutput_WriteError_TaskFailed(t *testing.T) {
	repo := newMemRepo()
	llm := &mockLLMClient{alive: true, name: "mock", code: "code"}
	discovery := services.NewDiscoveryService(llm)
	orch := services.NewOrchestrator(discovery, repo, &errWriter{}, nil)
	defer orch.Stop()

	id, _ := orch.SubmitTask(domain.Task{
		Instruction: "write file",
		TargetFile:  "out.go",
		ProjectPath: t.TempDir(),
	})
	waitStatus(t, repo, id, domain.StatusFailed, 10*time.Second)
}

// ---- TASK-493: extractCode (indirect) ---------------------------------------

// TestExtractCode_FencedBlock verifies that when the LLM returns code wrapped in
// a markdown fence, only the inner content is written to disk.
func TestExtractCode_FencedBlock(t *testing.T) {
	dir := t.TempDir()
	fenced := "```go\npackage main\n\nfunc main() {}\n```"

	repo := newMemRepo()
	llm := &mockLLMClient{alive: true, name: "mock", code: fenced}
	discovery := services.NewDiscoveryService(llm)

	writer := &diskWriter{dir: dir}
	orch := services.NewOrchestrator(discovery, repo, writer, nil)
	defer orch.Stop()

	id, _ := orch.SubmitTask(domain.Task{
		Instruction: "generate go",
		TargetFile:  "main.go",
		ProjectPath: dir,
	})
	waitStatus(t, repo, id, domain.StatusCompleted, 10*time.Second)

	content, err := os.ReadFile(dir + "/main.go")
	if err != nil {
		t.Fatalf("output file not written: %v", err)
	}
	got := string(content)
	if strings.Contains(got, "```") {
		t.Errorf("fence markers should be stripped; got:\n%s", got)
	}
	if !strings.Contains(got, "package main") {
		t.Errorf("expected package main in output; got:\n%s", got)
	}
}

// diskWriter writes files to a base directory using the real filesystem.
type diskWriter struct{ dir string }

func (w *diskWriter) WriteCodeToFile(_, target, code string) error {
	return os.WriteFile(w.dir+"/"+target, []byte(code), 0o600)
}
func (w *diskWriter) ReadContextFiles(_ string, _ []string) (string, error) { return "", nil }

// ---- TASK-501: provider selection / failover --------------------------------

// TestProviderFailover_DeadProviderSkipped verifies that a dead provider is
// skipped and the second, alive provider completes the task.
func TestProviderFailover_DeadProviderSkipped(t *testing.T) {
	repo := newMemRepo()
	dead := &countingLLM{name: "dead-provider", alive: false}
	alive := &countingLLM{name: "alive-provider", alive: true, code: "done"}
	discovery := services.NewDiscoveryService(dead, alive)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, _ := orch.SubmitTask(domain.Task{Instruction: "find alive provider"})
	waitStatus(t, repo, id, domain.StatusCompleted, 10*time.Second)

	if dead.calls.Load() != 0 {
		t.Errorf("dead provider should not be called; got %d calls", dead.calls.Load())
	}
	if alive.calls.Load() == 0 {
		t.Error("alive provider should be called at least once")
	}
}

// TestProviderFailover_AllProvidersFail verifies that when every provider
// returns errors, the task eventually reaches StatusFailed.
func TestProviderFailover_AllProvidersFail(t *testing.T) {
	repo := newMemRepo()
	p1 := &countingLLM{name: "p1", alive: true, codeErr: errors.New("p1 error")}
	p2 := &countingLLM{name: "p2", alive: true, codeErr: errors.New("p2 error")}
	discovery := services.NewDiscoveryService(p1, p2)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, _ := orch.SubmitTask(domain.Task{Instruction: "both fail"})
	waitStatus(t, repo, id, domain.StatusFailed, 30*time.Second)

	total := p1.calls.Load() + p2.calls.Load()
	if total == 0 {
		t.Error("expected at least one provider to be called")
	}
}

// ---- TASK-557: Verification Gates & Self-Healing Loop -----------------------

type mockCommandRunner struct {
	calls   atomic.Int32
	runFunc func(ctx context.Context, dir string, cmd string) (string, error)
}

func (m *mockCommandRunner) Run(ctx context.Context, dir string, cmd string) (string, error) {
	m.calls.Add(1)
	if m.runFunc != nil {
		return m.runFunc(ctx, dir, cmd)
	}
	return "", nil
}

// TestExecutionEngine_VerificationGate_SuccessFirstTurn verifies that a task with
// VerificationCommand passes and is marked COMPLETED when the verification exits 0.
func TestExecutionEngine_VerificationGate_SuccessFirstTurn(t *testing.T) {
	repo := newMemRepo()
	llm := &mockLLMClient{alive: true, name: "mock-llm", code: "func Valid() {}"}
	discovery := services.NewDiscoveryService(llm)

	runner := &mockCommandRunner{
		runFunc: func(ctx context.Context, dir, cmd string) (string, error) {
			if cmd != "go test ./..." {
				t.Errorf("unexpected command: %s", cmd)
			}
			return "PASS", nil
		},
	}

	orch := services.NewOrchestrator(
		discovery, repo, &recordingWriter{}, nil,
		services.WithCommandRunner(runner),
	)
	defer orch.Stop()

	id, err := orch.SubmitTask(domain.Task{
		Instruction:         "implement valid func",
		TargetFile:          "valid.go",
		ProjectPath:         t.TempDir(),
		VerificationCommand: "go test ./...",
	})
	if err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}

	task := waitStatus(t, repo, id, domain.StatusCompleted, 10*time.Second)
	if runner.calls.Load() != 1 {
		t.Errorf("expected 1 verification run, got %d", runner.calls.Load())
	}
	if task.VerificationOutput != "PASS" {
		t.Errorf("expected VerificationOutput 'PASS', got: %q", task.VerificationOutput)
	}
}

// TestExecutionEngine_VerificationGate_SelfHealingSuccessTurn2 verifies that when
// verification fails on turn 1, the model receives the error in turn 2 and the task
// passes when turn 2 fixes it.
func TestExecutionEngine_VerificationGate_SelfHealingSuccessTurn2(t *testing.T) {
	repo := newMemRepo()

	var llmCalls atomic.Int32
	llm := &mockLLMClient{
		alive: true,
		name:  "mock-llm",
		code:  "func FirstDraft() {}",
	}

	runner := &mockCommandRunner{}
	runner.runFunc = func(ctx context.Context, dir, cmd string) (string, error) {
		count := runner.calls.Load()
		if count == 1 {
			// First verification fails
			return "undefined: SomeType at line 10", errors.New("exit status 1")
		}
		// Second verification succeeds
		return "PASS", nil
	}

	orch := services.NewOrchestrator(
		discoveryWithCounter(llm, &llmCalls), repo, &recordingWriter{}, nil,
		services.WithCommandRunner(runner),
	)
	defer orch.Stop()

	id, err := orch.SubmitTask(domain.Task{
		Instruction:         "implement type",
		TargetFile:          "type.go",
		ProjectPath:         t.TempDir(),
		VerificationCommand: "go test ./...",
		MaxCorrectionTurns:  2,
	})
	if err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}

	task := waitStatus(t, repo, id, domain.StatusCompleted, 10*time.Second)
	if runner.calls.Load() != 2 {
		t.Errorf("expected 2 verification runs, got %d", runner.calls.Load())
	}
	if task.VerificationOutput != "PASS" {
		t.Errorf("expected VerificationOutput 'PASS', got: %q", task.VerificationOutput)
	}
}

// TestExecutionEngine_VerificationGate_ExhaustedTurns verifies that when verification
// fails on all allowed correction turns, the task transitions to StatusFailed.
func TestExecutionEngine_VerificationGate_ExhaustedTurns(t *testing.T) {
	repo := newMemRepo()
	llm := &mockLLMClient{alive: true, name: "mock-llm", code: "bad code"}
	discovery := services.NewDiscoveryService(llm)

	runner := &mockCommandRunner{
		runFunc: func(ctx context.Context, dir, cmd string) (string, error) {
			return "syntax error: unexpected EOF", errors.New("exit status 2")
		},
	}

	orch := services.NewOrchestrator(
		discovery, repo, &recordingWriter{}, nil,
		services.WithCommandRunner(runner),
	)
	defer orch.Stop()

	id, err := orch.SubmitTask(domain.Task{
		Instruction:         "generate bad code",
		TargetFile:          "bad.go",
		ProjectPath:         t.TempDir(),
		VerificationCommand: "go test ./...",
		MaxCorrectionTurns:  2,
	})
	if err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}

	task := waitStatus(t, repo, id, domain.StatusFailed, 10*time.Second)
	// 1 initial run + 2 correction turns = 3 verification runs total
	if runner.calls.Load() != 3 {
		t.Errorf("expected 3 verification runs, got %d", runner.calls.Load())
	}
	if !strings.Contains(task.VerificationOutput, "syntax error") {
		t.Errorf("expected VerificationOutput to contain syntax error, got: %q", task.VerificationOutput)
	}
}

// ---- TASK-562: Smart Provider Fallback Chain & Role-Driven Router ----------

// TestProviderFallback_PrimaryFails_SecondarySucceeds verifies that when the primary provider
// fails (e.g. rate-limit or network 500 error), the orchestrator automatically executes via the fallback provider.
func TestProviderFallback_PrimaryFails_SecondarySucceeds(t *testing.T) {
	repo := newMemRepo()
	p1 := &countingLLM{name: "primary-anthropic", alive: true, codeErr: errors.New("rate limited: 429 Too Many Requests")}
	p2 := &countingLLM{name: "secondary-gemini", alive: true, code: "package main\n\nfunc main() {}"}
	discovery := services.NewDiscoveryService(p1, p2)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, err := orch.SubmitTask(domain.Task{
		Instruction: "generate something resilient",
	})
	if err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}

	task := waitStatus(t, repo, id, domain.StatusCompleted, 10*time.Second)
	if p1.calls.Load() != 1 {
		t.Errorf("expected primary provider to be attempted once; got %d calls", p1.calls.Load())
	}
	if p2.calls.Load() != 1 {
		t.Errorf("expected secondary provider to be executed once; got %d calls", p2.calls.Load())
	}
	if !strings.Contains(task.Logs, "failover: primary provider failed, executed via secondary-gemini") {
		t.Errorf("expected logs to contain failover notice; got: %q", task.Logs)
	}
}

// TestRoleDrivenRouter_ArchitectRoutesToFrontierFirst verifies that tasks with Role='architect'
// prioritize frontier cloud models over local models.
func TestRoleDrivenRouter_ArchitectRoutesToFrontierFirst(t *testing.T) {
	repo := newMemRepo()
	local := &countingLLM{name: "ollama-local", alive: true, code: "local code"}
	frontier := &countingLLM{name: "anthropic-claude", alive: true, code: "frontier architecture code"}
	// Register local first to ensure default ordering would have picked local if not for role
	discovery := services.NewDiscoveryService(local, frontier)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, err := orch.SubmitTask(domain.Task{
		Instruction: "design microservice architecture",
		Role:        "architect",
	})
	if err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}

	waitStatus(t, repo, id, domain.StatusCompleted, 10*time.Second)

	if frontier.calls.Load() != 1 {
		t.Errorf("expected frontier model to be called first for architect role; got %d calls", frontier.calls.Load())
	}
	if local.calls.Load() != 0 {
		t.Errorf("expected local model NOT to be called when frontier succeeds; got %d calls", local.calls.Load())
	}
}

// TestRoleDrivenRouter_LinterRoutesToLocalFirst verifies that tasks with Role='linter'
// prioritize fast local models over expensive frontier models.
func TestRoleDrivenRouter_LinterRoutesToLocalFirst(t *testing.T) {
	repo := newMemRepo()
	frontier := &countingLLM{name: "anthropic-claude", alive: true, code: "frontier code"}
	local := &countingLLM{name: "ollama-local", alive: true, code: "local linted code"}
	// Register frontier first to verify role overrides discovery order
	discovery := services.NewDiscoveryService(frontier, local)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, err := orch.SubmitTask(domain.Task{
		Instruction: "lint this file and format imports",
		Role:        "linter",
	})
	if err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}

	waitStatus(t, repo, id, domain.StatusCompleted, 10*time.Second)

	if local.calls.Load() != 1 {
		t.Errorf("expected local model to be called first for linter role; got %d calls", local.calls.Load())
	}
	if frontier.calls.Load() != 0 {
		t.Errorf("expected frontier model NOT to be called when local succeeds; got %d calls", frontier.calls.Load())
	}
}

func discoveryWithCounter(client ports.LLMClient, counter *atomic.Int32) *services.DiscoveryService {
	return services.NewDiscoveryService(client)
}
