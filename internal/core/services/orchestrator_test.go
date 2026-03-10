package services_test

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

// --- In-memory stubs ----------------------------------------------------------

type memRepo struct {
	mu    sync.Mutex
	tasks map[string]domain.Task
}

func newMemRepo() *memRepo { return &memRepo{tasks: make(map[string]domain.Task)} }

func (r *memRepo) Save(t domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tasks[t.ID] = t
	return nil
}

func (r *memRepo) GetByID(id string) (domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tasks[id]
	if !ok {
		return domain.Task{}, errors.New("not found")
	}
	return t, nil
}

func (r *memRepo) GetPending() ([]domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.Task
	for _, t := range r.tasks {
		if t.Status == domain.StatusQueued || t.Status == domain.StatusProcessing {
			out = append(out, t)
		}
	}
	return out, nil
}

func (r *memRepo) UpdateStatus(id string, status domain.TaskStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tasks[id]
	if !ok {
		return errors.New("not found")
	}
	t.Status = status
	r.tasks[id] = t
	return nil
}

func (r *memRepo) UpdateLogs(id, logs string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tasks[id]
	if !ok {
		return errors.New("not found")
	}
	t.Logs = logs
	r.tasks[id] = t
	return nil
}

func (r *memRepo) GetByProjectPath(projectPath string) ([]domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.Task
	for _, t := range r.tasks {
		if t.ProjectPath == projectPath {
			out = append(out, t)
		}
	}
	return out, nil
}

type noopWriter struct{}

func (w *noopWriter) WriteCodeToFile(_, _, _ string) error                  { return nil }
func (w *noopWriter) ReadContextFiles(_ string, _ []string) (string, error) { return "", nil }

// --- Tests --------------------------------------------------------------------

func TestOrchestrator_SubmitTask_AssignsIDAndQueues(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService() // no providers — worker won't process
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	task := domain.Task{
		ProjectPath: "/tmp/project",
		TargetFile:  "main.go",
		Instruction: "write hello world",
	}

	id, err := orch.SubmitTask(task)
	if err != nil {
		t.Fatalf("SubmitTask returned error: %v", err)
	}
	if id == "" {
		t.Fatal("expected a non-empty task ID")
	}

	saved, err := repo.GetByID(id)
	if err != nil {
		t.Fatalf("task not persisted: %v", err)
	}
	if saved.Status != domain.StatusQueued {
		t.Errorf("expected status QUEUED, got %s", saved.Status)
	}
}

func TestOrchestrator_GetQueue_ReturnsPendingTasks(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	for i := 0; i < 3; i++ {
		_, err := orch.SubmitTask(domain.Task{Instruction: "task"})
		if err != nil {
			t.Fatalf("SubmitTask: %v", err)
		}
	}

	tasks, err := orch.GetQueue()
	if err != nil {
		t.Fatalf("GetQueue: %v", err)
	}
	if len(tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(tasks))
	}
}

func TestOrchestrator_CancelTask_RemovesFromQueue(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, _ := orch.SubmitTask(domain.Task{Instruction: "to cancel"})

	if err := orch.CancelTask(id); err != nil {
		t.Fatalf("CancelTask: %v", err)
	}

	tasks, _ := orch.GetQueue()
	for _, task := range tasks {
		if task.ID == id {
			t.Error("cancelled task still present in queue")
		}
	}
}

func TestOrchestrator_WorkerProcessesTask(t *testing.T) {
	repo := newMemRepo()
	llm := &mockLLMClient{alive: true, name: "mock", code: "package main"}
	discovery := services.NewDiscoveryService(llm)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, _ := orch.SubmitTask(domain.Task{Instruction: "write code"})

	// Allow the worker loop (2 s ticker) to fire
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(300 * time.Millisecond)
		saved, _ := repo.GetByID(id)
		if saved.Status == domain.StatusCompleted {
			return // success
		}
	}
	t.Errorf("task did not reach COMPLETED within timeout; last status from repo")
}

// --- Session isolation stubs -------------------------------------------------

type memSessionRepo struct {
	mu       sync.Mutex
	sessions map[string][]domain.Message // keyed by projectPath
}

