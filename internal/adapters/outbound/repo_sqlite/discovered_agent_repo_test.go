package repo_sqlite_test

import (
	"context"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
)

func TestDiscoveredAgentRepo_ListDiscoveredAgents_Empty(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()

	dr := repo_sqlite.NewDiscoveredAgentRepo(r)
	ctx := context.Background()

	agents, err := dr.ListDiscoveredAgents(ctx)
	if err != nil {
		t.Fatalf("ListDiscoveredAgents on empty DB: %v", err)
	}
	if agents == nil {
		// empty slice is fine; nil is also acceptable as long as no error
		agents = []domain.DiscoveredAgent{}
	}
	if len(agents) != 0 {
		t.Errorf("expected 0 agents, got %d", len(agents))
	}
}

func TestDiscoveredAgentRepo_UpsertDiscoveredAgent_InsertAndUpdate(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()

	dr := repo_sqlite.NewDiscoveredAgentRepo(r)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	agent := domain.DiscoveredAgent{
		ID:              "agent-1",
		Kind:            domain.AgentKindClaudeCLI,
		Name:            "Original Name",
		DetectionMethod: "process",
		ProcessName:     "claude",
		IsRunning:       true,
		LastSeen:        now,
	}

	if err := dr.UpsertDiscoveredAgent(ctx, agent); err != nil {
		t.Fatalf("UpsertDiscoveredAgent (insert): %v", err)
	}

	// Update the same agent ID with a different name
	agent.Name = "Updated Name"
	agent.IsRunning = false
	agent.LastSeen = now.Add(time.Minute)

	if err := dr.UpsertDiscoveredAgent(ctx, agent); err != nil {
		t.Fatalf("UpsertDiscoveredAgent (update): %v", err)
	}

	agents, err := dr.ListDiscoveredAgents(ctx)
	if err != nil {
		t.Fatalf("ListDiscoveredAgents: %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent after upsert, got %d", len(agents))
	}
	got := agents[0]
	if got.ID != "agent-1" {
		t.Errorf("ID: got %q, want %q", got.ID, "agent-1")
	}
	if got.Name != "Updated Name" {
		t.Errorf("Name: got %q, want %q", got.Name, "Updated Name")
	}
	if got.IsRunning != false {
		t.Errorf("IsRunning: got %v, want false", got.IsRunning)
	}
}

func TestDiscoveredAgentRepo_ListDiscoveredAgents_Populated(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()

	dr := repo_sqlite.NewDiscoveredAgentRepo(r)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)

	agents := []domain.DiscoveredAgent{
		{
			ID:              "ag-1",
			Kind:            domain.AgentKindCursor,
			Name:            "Cursor",
			DetectionMethod: "process",
			IsRunning:       true,
			LastSeen:        now.Add(-2 * time.Minute),
		},
		{
			ID:              "ag-2",
			Kind:            domain.AgentKindCopilot,
			Name:            "Copilot",
			DetectionMethod: "config",
			IsRunning:       false,
			LastSeen:        now,
		},
	}

	for _, a := range agents {
		if err := dr.UpsertDiscoveredAgent(ctx, a); err != nil {
			t.Fatalf("UpsertDiscoveredAgent %q: %v", a.ID, err)
		}
	}

	list, err := dr.ListDiscoveredAgents(ctx)
	if err != nil {
		t.Fatalf("ListDiscoveredAgents: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(list))
	}
	// Ordered by last_seen DESC — ag-2 (now) before ag-1 (now-2min)
	if list[0].ID != "ag-2" {
		t.Errorf("list[0].ID: got %q, want ag-2", list[0].ID)
	}
	if list[1].ID != "ag-1" {
		t.Errorf("list[1].ID: got %q, want ag-1", list[1].ID)
	}
}
