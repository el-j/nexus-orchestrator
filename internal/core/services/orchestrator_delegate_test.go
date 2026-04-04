package services_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

func TestDelegateToNexus_Success(t *testing.T) {
	t.Parallel()
	orch := newTestOrchestratorWithAISessionRepo(t)
	ctx := context.Background()

	sess, err := orch.RegisterAISession(ctx, domain.AISession{
		AgentName:   "Claude Desktop",
		Source:      domain.SessionSourceMCP,
		ExternalID:  "ext-delegate-1",
		ProjectPath: "/tmp/delegate-project",
	})
	if err != nil {
		t.Fatalf("RegisterAISession: %v", err)
	}

	instruction, err := orch.DelegateToNexus(ctx, sess.ID)
	if err != nil {
		t.Fatalf("DelegateToNexus: %v", err)
	}
	if !strings.Contains(instruction, "nexusOrchestrator") {
		t.Errorf("instruction missing 'nexusOrchestrator': %q", instruction)
	}

	sessions, err := orch.ListAISessions(ctx)
	if err != nil {
		t.Fatalf("ListAISessions: %v", err)
	}
	var found *domain.AISession
	for i := range sessions {
		if sessions[i].ID == sess.ID {
			found = &sessions[i]
			break
		}
	}
	if found == nil {
		t.Fatal("session not found after delegation")
	}
	if !found.DelegatedToNexus {
		t.Error("expected DelegatedToNexus=true after DelegateToNexus")
	}
	if found.DelegationTimestamp == nil {
		t.Error("expected non-nil DelegationTimestamp after DelegateToNexus")
	}
}

func TestDelegateToNexus_NotFound(t *testing.T) {
	t.Parallel()
	orch := newTestOrchestratorWithAISessionRepo(t)
	ctx := context.Background()

	_, err := orch.DelegateToNexus(ctx, "nonexistent-session-id")
	if err == nil {
		t.Fatal("expected error for nonexistent session ID, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected domain.ErrNotFound, got: %v", err)
	}
}

func TestGetDiscoveredAgents_NilScanner(t *testing.T) {
	t.Parallel()
	// NewOrchestrator with no agentScanner wired — scanner stays nil.
	orch := newTestOrchestratorWithAISessionRepo(t)
	ctx := context.Background()

	agents, err := orch.GetDiscoveredAgents(ctx)
	if err != nil {
		t.Fatalf("GetDiscoveredAgents: %v", err)
	}
	if len(agents) != 0 {
		t.Errorf("expected 0 agents with nil scanner, got %d", len(agents))
	}
}
