package services_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

// scriptedActivityRepo lets tests inject save/purge failures and observe purges.
type scriptedActivityRepo struct {
	actMemActivityRepo
	mu2      sync.Mutex
	saveErr  error
	purgeErr error
	purges   []time.Time
}

func (r *scriptedActivityRepo) SaveActivity(ctx context.Context, a domain.AIActivity) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	return r.actMemActivityRepo.SaveActivity(ctx, a)
}

func (r *scriptedActivityRepo) PurgeOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	r.mu2.Lock()
	r.purges = append(r.purges, cutoff)
	err := r.purgeErr
	r.mu2.Unlock()
	if err != nil {
		return 0, err
	}
	return r.actMemActivityRepo.PurgeOlderThan(ctx, cutoff)
}

func (r *scriptedActivityRepo) purgeCalls() int {
	r.mu2.Lock()
	defer r.mu2.Unlock()
	return len(r.purges)
}

type erroringReader struct{}

func (erroringReader) ReadActivities(context.Context, time.Time) ([]domain.AIActivity, error) {
	return nil, errInjected
}
func (erroringReader) SourceName() string { return "erroring" }

func newActivitySvc(repo ports.AIActivityRepository, sessions ports.AISessionRepository, readers ...ports.ActivityReader) *services.ActivityService {
	return services.NewActivityService(repo, sessions, readers...)
}

func TestActivityService_PurgeLoopRunsOnItsInterval(t *testing.T) {
	repo := &scriptedActivityRepo{}
	svc := newActivitySvc(repo, newFaultyAIRepo())
	svc.SetIntervals(time.Hour, 10*time.Millisecond)
	now := time.Now()
	_ = repo.actMemActivityRepo.SaveActivity(bg, domain.AIActivity{ID: "old", Timestamp: now.Add(-48 * time.Hour)})
	_ = repo.actMemActivityRepo.SaveActivity(bg, domain.AIActivity{ID: "new", Timestamp: now})
	svc.Start()
	defer svc.Stop()

	eventually(t, "purge loop to run", func() bool { return repo.purgeCalls() > 0 })
	eventually(t, "the 48h-old activity to be purged", func() bool {
		got, _ := repo.ListActivities(bg, domain.ActivityFilter{})
		return len(got) == 1 && got[0].ID == "new"
	})
	// A failing purge must not stop the loop.
	repo.mu2.Lock()
	repo.purgeErr = errInjected
	repo.mu2.Unlock()
	before := repo.purgeCalls()
	eventually(t, "purge loop to keep running after an error", func() bool { return repo.purgeCalls() > before+1 })
}

func TestActivityService_PollSurvivesReaderAndSaveErrors(t *testing.T) {
	repo := &scriptedActivityRepo{saveErr: errInjected}
	sessions := newFaultyAIRepo()
	good := &mockReader{name: "good", activities: []domain.AIActivity{{ID: "a1", AgentName: "claude", Timestamp: time.Now()}}}
	svc := newActivitySvc(repo, sessions, erroringReader{}, good)
	svc.SetIntervals(10*time.Millisecond, time.Hour)
	svc.Start()
	time.Sleep(60 * time.Millisecond) // several polls where every save fails
	svc.Stop()

	if got, _ := sessions.ListAISessions(bg); len(got) != 0 {
		t.Errorf("a failed save must not create a session, got %+v", got)
	}
}

func TestActivityService_BridgeSessionBranches(t *testing.T) {
	now := time.Now()
	act := func(id, agent, project, summary string, in, out int) domain.AIActivity {
		return domain.AIActivity{ID: id, AgentName: agent, ProjectPath: project, Summary: summary, TokensIn: in, TokensOut: out, Timestamp: now}
	}
	repo := &actMemActivityRepo{}
	sessions := newFaultyAIRepo()
	reader := &mockReader{name: "r", activities: []domain.AIActivity{
		act("1", "", "/p", "ignored: no agent name", 1, 1),
		act("2", "claude", "/p", "first", 3, 4),
		act("3", "claude", "/p", "second", 1, 1),
		act("4", "claude", "/p", "", 0, 0), // empty summary keeps LastMessage
		act("5", "claude", "/other", "other project", 0, 0),
	}}
	svc := newActivitySvc(repo, sessions, reader)
	svc.SetIntervals(time.Hour, time.Hour)
	svc.Start()
	eventually(t, "first poll to bridge sessions", func() bool {
		got, _ := sessions.ListAISessions(bg)
		return len(got) == 2
	})
	svc.Stop()

	all, _ := sessions.ListAISessions(bg)
	byProject := map[string]domain.AISession{}
	for _, s := range all {
		byProject[s.ProjectPath] = s
	}
	s := byProject["/p"]
	if s.ID != "act-claude-2" || s.Source != "discovered" || s.Status != domain.SessionStatusActive {
		t.Errorf("created session = %+v", s)
	}
	if s.MessageCount != 3 || s.TokensUsed != 9 || s.LastMessage != "second" || s.CurrentActivity != "" {
		t.Errorf("updates not applied (count=%d tokens=%d last=%q current=%q)", s.MessageCount, s.TokensUsed, s.LastMessage, s.CurrentActivity)
	}
	if byProject["/other"].ID != "act-claude-5" {
		t.Errorf("a different project must get its own session: %+v", byProject["/other"])
	}
}