func newMemSessionRepo() *memSessionRepo {
	return &memSessionRepo{sessions: make(map[string][]domain.Message)}
}

func (r *memSessionRepo) Save(s domain.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[s.ProjectPath] = s.Messages
	return nil
}

func (r *memSessionRepo) GetByProjectPath(p string) (domain.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	msgs, ok := r.sessions[p]
	if !ok {
		return domain.Session{}, domain.ErrNotFound
	}
	return domain.Session{ProjectPath: p, Messages: msgs}, nil
}

func (r *memSessionRepo) AppendMessage(p string, msg domain.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[p] = append(r.sessions[p], msg)
	return nil
}

// chatTrackingLLM wraps mockLLMClient and records Chat calls.
type chatTrackingLLM struct {
	mu sync.Mutex
	mockLLMClient
	chatCalled int
	lastMsgs   []domain.Message
}

func (c *chatTrackingLLM) Chat(msgs []domain.Message) (string, error) {
	c.mu.Lock()
	c.chatCalled++
	c.lastMsgs = msgs
	c.mu.Unlock()
	return c.mockLLMClient.Chat(msgs)
}

// --- Session isolation tests -------------------------------------------------

func TestOrchestrator_Session_UsesChatWhenRepoProvided(t *testing.T) {
	repo := newMemRepo()
	llm := &chatTrackingLLM{mockLLMClient: mockLLMClient{alive: true, name: "mock", code: "result"}}
	discovery := services.NewDiscoveryService(llm)
	sessRepo := newMemSessionRepo()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, sessRepo)
	defer orch.Stop()

	id, _ := orch.SubmitTask(domain.Task{ProjectPath: "/proj/a", Instruction: "do something"})

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(300 * time.Millisecond)
		saved, _ := repo.GetByID(id)
		if saved.Status == domain.StatusCompleted {
			llm.mu.Lock()
			called := llm.chatCalled
			llm.mu.Unlock()
			if called == 0 {
				t.Error("expected Chat() to be called when sessionRepo is provided")
			}
			return
		}
	}
	t.Errorf("task did not reach COMPLETED within timeout")
}

func TestOrchestrator_Session_HistoryAccumulatedPerProject(t *testing.T) {
	repo := newMemRepo()
	llm := &chatTrackingLLM{mockLLMClient: mockLLMClient{alive: true, name: "mock", code: "reply"}}
	discovery := services.NewDiscoveryService(llm)
	sessRepo := newMemSessionRepo()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, sessRepo)
	defer orch.Stop()

	path := "/proj/history"

	// Submit two sequential tasks for the same path.
	id1, _ := orch.SubmitTask(domain.Task{ProjectPath: path, Instruction: "first"})
	waitCompleted(t, repo, id1, 10*time.Second)

	id2, _ := orch.SubmitTask(domain.Task{ProjectPath: path, Instruction: "second"})
	waitCompleted(t, repo, id2, 10*time.Second)

	sess, err := sessRepo.GetByProjectPath(path)
	if err != nil {
		t.Fatalf("GetByProjectPath: %v", err)
	}
	// 4 messages: user+assistant for each task.
	if len(sess.Messages) != 4 {
		t.Errorf("expected 4 messages in session history, got %d", len(sess.Messages))
	}
}

func TestOrchestrator_Session_IsolatedByProjectPath(t *testing.T) {
	repo := newMemRepo()
	llm := &chatTrackingLLM{mockLLMClient: mockLLMClient{alive: true, name: "mock", code: "reply"}}
	discovery := services.NewDiscoveryService(llm)
	sessRepo := newMemSessionRepo()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, sessRepo)
	defer orch.Stop()

	idA, _ := orch.SubmitTask(domain.Task{ProjectPath: "/proj/alpha", Instruction: "alpha task"})
	waitCompleted(t, repo, idA, 10*time.Second)

	idB, _ := orch.SubmitTask(domain.Task{ProjectPath: "/proj/beta", Instruction: "beta task"})
	waitCompleted(t, repo, idB, 10*time.Second)

	sessA, _ := sessRepo.GetByProjectPath("/proj/alpha")
	sessB, _ := sessRepo.GetByProjectPath("/proj/beta")

	if len(sessA.Messages) != 2 {
		t.Errorf("alpha: expected 2 messages, got %d", len(sessA.Messages))
	}
	if len(sessB.Messages) != 2 {
		t.Errorf("beta: expected 2 messages, got %d", len(sessB.Messages))
	}
}

