package repo_sqlite_test

import (
	"context"
	"testing"
	"time"

	repo_sqlite "nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
)

// newActivityRepo creates a fresh in-memory repository and an ActivityRepo for testing.
func newActivityRepo(t *testing.T) (*repo_sqlite.Repository, *repo_sqlite.ActivityRepo) {
	t.Helper()
	r := newTestRepo(t)
	return r, repo_sqlite.NewActivityRepo(r)
}

// makeActivity returns an AIActivity with deterministic field values.
func makeActivity(id, agent, project string, actType domain.ActivityType, ts time.Time) domain.AIActivity {
	return domain.AIActivity{
		ID:           id,
		SessionID:    "sess-" + id,
		AgentName:    agent,
		ActivityType: actType,
		Summary:      "summary for " + id,
		ProjectPath:  project,
		Model:        "gpt-test",
		TokensIn:     10,
		TokensOut:    20,
		Timestamp:    ts,
		Metadata:     map[string]string{"key": "val"},
	}
}

func TestSaveActivity_RoundTrip(t *testing.T) {
	r, ar := newActivityRepo(t)
	defer r.Close()
	ctx := context.Background()

	ts := time.Now().Truncate(time.Millisecond)
	act := makeActivity("act-rt-1", "agent-a", "/proj/alpha", domain.ActivityTypeMessage, ts)
	if err := ar.SaveActivity(ctx, act); err != nil {
		t.Fatalf("SaveActivity: %v", err)
	}

	list, err := ar.ListActivities(ctx, domain.ActivityFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListActivities: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(list))
	}
	got := list[0]
	if got.ID != act.ID {
		t.Errorf("ID: got %q, want %q", got.ID, act.ID)
	}
	if got.AgentName != act.AgentName {
		t.Errorf("AgentName: got %q, want %q", got.AgentName, act.AgentName)
	}
	if got.ActivityType != act.ActivityType {
		t.Errorf("ActivityType: got %q, want %q", got.ActivityType, act.ActivityType)
	}
	if got.Summary != act.Summary {
		t.Errorf("Summary: got %q, want %q", got.Summary, act.Summary)
	}
	if got.ProjectPath != act.ProjectPath {
		t.Errorf("ProjectPath: got %q, want %q", got.ProjectPath, act.ProjectPath)
	}
	if got.TokensIn != act.TokensIn {
		t.Errorf("TokensIn: got %d, want %d", got.TokensIn, act.TokensIn)
	}
	if got.TokensOut != act.TokensOut {
		t.Errorf("TokensOut: got %d, want %d", got.TokensOut, act.TokensOut)
	}
	if got.Metadata["key"] != "val" {
		t.Errorf("Metadata key: got %q, want %q", got.Metadata["key"], "val")
	}
}

func TestListActivities_FilterByAgentName(t *testing.T) {
	r, ar := newActivityRepo(t)
	defer r.Close()
	ctx := context.Background()

	base := time.Now().Add(-10 * time.Second)
	acts := []domain.AIActivity{
		makeActivity("flt-agent-1", "agent-alpha", "/proj", domain.ActivityTypeMessage, base),
		makeActivity("flt-agent-2", "agent-beta", "/proj", domain.ActivityTypeMessage, base.Add(time.Second)),
		makeActivity("flt-agent-3", "agent-alpha", "/proj", domain.ActivityTypeToolUse, base.Add(2*time.Second)),
	}
	for _, a := range acts {
		if err := ar.SaveActivity(ctx, a); err != nil {
			t.Fatalf("SaveActivity %q: %v", a.ID, err)
		}
	}

	list, err := ar.ListActivities(ctx, domain.ActivityFilter{AgentName: "agent-alpha"})
	if err != nil {
		t.Fatalf("ListActivities: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 activities for agent-alpha, got %d", len(list))
	}
	for _, a := range list {
		if a.AgentName != "agent-alpha" {
			t.Errorf("unexpected AgentName %q in result", a.AgentName)
		}
	}
}

