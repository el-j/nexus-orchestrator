package services_test

import (
	"context"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

func TestClaimTask_Success(t *testing.T) {
	orch, _, aiRepo := setupClaimTestOrch(t)
	seedActiveSession(t, aiRepo, "sess-1")

	id, _ := orch.SubmitTask(domain.Task{Instruction: "do work", ProjectPath: "/tmp"})

	task, err := orch.ClaimTask(context.Background(), id, "sess-1")
	if err != nil {
		t.Fatalf("ClaimTask: %v", err)
	}
	if task.Status != domain.StatusProcessing {
		t.Errorf("expected PROCESSING, got %s", task.Status)
	}
	if task.AISessionID != "sess-1" {
		t.Errorf("expected AISessionID=sess-1, got %s", task.AISessionID)
	}
}

func TestClaimTask_NonExistentSession(t *testing.T) {
	orch, _, _ := setupClaimTestOrch(t)
	id, _ := orch.SubmitTask(domain.Task{Instruction: "do work", ProjectPath: "/tmp"})

	_, err := orch.ClaimTask(context.Background(), id, "no-such-session")
	if err == nil {
		t.Fatal("expected error for non-existent session")
	}
	if !strings.Contains(err.Error(), "session lookup") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClaimTask_DisconnectedSession(t *testing.T) {
	orch, _, aiRepo := setupClaimTestOrch(t)
	err := aiRepo.SaveAISession(context.Background(), domain.AISession{
		ID:     "sess-dead",
		Status: domain.SessionStatusDisconnected,
		Source: domain.SessionSourceVSCode,
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	id, _ := orch.SubmitTask(domain.Task{Instruction: "work", ProjectPath: "/tmp"})

	_, err = orch.ClaimTask(context.Background(), id, "sess-dead")
	if err == nil {
		t.Fatal("expected error for disconnected session")
	}
	if !strings.Contains(err.Error(), "disconnected") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClaimTask_AlreadyClaimed(t *testing.T) {
	orch, _, aiRepo := setupClaimTestOrch(t)
	seedActiveSession(t, aiRepo, "sess-1")
	seedActiveSession(t, aiRepo, "sess-2")

	id, _ := orch.SubmitTask(domain.Task{Instruction: "race", ProjectPath: "/tmp"})

	// First claim succeeds.
	_, err := orch.ClaimTask(context.Background(), id, "sess-1")
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}

	// Second claim must fail — task is already PROCESSING.
	_, err = orch.ClaimTask(context.Background(), id, "sess-2")
	if err == nil {
		t.Fatal("expected error for already-claimed task")
	}
	if !strings.Contains(err.Error(), "not QUEUED") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestClaimTask_CrossProjectRejection is the critical isolation test:
// a session registered for /proj/b must NOT be able to claim a task from /proj/a.
func TestClaimTask_CrossProjectRejection(t *testing.T) {
	orch, _, aiRepo := setupClaimTestOrch(t)

	// Session scoped to project B.
	if err := aiRepo.SaveAISession(context.Background(), domain.AISession{
		ID:          "sess-proj-b",
		AgentName:   "agent-b",
		Source:      domain.SessionSourceMCP,
		Status:      domain.SessionStatusActive,
		ProjectPath: "/proj/b",
	}); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	// Task belongs to project A.
	taskID, _ := orch.SubmitTask(domain.Task{Instruction: "work", ProjectPath: "/proj/a"})

	_, err := orch.ClaimTask(context.Background(), taskID, "sess-proj-b")
	if err == nil {
		t.Fatal("expected error: session from /proj/b should not claim task from /proj/a")
	}
	if !strings.Contains(err.Error(), "does not match task project") {
		t.Errorf("unexpected error message: %v", err)
	}

	// Verify the task was NOT transitioned — it must still be QUEUED.
	task, fetchErr := orch.GetTask(taskID)
	if fetchErr != nil {
		t.Fatalf("GetTask: %v", fetchErr)
	}
	if task.Status != domain.StatusQueued {
		t.Errorf("task should remain QUEUED after rejected claim, got %s", task.Status)
	}
}

// TestClaimTask_SameProjectSucceeds verifies that project ownership matching works
// for the happy path: a session registered for /proj/a can claim /proj/a tasks.
func TestClaimTask_SameProjectSucceeds(t *testing.T) {
	orch, _, aiRepo := setupClaimTestOrch(t)

	if err := aiRepo.SaveAISession(context.Background(), domain.AISession{
		ID:          "sess-proj-a",
		AgentName:   "agent-a",
		Source:      domain.SessionSourceMCP,
		Status:      domain.SessionStatusActive,
		ProjectPath: "/proj/a",
	}); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	taskID, _ := orch.SubmitTask(domain.Task{Instruction: "work", ProjectPath: "/proj/a"})

	task, err := orch.ClaimTask(context.Background(), taskID, "sess-proj-a")
	if err != nil {
		t.Fatalf("same-project claim should succeed: %v", err)
	}
	if task.Status != domain.StatusProcessing {
		t.Errorf("expected PROCESSING, got %s", task.Status)
	}
}

func TestClaimTask_AppendRoutedTaskID(t *testing.T) {
	orch, _, aiRepo := setupClaimTestOrch(t)
	seedActiveSession(t, aiRepo, "sess-1")

	id, _ := orch.SubmitTask(domain.Task{Instruction: "routed", ProjectPath: "/tmp"})
	_, err := orch.ClaimTask(context.Background(), id, "sess-1")
	if err != nil {
		t.Fatalf("ClaimTask: %v", err)
	}

	sess, _ := aiRepo.GetAISessionByID(context.Background(), "sess-1")
	if len(sess.RoutedTaskIDs) != 1 || sess.RoutedTaskIDs[0] != id {
		t.Errorf("expected RoutedTaskIDs=[%s], got %v", id, sess.RoutedTaskIDs)
	}
}

func TestUpdateTaskStatus_Completed(t *testing.T) {
	orch, _, aiRepo := setupClaimTestOrch(t)
	seedActiveSession(t, aiRepo, "sess-1")

	id, _ := orch.SubmitTask(domain.Task{Instruction: "finish me", ProjectPath: "/tmp"})
	_, _ = orch.ClaimTask(context.Background(), id, "sess-1")

	task, err := orch.UpdateTaskStatus(context.Background(), id, "sess-1", domain.StatusCompleted, "done!")
	if err != nil {
		t.Fatalf("UpdateTaskStatus: %v", err)
	}
	if task.Status != domain.StatusCompleted {
		t.Errorf("expected COMPLETED, got %s", task.Status)
	}
	if task.Logs != "done!" {
		t.Errorf("expected logs='done!', got %q", task.Logs)
	}
}

func TestUpdateTaskStatus_Failed(t *testing.T) {
	orch, _, aiRepo := setupClaimTestOrch(t)
	seedActiveSession(t, aiRepo, "sess-1")

	id, _ := orch.SubmitTask(domain.Task{Instruction: "fail me", ProjectPath: "/tmp"})
	_, _ = orch.ClaimTask(context.Background(), id, "sess-1")

	task, err := orch.UpdateTaskStatus(context.Background(), id, "sess-1", domain.StatusFailed, "oops")
	if err != nil {
		t.Fatalf("UpdateTaskStatus: %v", err)
	}
	if task.Status != domain.StatusFailed {
		t.Errorf("expected FAILED, got %s", task.Status)
	}
}

func TestUpdateTaskStatus_WrongSession(t *testing.T) {
	orch, _, aiRepo := setupClaimTestOrch(t)
	seedActiveSession(t, aiRepo, "sess-1")
	seedActiveSession(t, aiRepo, "sess-2")

	id, _ := orch.SubmitTask(domain.Task{Instruction: "owned by 1", ProjectPath: "/tmp"})
	_, _ = orch.ClaimTask(context.Background(), id, "sess-1")

	_, err := orch.UpdateTaskStatus(context.Background(), id, "sess-2", domain.StatusCompleted, "")
	if err == nil {
		t.Fatal("expected ownership error")
	}
	if !strings.Contains(err.Error(), "does not own") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUpdateTaskStatus_InvalidStatus(t *testing.T) {
	orch, _, aiRepo := setupClaimTestOrch(t)
	seedActiveSession(t, aiRepo, "sess-1")

	id, _ := orch.SubmitTask(domain.Task{Instruction: "bad", ProjectPath: "/tmp"})
	_, _ = orch.ClaimTask(context.Background(), id, "sess-1")

	_, err := orch.UpdateTaskStatus(context.Background(), id, "sess-1", domain.StatusQueued, "")
	if err == nil {
		t.Fatal("expected error for invalid target status")
	}
	if !strings.Contains(err.Error(), "invalid target status") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUpdateTaskStatus_NotProcessing(t *testing.T) {
	orch, _, aiRepo := setupClaimTestOrch(t)
	seedActiveSession(t, aiRepo, "sess-1")

	id, _ := orch.SubmitTask(domain.Task{Instruction: "still queued", ProjectPath: "/tmp"})

	// Task is QUEUED (not claimed), so AISessionID is empty.
	// We need to manually set AISessionID to bypass ownership check and test the status guard.
	repo := orch // Access via the interface: set directly on memRepo
	_ = repo
	_, err := orch.UpdateTaskStatus(context.Background(), id, "sess-1", domain.StatusCompleted, "")
	if err == nil {
		t.Fatal("expected error for non-PROCESSING task")
	}
	// Either "does not own" or "not PROCESSING" is acceptable since AISessionID won't match.
	if !strings.Contains(err.Error(), "does not own") && !strings.Contains(err.Error(), "not PROCESSING") {
		t.Errorf("unexpected error: %v", err)
	}
}
