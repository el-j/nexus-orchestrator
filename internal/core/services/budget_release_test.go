package services_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

// Raising (or removing) a budget while a task waits for it must not keep the task parked until midnight.
func TestBudget_RaisingTheBudgetReleasesAWaitingTask(t *testing.T) {
	for name, after := range map[string]string{
		"raised":  `{"budgets":{"solo":{"dailyTokens":1000000}}}`,
		"removed": `{}`,
	} {
		t.Run(name, func(t *testing.T) {
			ledger := &memUsage{}
			solo := &scriptLLM{name: "solo", alive: true, replies: []string{"done"}}
			e := newEngine(t, &noopWriter{}, nil, nil, solo)
			services.WithUsageRepo(ledger)(e.orch)
			dir := ladderProject(t, `{"budgets":{"solo":{"dailyTokens":10}}}`)
			_ = ledger.Record(t.Context(), domain.UsageRecord{Provider: "solo", Usage: domain.Usage{InputTokens: 50}})

			id := e.queue(domain.Task{Instruction: "x", Role: "architect", ProjectPath: dir})
			e.runOne(id, domain.StatusWaiting)

			if n, err := e.orch.ResumeWaitingTasks(t.Context(), time.Now()); err != nil || n != 0 {
				t.Fatalf("still over budget: nothing resumes, got %d %v", n, err)
			}
			if err := os.WriteFile(filepath.Join(dir, services.PolicyFile), []byte(after), 0o644); err != nil {
				t.Fatal(err)
			}
			if n, err := e.orch.ResumeWaitingTasks(t.Context(), time.Now()); err != nil || n != 1 {
				t.Fatalf("the budget was lifted: the task resumes, got %d %v", n, err)
			}
			e.runOne(id, domain.StatusCompleted)
			if solo.calls != 1 {
				t.Errorf("the provider ran the task: %d", solo.calls)
			}
		})
	}
}
