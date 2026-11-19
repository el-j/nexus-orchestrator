package services

import (
	"context"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// PreviewRouting shows how routing would order the candidate providers for a task of the given role in
// the project right now: the ladder, the order, which providers are cooling down and the quota readings.
// It changes nothing.
func (o *OrchestratorService) PreviewRouting(ctx context.Context, projectPath, role string) (domain.RoutingPreview, error) {
	task := domain.Task{ProjectPath: projectPath, Role: role}
	chain, err := o.resolveProviderWithFallback(ctx, task)
	if err != nil {
		return domain.RoutingPreview{}, err
	}
	chain, ladder := o.routeChain(chain, task)
	threshold := domain.DefaultQuotaThreshold
	if policy, ok, perr := LoadProjectPolicy(projectPath); perr == nil && ok {
		threshold = policy.QuotaLimit()
	}
	preview := domain.RoutingPreview{Role: role, Ladder: ladder, QuotaThreshold: threshold, Order: []domain.RoutingEntry{}}
	now := time.Now()
	for i, c := range chain {
		e := domain.RoutingEntry{Position: i + 1, Provider: c.ProviderName(), Model: c.ActiveModel(), Rung: rungOf(ladder, c)}
		if cd, cooling := o.discovery.Cooldown(c, now); cooling {
			e.CoolingUntil, e.CooldownReason = cd.until, cd.reason
		}
		if q, ok := o.discovery.quotas.current(cooldownKey(c), now); ok && q.Utilization >= 0 {
			u := q.Utilization
			e.Utilization = &u
		}
		preview.Order = append(preview.Order, e)
	}
	return preview, nil
}