func waitCompleted(t *testing.T, repo *memRepo, id string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(300 * time.Millisecond)
		saved, _ := repo.GetByID(id)
		if saved.Status == domain.StatusCompleted {
			return
		}
	}
	t.Fatalf("task %s did not reach COMPLETED within %s", id, timeout)
}

// --- Pre-flight context-window guard tests -----------------------------------

func TestOrchestrator_PreFlight_TooLarge(t *testing.T) {
	repo := newMemRepo()
	// contextLimit=10 means limit-512 = -502; any non-empty instruction overflows
	llm := &mockLLMClient{alive: true, name: "mock", code: "ok", contextLimit: 10}
	discovery := services.NewDiscoveryService(llm)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, err := orch.SubmitTask(domain.Task{
		Instruction: strings.Repeat("x", 200), // ~50 tokens
	})
	if err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
		saved, _ := repo.GetByID(id)
		if saved.Status == domain.StatusTooLarge {
			if !strings.Contains(saved.Logs, "context too large") {
				t.Errorf("expected 'context too large' in Logs, got: %s", saved.Logs)
			}
			return
		}
		if saved.Status == domain.StatusCompleted || saved.Status == domain.StatusFailed {
			t.Fatalf("expected StatusTooLarge but got %s (logs: %s)", saved.Status, saved.Logs)
		}
	}
	t.Fatal("task did not reach StatusTooLarge within timeout")
}

func TestOrchestrator_PreFlight_NoLimitSkipsCheck(t *testing.T) {
	repo := newMemRepo()
	// contextLimit=0 means no pre-flight check; large instruction must still complete
	llm := &mockLLMClient{alive: true, name: "mock", code: "package main", contextLimit: 0}
	discovery := services.NewDiscoveryService(llm)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, _ := orch.SubmitTask(domain.Task{Instruction: strings.Repeat("x", 10000)})
	waitCompleted(t, repo, id, 10*time.Second)
}

func TestOrchestrator_PreFlight_WithinLimit_Completes(t *testing.T) {
	repo := newMemRepo()
	// contextLimit=8192 — "short instruction" is well under 8192-512=7680 tokens
	llm := &mockLLMClient{alive: true, name: "mock", code: "ok", contextLimit: 8192}
	discovery := services.NewDiscoveryService(llm)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, _ := orch.SubmitTask(domain.Task{Instruction: "short instruction"})
	waitCompleted(t, repo, id, 10*time.Second)
}

func TestOrchestrator_StatusNoProvider_WhenModelUnavailable(t *testing.T) {
	repo := newMemRepo()
	// Provider is alive but only has "llama3" — "gpt-4o" will not be found.
	llm := &mockLLMClient{alive: true, name: "Ollama", activeModel: "llama3", models: []string{"llama3"}}
	discovery := services.NewDiscoveryService(llm)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, err := orch.SubmitTask(domain.Task{
		Instruction: "do something",
		ModelID:     "gpt-4o",
	})
	if err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
		saved, _ := repo.GetByID(id)
		if saved.Status == domain.StatusNoProvider {
			return // success
		}
		if saved.Status == domain.StatusCompleted || saved.Status == domain.StatusFailed {
			t.Fatalf("expected StatusNoProvider but got %s", saved.Status)
		}
	}
	t.Fatal("task did not reach StatusNoProvider within timeout")
}

// --- Stop / CancelTask guard tests -------------------------------------------

func TestOrchestrator_CancelTask_UnknownReturnsNotFound(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	err := orch.CancelTask("unknown-task-id")
	if err == nil {
		t.Fatal("expected error for unknown task, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected domain.ErrNotFound, got: %v", err)
	}
}