func TestListActivities_FilterByProjectPath(t *testing.T) {
	r, ar := newActivityRepo(t)
	defer r.Close()
	ctx := context.Background()

	base := time.Now().Add(-10 * time.Second)
	acts := []domain.AIActivity{
		makeActivity("flt-proj-1", "agent-x", "/proj/one", domain.ActivityTypeMessage, base),
		makeActivity("flt-proj-2", "agent-x", "/proj/two", domain.ActivityTypeMessage, base.Add(time.Second)),
		makeActivity("flt-proj-3", "agent-y", "/proj/one", domain.ActivityTypeToolUse, base.Add(2*time.Second)),
	}
	for _, a := range acts {
		if err := ar.SaveActivity(ctx, a); err != nil {
			t.Fatalf("SaveActivity %q: %v", a.ID, err)
		}
	}

	list, err := ar.ListActivities(ctx, domain.ActivityFilter{ProjectPath: "/proj/one"})
	if err != nil {
		t.Fatalf("ListActivities: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 activities for /proj/one, got %d", len(list))
	}
	for _, a := range list {
		if a.ProjectPath != "/proj/one" {
			t.Errorf("unexpected ProjectPath %q in result", a.ProjectPath)
		}
	}
}

func TestListActivities_FilterBySince(t *testing.T) {
	r, ar := newActivityRepo(t)
	defer r.Close()
	ctx := context.Background()

	// Use deterministic timestamps separated by explicit offsets — no sleep.
	anchor := time.Now().Add(-5 * time.Second)
	cutoff := anchor.Add(2 * time.Second)

	old := makeActivity("flt-since-old", "agent-z", "/proj", domain.ActivityTypeMessage, anchor)
	newAct := makeActivity("flt-since-new", "agent-z", "/proj", domain.ActivityTypeMessage, anchor.Add(3*time.Second))

	for _, a := range []domain.AIActivity{old, newAct} {
		if err := ar.SaveActivity(ctx, a); err != nil {
			t.Fatalf("SaveActivity %q: %v", a.ID, err)
		}
	}

	list, err := ar.ListActivities(ctx, domain.ActivityFilter{Since: cutoff})
	if err != nil {
		t.Fatalf("ListActivities with Since: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 activity since cutoff, got %d", len(list))
	}
	if list[0].ID != newAct.ID {
		t.Errorf("expected %q, got %q", newAct.ID, list[0].ID)
	}
}

func TestListActivities_FilterByType(t *testing.T) {
	r, ar := newActivityRepo(t)
	defer r.Close()
	ctx := context.Background()

	base := time.Now().Add(-10 * time.Second)
	acts := []domain.AIActivity{
		makeActivity("flt-type-1", "agent-q", "/proj", domain.ActivityTypeMessage, base),
		makeActivity("flt-type-2", "agent-q", "/proj", domain.ActivityTypeToolUse, base.Add(time.Second)),
		makeActivity("flt-type-3", "agent-q", "/proj", domain.ActivityTypeFileEdit, base.Add(2*time.Second)),
	}
	for _, a := range acts {
		if err := ar.SaveActivity(ctx, a); err != nil {
			t.Fatalf("SaveActivity %q: %v", a.ID, err)
		}
	}

	list, err := ar.ListActivities(ctx, domain.ActivityFilter{Type: domain.ActivityTypeToolUse})
	if err != nil {
		t.Fatalf("ListActivities: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 tool_use activity, got %d", len(list))
	}
	if list[0].ActivityType != domain.ActivityTypeToolUse {
		t.Errorf("ActivityType: got %q, want %q", list[0].ActivityType, domain.ActivityTypeToolUse)
	}
}

func TestListActivities_Limit(t *testing.T) {
	r, ar := newActivityRepo(t)
	defer r.Close()
	ctx := context.Background()

	base := time.Now().Add(-10 * time.Second)
	for i := 0; i < 5; i++ {
		a := makeActivity(
			"lim-act-"+string(rune('0'+i)),
			"agent-lim",
			"/proj",
			domain.ActivityTypeMessage,
			base.Add(time.Duration(i)*time.Second),
		)
		if err := ar.SaveActivity(ctx, a); err != nil {
			t.Fatalf("SaveActivity: %v", err)
		}
	}

	list, err := ar.ListActivities(ctx, domain.ActivityFilter{Limit: 3})
	if err != nil {
		t.Fatalf("ListActivities with Limit: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 activities (limit), got %d", len(list))
	}
}

