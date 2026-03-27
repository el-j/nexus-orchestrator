package services_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

// --- in-memory stubs (activity-test-scoped names to avoid conflicts) ---

type actMemActivityRepo struct {
	mu         sync.Mutex
	activities []domain.AIActivity
}

func (r *actMemActivityRepo) SaveActivity(_ context.Context, a domain.AIActivity) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.activities = append(r.activities, a)
	return nil
}

func (r *actMemActivityRepo) ListActivities(_ context.Context, f domain.ActivityFilter) ([]domain.AIActivity, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.AIActivity
	for _, a := range r.activities {
		if f.AgentName != "" && a.AgentName != f.AgentName {
			continue
		}
		if !f.Since.IsZero() && a.Timestamp.Before(f.Since) {
			continue
		}
		out = append(out, a)
		if f.Limit > 0 && len(out) >= f.Limit {
			break
		}
	}
	return out, nil
}

func (r *actMemActivityRepo) PurgeOlderThan(_ context.Context, cutoff time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var kept []domain.AIActivity
	var purged int64
	for _, a := range r.activities {
		if a.Timestamp.Before(cutoff) {
			purged++
		} else {
			kept = append(kept, a)
		}
	}
	r.activities = kept
	return purged, nil
}

type mockReader struct {
	name       string
	activities []domain.AIActivity
}

func (m *mockReader) ReadActivities(_ context.Context, _ time.Time) ([]domain.AIActivity, error) {
	return m.activities, nil
}

func (m *mockReader) SourceName() string { return m.name }

// Ensure stubs satisfy ports interfaces at compile time.
var _ ports.AIActivityRepository = (*actMemActivityRepo)(nil)
var _ ports.ActivityReader = (*mockReader)(nil)

// --- tests ---

func TestActivityService_GetRecentActivities(t *testing.T) {
	repo := &actMemActivityRepo{}
	sess := newMemAISessionRepo()
	svc := services.NewActivityService(repo, sess)

	ctx := context.Background()
	now := time.Now()

	// Seed the repo directly to test the query path independently of poll().
	activities := []domain.AIActivity{
		{ID: "a1", AgentName: "claude", Summary: "first", Timestamp: now.Add(-30 * time.Minute)},
		{ID: "a2", AgentName: "copilot", Summary: "second", Timestamp: now.Add(-10 * time.Minute)},
		{ID: "a3", AgentName: "claude", Summary: "third", Timestamp: now.Add(-1 * time.Minute)},
	}
	for _, a := range activities {
		if err := repo.SaveActivity(ctx, a); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	t.Run("no filter returns all", func(t *testing.T) {
		got, err := svc.GetRecentActivities(ctx, domain.ActivityFilter{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 3 {
			t.Errorf("expected 3, got %d", len(got))
		}
	})

	t.Run("filter by agent name", func(t *testing.T) {
		got, err := svc.GetRecentActivities(ctx, domain.ActivityFilter{AgentName: "claude"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Errorf("expected 2, got %d", len(got))
		}
	})

	t.Run("limit respected", func(t *testing.T) {
		got, err := svc.GetRecentActivities(ctx, domain.ActivityFilter{Limit: 1})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Errorf("expected 1, got %d", len(got))
		}
	})
}

func TestActivityService_GetTimeline(t *testing.T) {
	repo := &actMemActivityRepo{}
	svc := services.NewActivityService(repo, newMemAISessionRepo())

	ctx := context.Background()
	now := time.Now()

	_ = repo.SaveActivity(ctx, domain.AIActivity{ID: "t1", Timestamp: now.Add(-2 * time.Hour)})
	_ = repo.SaveActivity(ctx, domain.AIActivity{ID: "t2", Timestamp: now.Add(-30 * time.Minute)})

	got, err := svc.GetTimeline(ctx, now.Add(-1*time.Hour), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected 1 result since 1h ago, got %d", len(got))
	}
}

func TestActivityService_PollSavesAndBridgesSession(t *testing.T) {
	repo := &actMemActivityRepo{}
	sessRepo := newMemAISessionRepo()

	activity := domain.AIActivity{
		ID:          "poll-1",
		AgentName:   "claude",
		ProjectPath: "/tmp/myproject",
		Summary:     "editing files",
		Timestamp:   time.Now(),
		TokensIn:    10,
		TokensOut:   20,
	}
	reader := &mockReader{name: "test-reader", activities: []domain.AIActivity{activity}}

	var broadcastCalls int
	broadcaster := &stubBroadcaster{onBroadcast: func(a domain.AIActivity) { broadcastCalls++ }}

	svc := services.NewActivityService(repo, sessRepo, reader)
	svc.SetBroadcaster(broadcaster)

	// Start, give one poll cycle, then stop.
	svc.Start()
	time.Sleep(50 * time.Millisecond)
	svc.Stop()

	ctx := context.Background()

	// Activity should be persisted.
	got, err := svc.GetRecentActivities(ctx, domain.ActivityFilter{AgentName: "claude"})
	if err != nil {
		t.Fatalf("list activities: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected at least 1 activity saved by poll")
	}

	// Session should be auto-created.
	sessions, err := sessRepo.ListAISessions(ctx)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) == 0 {
		t.Fatal("expected session to be bridged from activity")
	}
	sess := sessions[0]
	if sess.AgentName != "claude" {
		t.Errorf("expected agentName claude, got %s", sess.AgentName)
	}
	if sess.Status != domain.SessionStatusActive {
		t.Errorf("expected status active, got %s", sess.Status)
	}

	// Broadcaster should have been called.
	if broadcastCalls == 0 {
		t.Error("expected broadcaster to be called")
	}
}

type stubBroadcaster struct {
	onBroadcast func(domain.AIActivity)
}

func (b *stubBroadcaster) BroadcastActivityEvent(a domain.AIActivity) {
	if b.onBroadcast != nil {
		b.onBroadcast(a)
	}
}

var _ ports.ActivityBroadcaster = (*stubBroadcaster)(nil)
