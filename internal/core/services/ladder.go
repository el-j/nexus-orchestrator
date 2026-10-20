package services

import (
	"slices"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// rungOf returns the index of the first ladder rung that fits c, or -1.
func rungOf(ladder []string, c ports.LLMClient) int {
	for i, r := range ladder {
		if domain.RungMatches(r, c.ProviderName(), c.ActiveModel()) {
			return i
		}
	}
	return -1
}

// routeChain orders the candidate providers of a task: first by the project's ladder for the task's
// role (cheapest rung first; providers outside the ladder after its rungs), then providers whose plan
// allowance is nearly used up behind the others (see routeByQuota). It never removes a candidate. A
// task that names its provider is left alone. It returns the chain and the ladder that applies.
func (o *OrchestratorService) routeChain(chain []ports.LLMClient, task domain.Task) ([]ports.LLMClient, []string) {
	policy, ok, err := LoadProjectPolicy(task.ProjectPath)
	if err != nil || !ok {
		policy = domain.ProjectPolicy{}
	}
	o.enforceBudgets(chain, policy)
	if task.ProviderName != "" {
		return chain, nil
	}
	out := slices.Clone(chain)
	ladder := policy.LadderFor(task.Role)
	if len(ladder) > 0 {
		pos := func(c ports.LLMClient) int {
			if i := rungOf(ladder, c); i >= 0 {
				return i
			}
			return len(ladder)
		}
		slices.SortStableFunc(out, func(a, b ports.LLMClient) int { return pos(a) - pos(b) })
	}
	return o.partitionByQuota(out, policy.QuotaLimit()), ladder
}

// stepUp returns the provider that should correct a failed verification instead of cur: the nearest
// rung above cur's that is not cooling down. It returns cur when there is no ladder, cur is not on it
// or already on its top rung.
func (o *OrchestratorService) stepUp(run *runHandle, cur ports.LLMClient) ports.LLMClient {
	if len(run.ladder) == 0 {
		return cur
	}
	here := rungOf(run.ladder, cur)
	if here < 0 {
		return cur
	}
	var best ports.LLMClient
	bestRung := len(run.ladder)
	now := time.Now()
	for _, c := range run.chain {
		r := rungOf(run.ladder, c)
		if r <= here || r >= bestRung {
			continue
		}
		if _, cooling := o.discovery.Cooldown(c, now); cooling {
			continue
		}
		best, bestRung = c, r
	}
	if best == nil {
		return cur
	}
	return best
}