func TestActivityService_BridgeSessionToleratesRepositoryErrors(t *testing.T) {
	repo := &actMemActivityRepo{}
	sessions := newFaultyAIRepo()
	reader := &mockReader{name: "r", activities: []domain.AIActivity{{ID: "1", AgentName: "a", ProjectPath: "/p", Timestamp: time.Now()}}}

	sessions.set("List", errInjected)
	svc := newActivitySvc(repo, sessions, reader)
	svc.SetIntervals(time.Hour, time.Hour)
	svc.Start()
	time.Sleep(30 * time.Millisecond)
	svc.Stop()

	// Create failure.
	sessions2 := newFaultyAIRepo()
	sessions2.set("Save", errInjected)
	svc2 := newActivitySvc(repo, sessions2, reader)
	svc2.SetIntervals(time.Hour, time.Hour)
	svc2.Start()
	time.Sleep(30 * time.Millisecond)
	svc2.Stop()
	if got, _ := sessions2.ListAISessions(bg); len(got) != 0 {
		t.Errorf("save failed, nothing should be stored: %+v", got)
	}

	// Update failure: pre-create the session, then fail saves.
	sessions3 := newFaultyAIRepo()
	_ = sessions3.memAISessionRepo.SaveAISession(bg, domain.AISession{ID: "x", AgentName: "a", ProjectPath: "/p", Source: "discovered", Status: domain.SessionStatusIdle})
	sessions3.set("Save", errInjected)
	svc3 := newActivitySvc(repo, sessions3, reader)
	svc3.SetIntervals(time.Hour, time.Hour)
	svc3.Start()
	time.Sleep(30 * time.Millisecond)
	svc3.Stop()
	if got, _ := sessions3.GetAISessionByID(bg, "x"); got.Status != domain.SessionStatusIdle {
		t.Errorf("failed update must leave the session untouched, got %s", got.Status)
	}
}

func TestActivityService_IdleAndDisconnectSweep(t *testing.T) {
	sessions := newFaultyAIRepo()
	now := time.Now()
	seed := func(id, agent string, source domain.AISessionSource, last time.Time, st domain.AISessionStatus) {
		_ = sessions.memAISessionRepo.SaveAISession(bg, domain.AISession{
			ID: id, AgentName: agent, ProjectPath: "/p", Source: source, LastActivity: last, Status: st,
		})
	}
	seed("fresh", "fresh-agent", "discovered", now, domain.SessionStatusActive)
	seed("idle", "idle-agent", "discovered", now.Add(-6*time.Minute), domain.SessionStatusActive)
	seed("gone", "gone-agent", "discovered", now.Add(-20*time.Minute), domain.SessionStatusIdle)
	seed("already", "already-agent", "discovered", now.Add(-20*time.Minute), domain.SessionStatusDisconnected)
	seed("manual", "manual-agent", domain.SessionSourceVSCode, now.Add(-2*time.Hour), domain.SessionStatusActive)
	seed("seen", "seen-agent", "discovered", now.Add(-2*time.Hour), domain.SessionStatusActive)

	svc := newActivitySvc(&actMemActivityRepo{}, sessions)
	// lastSeen overrides a stale persisted LastActivity.
	svc.MarkSeen("seen-agent", "/p", now)
	svc.CheckIdleDisconnect(bg)

	want := map[string]domain.AISessionStatus{
		"fresh":   domain.SessionStatusActive,
		"idle":    domain.SessionStatusIdle,
		"gone":    domain.SessionStatusDisconnected,
		"already": domain.SessionStatusDisconnected,
		"manual":  domain.SessionStatusActive, // not auto-created: never managed
		"seen":    domain.SessionStatusActive,
	}
	for id, st := range want {
		if got, _ := sessions.GetAISessionByID(bg, id); got.Status != st {
			t.Errorf("session %s: status %s, want %s", id, got.Status, st)
		}
	}

	// Repository failures are tolerated.
	sessions.set("List", errInjected)
	svc.CheckIdleDisconnect(bg)
	sessions.set("List", nil)
	sessions.set("Save", errors.New("boom"))
	_ = sessions.memAISessionRepo.SaveAISession(bg, domain.AISession{ID: "stale2", AgentName: "s2", Source: "discovered", LastActivity: now.Add(-time.Hour), Status: domain.SessionStatusActive})
	svc.CheckIdleDisconnect(bg)
}
