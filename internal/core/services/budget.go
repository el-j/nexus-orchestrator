package services

import (
	"fmt"
	"strings"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// startOfLocalDay returns midnight at the start of now's local day.
func startOfLocalDay(now time.Time) time.Time {
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, now.Location())
}

// enforceBudgets puts every candidate that used up its daily token budget (policy `budgets`, counted from
// the usage ledger: input plus output tokens since local midnight) on cooldown until the next local
// midnight, so the usual machinery applies: it is skipped, and the task waits if every candidate is out.
// Without a ledger, a budget or when the ledger cannot be read nothing is enforced.
func (o *OrchestratorService) enforceBudgets(chain []ports.LLMClient, policy domain.ProjectPolicy) {
	if o.usageRepo == nil || len(policy.Budgets) == 0 {
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
	for _, c := range chain {
		b, ok := policy.BudgetFor(c.ProviderName())
		if !ok || used[strings.ToLower(strings.TrimSpace(c.ProviderName()))] < b.DailyTokens {
			continue
		}
		o.discovery.MarkCooldown(c, reset, domain.ErrKindQuotaExhausted,
			fmt.Sprintf("%s: daily budget of %d tokens used", c.ProviderName(), b.DailyTokens))
	}
}
