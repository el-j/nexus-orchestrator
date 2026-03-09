package services_test

import (
	"errors"
	"testing"
	"time"

	"nexus-ai/internal/core/domain"
	"nexus-ai/internal/core/services"
)

// --- In-memory stubs ----------------------------------------------------------

type memRepo struct {
	tasks map[string]domain.Task
}

func newMemRepo() *memRepo { return &memRepo{tasks: make(map[string]domain.Task)} }

func (r *memRepo) Save(t domain.Task) error {
	r.tasks[t.ID] = t
	return nil
}

func (r *memRepo) GetByID(id string) (domain.Task, error) {
	t, ok := r.tasks[id]
	if !ok {
		return domain.Task{}, errors.New("not found")
	}
	return t, nil
}

func (r *memRepo) GetPending() ([]domain.Task, error) {
	var out []domain.Task
	for _, t := range r.tasks {
		if t.Status == domain.StatusQueued || t.Status == domain.StatusProcessing {
			out = append(out, t)
		}
	}
	return out, nil
}

func (r *memRepo) UpdateStatus(id string, status domain.TaskStatus) error {
	t, ok := r.tasks[id]
	if !ok {
		return errors.New("not found")
	}
	t.Status = status
	r.tasks[id] = t
	return nil
}

type noopWriter struct{}

func (w *noopWriter) WriteCodeToFile(_, _, _ string) error           { return nil }
func (w *noopWriter) ReadContextFiles(_ string, _ []string) (string, error) { return "", nil }

// --- Tests --------------------------------------------------------------------

func TestOrchestrator_SubmitTask_AssignsIDAndQueues(t *testing.T) {
	repo := newMemRepo()
	discovery := services.NewDiscoveryService() // no providers — worker won't process
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{})
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
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{})
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
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{})
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
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{})
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
