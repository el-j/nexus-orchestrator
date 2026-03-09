package services_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"nexus-ai/internal/core/domain"
	"nexus-ai/internal/core/services"
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