func TestListActivities_CombinedFilters(t *testing.T) {
	r, ar := newActivityRepo(t)
	defer r.Close()
	ctx := context.Background()

	anchor := time.Now().Add(-20 * time.Second)
	cutoff := anchor.Add(5 * time.Second)

	// Only this one should match: correct agent, project, type, and after cutoff.
	target := domain.AIActivity{
		ID:           "combo-match",
		AgentName:    "combo-agent",
		ActivityType: domain.ActivityTypeFileEdit,
		Summary:      "combo match",
		ProjectPath:  "/proj/combo",
		Timestamp:    anchor.Add(10 * time.Second),
	}
	others := []domain.AIActivity{
		// wrong agent
		{ID: "combo-wrong-agent", AgentName: "other-agent", ActivityType: domain.ActivityTypeFileEdit, ProjectPath: "/proj/combo", Timestamp: anchor.Add(10 * time.Second)},
		// wrong project
		{ID: "combo-wrong-proj", AgentName: "combo-agent", ActivityType: domain.ActivityTypeFileEdit, ProjectPath: "/proj/other", Timestamp: anchor.Add(10 * time.Second)},
		// wrong type
		{ID: "combo-wrong-type", AgentName: "combo-agent", ActivityType: domain.ActivityTypeMessage, ProjectPath: "/proj/combo", Timestamp: anchor.Add(10 * time.Second)},
		// before cutoff
		{ID: "combo-too-old", AgentName: "combo-agent", ActivityType: domain.ActivityTypeFileEdit, ProjectPath: "/proj/combo", Timestamp: anchor.Add(time.Second)},
	}

	if err := ar.SaveActivity(ctx, target); err != nil {
		t.Fatalf("SaveActivity target: %v", err)
	}
	for _, a := range others {
		if err := ar.SaveActivity(ctx, a); err != nil {
			t.Fatalf("SaveActivity %q: %v", a.ID, err)
		}
	}

	list, err := ar.ListActivities(ctx, domain.ActivityFilter{
		AgentName:   "combo-agent",
		ProjectPath: "/proj/combo",
		Type:        domain.ActivityTypeFileEdit,
		Since:       cutoff,
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("ListActivities combined: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 combined result, got %d", len(list))
	}
	if list[0].ID != "combo-match" {
		t.Errorf("expected combo-match, got %q", list[0].ID)
	}
}

func TestPurgeOlderThan(t *testing.T) {
	r, ar := newActivityRepo(t)
	defer r.Close()
	ctx := context.Background()

	anchor := time.Now().Add(-20 * time.Second)
	cutoff := anchor.Add(10 * time.Second)

	old1 := makeActivity("purge-old-1", "agent-p", "/proj", domain.ActivityTypeMessage, anchor.Add(time.Second))
	old2 := makeActivity("purge-old-2", "agent-p", "/proj", domain.ActivityTypeMessage, anchor.Add(3*time.Second))
	keep := makeActivity("purge-keep-1", "agent-p", "/proj", domain.ActivityTypeMessage, anchor.Add(15*time.Second))

	for _, a := range []domain.AIActivity{old1, old2, keep} {
		if err := ar.SaveActivity(ctx, a); err != nil {
			t.Fatalf("SaveActivity %q: %v", a.ID, err)
		}
	}

	n, err := ar.PurgeOlderThan(ctx, cutoff)
	if err != nil {
		t.Fatalf("PurgeOlderThan: %v", err)
	}
	if n != 2 {
		t.Errorf("PurgeOlderThan: deleted %d rows, want 2", n)
	}

	remaining, err := ar.ListActivities(ctx, domain.ActivityFilter{Limit: 100})
	if err != nil {
		t.Fatalf("ListActivities after purge: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining activity, got %d", len(remaining))
	}
	if remaining[0].ID != "purge-keep-1" {
		t.Errorf("remaining activity ID: got %q, want purge-keep-1", remaining[0].ID)
	}
}
