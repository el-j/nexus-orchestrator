package services

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// budgetReason is the text every budget cooldown carries, so a cooldown that the budget put there can be
// told from one a provider's own limit put there.
const budgetReason = "daily budget of"

// startOfLocalDay returns midnight at the start of now's local day.
func startOfLocalDay(now time.Time) time.Time {
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, now.Location())
}

// enforceBudgets puts every candidate that used up its daily token budget (policy `budgets`, counted from
// the usage ledger: input plus output tokens since local midnight) on cooldown until the next local
// midnight, so the usual machinery applies: it is skipped, and the task waits if every candidate is out.
// A cooldown the budget set earlier is lifted again when the budget was raised or removed in the
// meantime. Without a ledger, a budget or when the ledger cannot be read nothing is enforced.
func (o *OrchestratorService) enforceBudgets(chain []ports.LLMClient, policy domain.ProjectPolicy) {
	if o.usageRepo == nil || len(policy.Budgets) == 0 {
		o.liftBudgetCooldowns(chain, nil)
		return
	}
	now := time.Now()
	sums, err := o.usageRepo.Summarize(o.dbCtx(), startOfLocalDay(now))
	if err != nil {
		return
	}
	used := map[string]int{}
	for _, s := range sums {
		used[strings.ToLower(strings.TrimSpace(s.Provider))] += s.InputTokens + s.OutputTokens
	}
	reset := startOfLocalDay(now).AddDate(0, 0, 1)
	var over []ports.LLMClient
	for _, c := range chain {
		b, ok := policy.BudgetFor(c.ProviderName())
		if !ok || used[strings.ToLower(strings.TrimSpace(c.ProviderName()))] < b.DailyTokens {
			continue
		}
		over = append(over, c)
		o.discovery.MarkCooldown(c, reset, domain.ErrKindQuotaExhausted,
			fmt.Sprintf("%s: %s %d tokens used", c.ProviderName(), budgetReason, b.DailyTokens))
	}
	o.liftBudgetCooldowns(chain, over)
}

// liftBudgetCooldowns clears the cooldowns of the chain's candidates that a budget put there and that are
// not in keep (the ones still over their budget).
func (o *OrchestratorService) liftBudgetCooldowns(chain, keep []ports.LLMClient) {
	now := time.Now()
	for _, c := range chain {
		if slices.Contains(keep, c) {
			continue
		}
		if cd, cooling := o.discovery.Cooldown(c, now); cooling && strings.Contains(cd.reason, budgetReason) {
			o.discovery.ClearCooldown(c)
		}
	}
}

// firstAvailable returns the first candidate of the chain that is not cooling down, or nil.
func (o *OrchestratorService) firstAvailable(chain []ports.LLMClient) ports.LLMClient {
	now := time.Now()
	for _, c := range chain {
		if _, cooling := o.discovery.Cooldown(c, now); !cooling {
			return c
		}
	}
	return nil
}

// holdReleased reports whether a WAITING task that has not reached its wait time can run again because
// the cooldowns it waited for are gone, for example a budget that was raised. A task that names its
// provider, or whose providers cannot be resolved, stays put.
func (o *OrchestratorService) holdReleased(ctx context.Context, t domain.Task) bool {
	if t.ProviderName != "" {
		return false
	}
	chain, err := o.resolveProviderWithFallback(ctx, t)
	if err != nil || len(chain) == 0 {
		return false
	}
	chain, _ = o.routeChain(chain, t)
	_, cooling := o.discovery.cooldowns.earliest(chainKeys(chain), time.Now())
	return !cooling
}