func TestOrchestrator_Stop_ThenSubmitReturnsError(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)

	orch.Stop()

	_, err := orch.SubmitTask(domain.Task{Instruction: "should fail"})
	if err == nil {
		t.Fatal("expected error after Stop, got nil")
	}
	if !strings.Contains(err.Error(), "stopped") {
		t.Errorf("expected 'stopped' in error message, got: %v", err)
	}
}

func TestOrchestrator_Stop_Idempotent(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)

	// Calling Stop() multiple times must not panic.
	orch.Stop()
	orch.Stop()
	orch.Stop()
}

// --- Provider management tests -----------------------------------------------

func TestOrchestrator_RegisterCloudProvider_NoFactory(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	// No factory wired — must return an error.
	err := orch.RegisterCloudProvider(domain.ProviderConfig{Name: "x", Kind: domain.ProviderKindOllama})
	if err == nil {
		t.Fatal("expected error when no provider factory is configured")
	}
}

func TestOrchestrator_RegisterCloudProvider_FactoryError(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	orch.WithProviderFactory(func(cfg domain.ProviderConfig) (ports.LLMClient, error) {
		return nil, errors.New("unsupported kind")
	})

	err := orch.RegisterCloudProvider(domain.ProviderConfig{Name: "bad", Kind: "unknown"})
	if err == nil {
		t.Fatal("expected error from factory")
	}
}

func TestOrchestrator_RegisterCloudProvider_Success(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	stub := &mockLLMClient{alive: true, name: "new-provider", models: []string{"llama3"}}
	orch.WithProviderFactory(func(cfg domain.ProviderConfig) (ports.LLMClient, error) {
		return stub, nil
	})

	if err := orch.RegisterCloudProvider(domain.ProviderConfig{Name: "new-provider", Kind: domain.ProviderKindOllama}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it was added to discovery.
	providers, err := orch.GetProviders()
	if err != nil {
		t.Fatalf("GetProviders: %v", err)
	}
	found := false
	for _, p := range providers {
		if p.Name == "new-provider" {
			found = true
		}
	}
	if !found {
		t.Error("registered provider not visible via GetProviders()")
	}
}

func TestOrchestrator_RemoveProvider_NotFound(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	err := orch.RemoveProvider("ghost")
	if err == nil {
		t.Fatal("expected error when removing non-existent provider")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected domain.ErrNotFound, got: %v", err)
	}
}

func TestOrchestrator_RemoveProvider_Success(t *testing.T) {
	repo := newMemRepo()
	stub := &mockLLMClient{alive: true, name: "removeme"}
	discovery := services.NewDiscoveryService(stub)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	if err := orch.RemoveProvider("removeme"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	providers, _ := orch.GetProviders()
	for _, p := range providers {
		if p.Name == "removeme" {
			t.Error("provider still present after removal")
		}
	}
}

func TestOrchestrator_GetProviderModels_NotFound(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	_, err := orch.GetProviderModels("ghost")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected domain.ErrNotFound, got: %v", err)
	}
}

func TestOrchestrator_GetProviderModels_Success(t *testing.T) {
	repo := newMemRepo()
	stub := &mockLLMClient{alive: true, name: "mycloud", models: []string{"model-a", "model-b"}}
	discovery := services.NewDiscoveryService(stub)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	models, err := orch.GetProviderModels("mycloud")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0] != "model-a" || models[1] != "model-b" {
		t.Errorf("unexpected models: %v", models)
	}
}

// --- Command-aware routing tests ---------------------------------------------

