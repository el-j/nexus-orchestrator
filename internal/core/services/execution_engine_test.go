package services_test

// Tests for TASK-493 (execution_engine.go behaviour) and
// TASK-501 (provider selection / failover).
//
// All internal methods (selectProviderForTask, buildChatContext,
// executeGeneration, writeTaskOutput, extractCode, estimateTokens)
// are unexported; they are exercised indirectly through the public
// OrchestratorService API.

import (
	"errors"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
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
