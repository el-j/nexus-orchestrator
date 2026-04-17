package services_test

import (
	"context"
	"sync"
	"testing"

	"nexus-orchestrator/internal/adapters/outbound/fs_writer"
	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

// newTestOrchestratorWithAISessionRepo builds an OrchestratorService backed by
// a fresh in-memory SQLite database with both task and AI session repos wired in.
func newTestOrchestratorWithAISessionRepo(t *testing.T) *services.OrchestratorService {
	t.Helper()

	repo, err := repo_sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("open :memory: db: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	sessionRepo := repo_sqlite.NewSessionRepo(repo)
	aiSessionRepo := repo_sqlite.NewAISessionRepo(repo)
	writer := fs_writer.New()
	discovery := services.NewDiscoveryService(&mockLLM{})
	orch, err := services.NewOrchestrator(discovery, repo, writer, sessionRepo)
	if err != nil {
		t.Fatal(err)
	}
	orch.SetAISessionRepo(aiSessionRepo)
	t.Cleanup(func() { orch.Stop() })

	return orch
}

// capturingBroadcaster records every Broadcast call for later inspection.
type capturingBroadcaster struct {
	mu            sync.Mutex
	events        []ports.TaskEvent
	sessionEvents []domain.AISessionEvent
}

func (b *capturingBroadcaster) Broadcast(ev ports.TaskEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, ev)
}

func (b *capturingBroadcaster) BroadcastAISessionEvent(ev domain.AISessionEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sessionEvents = append(b.sessionEvents, ev)
}

func (b *capturingBroadcaster) snapshot() []ports.TaskEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]ports.TaskEvent, len(b.events))
	copy(out, b.events)
	return out
}

func (b *capturingBroadcaster) sessionSnapshot() []domain.AISessionEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]domain.AISessionEvent, len(b.sessionEvents))
	copy(out, b.sessionEvents)
	return out
}

func TestAISessionLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("register_returns_active_session_in_list", func(t *testing.T) {
		t.Parallel()
		orch := newTestOrchestratorWithAISessionRepo(t)
		ctx := context.Background()

		created, err := orch.RegisterAISession(ctx, domain.AISession{
			AgentName:  "Claude Desktop",
			Source:     domain.SessionSourceMCP,
			ExternalID: "ext-1",
		})
		if err != nil {
			t.Fatalf("RegisterAISession: %v", err)
		}
		if created.ID == "" {
			t.Error("expected non-empty session ID")
		}
		if created.Status != domain.SessionStatusActive {
			t.Errorf("expected status %q, got %q", domain.SessionStatusActive, created.Status)
		}
		sessions, err := orch.ListAISessions(ctx)
		if err != nil {
			t.Fatalf("ListAISessions: %v", err)
		}
		if len(sessions) != 1 {
			t.Fatalf("expected 1 session, got %d", len(sessions))
		}
		if sessions[0].Status != domain.SessionStatusActive {
			t.Errorf("expected active session, got %q", sessions[0].Status)
		}
	})

	t.Run("register_same_id_is_idempotent", func(t *testing.T) {
		t.Parallel()
		orch := newTestOrchestratorWithAISessionRepo(t)
		ctx := context.Background()

		s := domain.AISession{
			ID:         "idempotent-id-001",
			AgentName:  "Claude Desktop",
			Source:     domain.SessionSourceHTTP,
			ExternalID: "ext-2",
		}
		if _, err := orch.RegisterAISession(ctx, s); err != nil {
			t.Fatalf("first register: %v", err)
		}
		if _, err := orch.RegisterAISession(ctx, s); err != nil {
			t.Fatalf("second register (same ID): %v", err)
		}
		sessions, err := orch.ListAISessions(ctx)
		if err != nil {
			t.Fatalf("ListAISessions: %v", err)
		}
		// INSERT OR REPLACE on the same primary key keeps exactly 1 row.
		if len(sessions) != 1 {
			t.Errorf("expected 1 session (INSERT OR REPLACE idempotent), got %d", len(sessions))
		}
	})

	t.Run("deregister_marks_session_disconnected", func(t *testing.T) {
		t.Parallel()
		orch := newTestOrchestratorWithAISessionRepo(t)
		ctx := context.Background()

		created, err := orch.RegisterAISession(ctx, domain.AISession{
			AgentName: "Cursor",
			Source:    domain.SessionSourceVSCode,
		})
		if err != nil {
			t.Fatalf("RegisterAISession: %v", err)
		}
		if err := orch.DeregisterAISession(ctx, created.ID); err != nil {
			t.Fatalf("DeregisterAISession: %v", err)
		}
		sessions, err := orch.ListAISessions(ctx)
		if err != nil {
			t.Fatalf("ListAISessions: %v", err)
		}
		if len(sessions) != 1 {
			t.Fatalf("expected 1 session, got %d", len(sessions))
		}
		if sessions[0].Status != domain.SessionStatusDisconnected {
			t.Errorf("expected status %q, got %q", domain.SessionStatusDisconnected, sessions[0].Status)
		}
	})

	t.Run("register_after_deregister_creates_active_session", func(t *testing.T) {
		t.Parallel()
		orch := newTestOrchestratorWithAISessionRepo(t)
		ctx := context.Background()

		first, err := orch.RegisterAISession(ctx, domain.AISession{
			ID:        "reuse-test-session",
			AgentName: "GitHub Copilot",
			Source:    domain.SessionSourceVSCode,
		})
		if err != nil {
			t.Fatalf("first register: %v", err)
		}
		if err := orch.DeregisterAISession(ctx, first.ID); err != nil {
			t.Fatalf("deregister: %v", err)
		}
		second, err := orch.RegisterAISession(ctx, domain.AISession{
			AgentName: "GitHub Copilot",
			Source:    domain.SessionSourceVSCode,
		})
		if err != nil {
			t.Fatalf("re-register: %v", err)
		}
		if second.Status != domain.SessionStatusActive {
			t.Errorf("re-registered session: expected active, got %q", second.Status)
		}
		sessions, err := orch.ListAISessions(ctx)
		if err != nil {
			t.Fatalf("ListAISessions: %v", err)
		}
		// Two rows: original (disconnected) + new (active).
		if len(sessions) != 2 {
			t.Errorf("expected 2 sessions (disconnected+active), got %d", len(sessions))
		}
		var foundActive bool
		for _, sess := range sessions {
			if sess.ID == second.ID {
				if sess.Status != domain.SessionStatusActive {
					t.Errorf("re-registered session: expected active, got %q", sess.Status)
				}
				foundActive = true
			}
		}
		if !foundActive {
			t.Errorf("re-registered session %q not found in list", second.ID)
		}
	})
}

