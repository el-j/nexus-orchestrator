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

// GetUsageForProject adds the usage ledger up for calls belonging to projectPath at or after since.
func (o *OrchestratorService) GetUsageForProject(ctx context.Context, since time.Time, projectPath string) ([]domain.UsageSummary, error) {
	if o.usageRepo == nil {
		return []domain.UsageSummary{}, nil
	}
	if projectPath == "" {
		return o.usageRepo.Summarize(ctx, since)
	}
	return o.usageRepo.SummarizeForProject(ctx, since, projectPath)
}

// RecordUsage directly records an LLM usage entry into the ledger.
func (o *OrchestratorService) RecordUsage(ctx context.Context, rec domain.UsageRecord) error {
	if o.usageRepo == nil {
		return nil
	}
	return o.usageRepo.Record(ctx, rec)
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
		rec.ProjectPath = t.ProjectPath
	}
	if err := o.usageRepo.Record(o.dbCtx(), rec); err != nil {
		log.Printf("orchestrator: record usage for task %s: %v", taskID, err)
	}
}
