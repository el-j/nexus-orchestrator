package services_test

import (
	"context"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

// A task the engine found no provider for can be taken by an agent session.
func TestClaimTask_ATaskWithNoProviderCanBeClaimed(t *testing.T) {
	orch, repo, aiRepo := setupClaimTestOrch(t)
	seedActiveSession(t, aiRepo, "sess-1")
	id, _ := orch.SubmitTask(domain.Task{Instruction: "no model here", ProjectPath: "/tmp"})
	if err := repo.UpdateStatus(context.Background(), id, domain.StatusNoProvider); err != nil {
		t.Fatal(err)
	}
	task, err := orch.ClaimTask(context.Background(), id, "sess-1")
	if err != nil || task.Status != domain.StatusProcessing || task.AISessionID != "sess-1" {
		t.Fatalf("claim: %+v %v", task, err)
	}
	// Another status is still refused.
	if err := repo.UpdateStatus(context.Background(), id, domain.StatusCompleted); err != nil {
		t.Fatal(err)
	}
	if _, err := orch.ClaimTask(context.Background(), id, "sess-1"); err == nil {
		t.Error("a COMPLETED task cannot be claimed")
	}
}