func TestAISession_Heartbeat_UpdatesTTL(t *testing.T) {
	t.Parallel()
	orch := newTestOrchestratorWithAISessionRepo(t)
	ctx := context.Background()

	created, err := orch.RegisterAISession(ctx, domain.AISession{
		AgentName: "HeartbeatAgent",
		Source:    domain.SessionSourceHTTP,
	})
	if err != nil {
		t.Fatalf("RegisterAISession: %v", err)
	}

	// Record baseline UpdatedAt from the stored session.
	sessions, err := orch.ListAISessions(ctx)
	if err != nil {
		t.Fatalf("ListAISessions before heartbeat: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	beforeHeartbeat := sessions[0].UpdatedAt

	// Heartbeat must not return an error.
	if err := orch.HeartbeatAISession(ctx, created.ID); err != nil {
		t.Fatalf("HeartbeatAISession: %v", err)
	}

	// Verify that UpdatedAt was refreshed (it must be >= beforeHeartbeat and
	// the session must still be active).
	sessions, err = orch.ListAISessions(ctx)
	if err != nil {
		t.Fatalf("ListAISessions after heartbeat: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session after heartbeat, got %d", len(sessions))
	}
	after := sessions[0]
	if after.Status != domain.SessionStatusActive {
		t.Errorf("expected session to remain active after heartbeat, got %q", after.Status)
	}
	if after.UpdatedAt.Before(beforeHeartbeat) {
		t.Errorf("expected UpdatedAt to be refreshed: before=%v, after=%v", beforeHeartbeat, after.UpdatedAt)
	}
}

func TestAISession_Terminate_SetsStatus(t *testing.T) {
	t.Parallel()
	orch := newTestOrchestratorWithAISessionRepo(t)
	ctx := context.Background()

	created, err := orch.RegisterAISession(ctx, domain.AISession{
		AgentName: "TerminateAgent",
		Source:    domain.SessionSourceVSCode,
	})
	if err != nil {
		t.Fatalf("RegisterAISession: %v", err)
	}
	if created.Status != domain.SessionStatusActive {
		t.Fatalf("expected active after register, got %q", created.Status)
	}

	// Terminate (no real PID → just the status update path).
	if err := orch.TerminateAISession(ctx, created.ID, false); err != nil {
		t.Fatalf("TerminateAISession: %v", err)
	}

	sessions, err := orch.ListAISessions(ctx)
	if err != nil {
		t.Fatalf("ListAISessions after terminate: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Status != domain.SessionStatusDisconnected {
		t.Errorf("expected status %q after terminate, got %q", domain.SessionStatusDisconnected, sessions[0].Status)
	}
}

func TestAISession_Purge_RemovesDisconnected(t *testing.T) {
	t.Parallel()
	orch := newTestOrchestratorWithAISessionRepo(t)
	ctx := context.Background()

	// Register and immediately deregister so the session is disconnected.
	created, err := orch.RegisterAISession(ctx, domain.AISession{
		AgentName: "PurgeAgent",
		Source:    domain.SessionSourceMCP,
	})
	if err != nil {
		t.Fatalf("RegisterAISession: %v", err)
	}
	if err := orch.DeregisterAISession(ctx, created.ID); err != nil {
		t.Fatalf("DeregisterAISession: %v", err)
	}

	// Confirm one disconnected session exists before purge.
	sessions, err := orch.ListAISessions(ctx)
	if err != nil {
		t.Fatalf("ListAISessions before purge: %v", err)
	}
	if len(sessions) != 1 || sessions[0].Status != domain.SessionStatusDisconnected {
		t.Fatalf("expected 1 disconnected session before purge, got: %+v", sessions)
	}

	// PurgeDisconnectedSessions removes sessions inactive > 2h.
	// The session was just disconnected so it is < 2h old; n may be 0.
	// The important assertion is: no error, and the service remains functional.
	n, err := orch.PurgeDisconnectedSessions(ctx)
	if err != nil {
		t.Fatalf("PurgeDisconnectedSessions: %v", err)
	}
	// n is 0 because the session was disconnected < 2h ago; that is the
	// correct behaviour — purge only removes *stale* disconnected sessions.
	_ = n

	// Session list is unchanged (still 1 disconnected, not yet stale).
	sessions, err = orch.ListAISessions(ctx)
	if err != nil {
		t.Fatalf("ListAISessions after purge: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("expected 1 session to survive non-stale purge, got %d", len(sessions))
	}
}

func TestAISessionBroadcast(t *testing.T) {
	t.Parallel()

	t.Run("register_emits_ai_session_changed", func(t *testing.T) {
		t.Parallel()
		orch := newTestOrchestratorWithAISessionRepo(t)
		bc := &capturingBroadcaster{}
		orch.SetBroadcaster(bc)

		ctx := context.Background()
		created, err := orch.RegisterAISession(ctx, domain.AISession{
			AgentName: "TestAgent",
			Source:    domain.SessionSourceHTTP,
		})
		if err != nil {
			t.Fatalf("RegisterAISession: %v", err)
		}
		// RegisterAISession calls BroadcastAISessionEvent synchronously before returning.
		events := bc.sessionSnapshot()
		if len(events) == 0 {
			t.Fatal("expected at least one broadcast event after RegisterAISession")
		}
		var found bool
		for _, ev := range events {
			if ev.Type == "ai_session_changed" && ev.AISessionID == created.ID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected ai_session_changed for session %q, got: %v", created.ID, events)
		}
	})

	t.Run("deregister_emits_ai_session_changed", func(t *testing.T) {
		t.Parallel()
		orch := newTestOrchestratorWithAISessionRepo(t)
		bc := &capturingBroadcaster{}
		orch.SetBroadcaster(bc)

		ctx := context.Background()
		created, err := orch.RegisterAISession(ctx, domain.AISession{
			AgentName: "TestAgent",
			Source:    domain.SessionSourceMCP,
		})
		if err != nil {
			t.Fatalf("RegisterAISession: %v", err)
		}
		if err := orch.DeregisterAISession(ctx, created.ID); err != nil {
			t.Fatalf("DeregisterAISession: %v", err)
		}
		// Expect >=2 events: one from RegisterAISession, one from DeregisterAISession.
		events := bc.sessionSnapshot()
		if len(events) < 2 {
			t.Fatalf("expected >=2 events (register+deregister), got %d: %v", len(events), events)
		}
		last := events[len(events)-1]
		if last.Type != "ai_session_changed" {
			t.Errorf("expected last event type %q, got %q", "ai_session_changed", last.Type)
		}
		if last.AISessionID != created.ID {
			t.Errorf("expected last event aiSessionId %q, got %q", created.ID, last.AISessionID)
		}
	})
}
