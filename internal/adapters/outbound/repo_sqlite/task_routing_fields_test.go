package repo_sqlite_test

import (
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// A task's role and agentic settings must survive the queue: the worker re-reads
// the task from the database, so a field that is not stored is silently lost
// (role-based routing never took effect before it was persisted).
func TestTaskRoleAndAgenticSettingsRoundTrip(t *testing.T) {
	r := newTestRepo(t)
	task := newTask("t-role", domain.StatusQueued, time.Now())
	task.Role = "architect"
	task.Agentic = true
	task.MaxAgentTurns = 7
	if err := r.Save(task); err != nil {
		t.Fatal(err)
	}
	got, err := r.GetByID("t-role")
	if err != nil {
		t.Fatal(err)
	}
	if got.Role != "architect" || !got.Agentic || got.MaxAgentTurns != 7 {
		t.Errorf("round trip lost fields: role=%q agentic=%v turns=%d", got.Role, got.Agentic, got.MaxAgentTurns)
	}
	claimed, err := r.ClaimNextQueued()
	if err != nil {
		t.Fatal(err)
	}
	if claimed.Role != "architect" || !claimed.Agentic {
		t.Errorf("claimed task lost fields: %+v", claimed)
	}
}