func TestOrchestrator_CommandPlan_Succeeds(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, err := orch.SubmitTask(domain.Task{
		ProjectPath: "/proj/cmd",
		Instruction: "create plan",
		Command:     domain.CommandPlan,
	})
	if err != nil {
		t.Fatalf("SubmitTask with CommandPlan: %v", err)
	}
	saved, err := repo.GetByID(id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if saved.Command != domain.CommandPlan {
		t.Errorf("expected command %q, got %q", domain.CommandPlan, saved.Command)
	}
}

func TestOrchestrator_CommandExecute_NoPlan_ReturnsErrNoPlan(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	_, err := orch.SubmitTask(domain.Task{
		ProjectPath: "/proj/no-plan",
		Instruction: "execute now",
		Command:     domain.CommandExecute,
	})
	if err == nil {
		t.Fatal("expected ErrNoPlan, got nil")
	}
	if !errors.Is(err, domain.ErrNoPlan) {
		t.Errorf("expected ErrNoPlan, got: %v", err)
	}
}

func TestOrchestrator_CommandExecute_WithPlan_Succeeds(t *testing.T) {
	repo := newMemRepo()
	llm := &mockLLMClient{alive: true, name: "mock", code: "done"}
	discovery := services.NewDiscoveryService(llm)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	// Submit a plan task first
	planID, err := orch.SubmitTask(domain.Task{
		ProjectPath: "/proj/has-plan",
		Instruction: "plan the work",
		Command:     domain.CommandPlan,
	})
	if err != nil {
		t.Fatalf("SubmitTask plan: %v", err)
	}

	// Wait for plan to complete
	waitCompleted(t, repo, planID, 10*time.Second)

	// Now submit execute — should succeed
	execID, err := orch.SubmitTask(domain.Task{
		ProjectPath: "/proj/has-plan",
		Instruction: "do the work",
		Command:     domain.CommandExecute,
	})
	if err != nil {
		t.Fatalf("SubmitTask execute: %v", err)
	}
	if execID == "" {
		t.Fatal("expected non-empty task ID")
	}
}

func TestOrchestrator_InvalidCommand_ReturnsError(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	_, err := orch.SubmitTask(domain.Task{
		ProjectPath: "/proj/bad",
		Instruction: "bad command",
		Command:     domain.CommandType("bogus"),
	})
	if err == nil {
		t.Fatal("expected error for invalid command, got nil")
	}
	if !strings.Contains(err.Error(), "invalid command type") {
		t.Errorf("expected 'invalid command type' in error, got: %v", err)
	}
}

func TestOrchestrator_EmptyCommand_DefaultsToAuto(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	id, err := orch.SubmitTask(domain.Task{
		ProjectPath: "/proj/default",
		Instruction: "auto route",
		Command:     "",
	})
	if err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}
	saved, err := repo.GetByID(id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if saved.Command != domain.CommandAuto {
		t.Errorf("expected command %q, got %q", domain.CommandAuto, saved.Command)
	}
}

func TestOrchestrator_CommandExecute_PlanNotCompleted_ReturnsErrNoPlan(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	// Submit a plan task but DON'T wait for it to complete — it stays QUEUED
	_, err := orch.SubmitTask(domain.Task{
		ProjectPath: "/proj/pending-plan",
		Instruction: "plan the work",
		Command:     domain.CommandPlan,
	})
	if err != nil {
		t.Fatalf("SubmitTask plan: %v", err)
	}

	// Now submit execute — should fail because plan is not completed
	_, err = orch.SubmitTask(domain.Task{
		ProjectPath: "/proj/pending-plan",
		Instruction: "execute now",
		Command:     domain.CommandExecute,
	})
	if err == nil {
		t.Fatal("expected ErrNoPlan, got nil")
	}
	if !errors.Is(err, domain.ErrNoPlan) {
		t.Errorf("expected ErrNoPlan, got: %v", err)
	}
}

func TestOrchestrator_StatusNoProvider_AfterRemove(t *testing.T) {
	repo := newMemRepo()
	// Provider supports llama3. We remove it, then submit a task that needs it.
	stub := &mockLLMClient{alive: true, name: "ollama", activeModel: "llama3", models: []string{"llama3"}}
	discovery := services.NewDiscoveryService(stub)
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil)
	defer orch.Stop()

	if err := orch.RemoveProvider("ollama"); err != nil {
		t.Fatalf("RemoveProvider: %v", err)
	}

	id, err := orch.SubmitTask(domain.Task{Instruction: "do something", ModelID: "llama3"})
	if err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
		saved, _ := repo.GetByID(id)
		if saved.Status == domain.StatusNoProvider {
			return
		}
		if saved.Status == domain.StatusCompleted || saved.Status == domain.StatusFailed {
			t.Fatalf("expected StatusNoProvider but got %s", saved.Status)
		}
	}
	t.Fatal("task did not reach StatusNoProvider within timeout")
}
