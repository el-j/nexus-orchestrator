package services_test

import (
	"context"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

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
		saved, _ := repo.GetByID(context.Background(), id)
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

	sess, err := sessRepo.GetByProjectPath(context.Background(), path)
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

	sessA, _ := sessRepo.GetByProjectPath(context.Background(), "/proj/alpha")
	sessB, _ := sessRepo.GetByProjectPath(context.Background(), "/proj/beta")

	if len(sessA.Messages) != 2 {
		t.Errorf("alpha: expected 2 messages, got %d", len(sessA.Messages))
	}
	if len(sessB.Messages) != 2 {
		t.Errorf("beta: expected 2 messages, got %d", len(sessB.Messages))
	}
}
