package services

import (
	"context"
	"log"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// GetUsage adds the usage ledger up per provider, model and role for the calls made at or after
// since (the zero time means everything). Without a ledger the answer is empty.
func (o *OrchestratorService) GetUsage(ctx context.Context, since time.Time) ([]domain.UsageSummary, error) {
	if o.usageRepo == nil {
		return []domain.UsageSummary{}, nil
	}
	return o.usageRepo.Summarize(ctx, since)
}

// recordUsage stores what the call that just succeeded consumed, when the provider reports it. A
// ledger problem is logged and never fails the task.
func (o *OrchestratorService) recordUsage(taskID string, llm ports.LLMClient) {
	if o.usageRepo == nil {
		return
	}
	reporter, ok := llm.(ports.UsageReporter)
	if !ok {
		return
	}
	u, ok := reporter.LastUsage()
	if !ok {
		return
	}
	if u.Model == "" {
		u.Model = llm.ActiveModel()
	}
	rec := domain.UsageRecord{At: time.Now(), TaskID: taskID, Provider: llm.ProviderName(), Usage: u}
	if t, err := o.repo.GetByID(o.dbCtx(), taskID); err == nil {
		rec.Role = t.Role
	}
	if err := o.usageRepo.Record(o.dbCtx(), rec); err != nil {
		log.Printf("orchestrator: record usage for task %s: %v", taskID, err)
	}
}
