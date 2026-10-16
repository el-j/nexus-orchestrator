package services

import (
	"slices"
	"sync"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// quotaBook remembers the plan's own reading of each provider's allowance (see ports.QuotaReporter).
// The zero value is ready to use and every method is safe for concurrent use.
type quotaBook struct {
	mu sync.Mutex
	m  map[string]domain.QuotaReading
}

func (b *quotaBook) set(key string, q domain.QuotaReading) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.m == nil {
		b.m = make(map[string]domain.QuotaReading)
	}
	b.m[key] = q
}

// current returns the reading of key unless its window has reset since: an old reading says nothing
// about the allowance now.
func (b *quotaBook) current(key string, now time.Time) (domain.QuotaReading, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	q, ok := b.m[key]
	if !ok || (!q.ResetsAt.IsZero() && !q.ResetsAt.After(now)) {
		return domain.QuotaReading{}, false
	}
	return q, true
}

// noteQuota stores the quota reading a provider's client reported with the call that just ended, also
// when the call failed (a refused call carries the reading that explains it).
func (o *OrchestratorService) noteQuota(llm ports.LLMClient) {
	if r, ok := llm.(ports.QuotaReporter); ok {
		if q, ok := r.LastQuota(); ok {
			o.discovery.quotas.set(cooldownKey(llm), q)
		}
	}
}

// routeByQuota moves providers whose plan allowance is used up to the threshold or beyond behind the
// others, keeping the order within each group. It never removes a provider: when every candidate is
// near its limit the order stands. A task that names its provider is left alone.
func (o *OrchestratorService) routeByQuota(chain []ports.LLMClient, task domain.Task) []ports.LLMClient {
	if task.ProviderName != "" || len(chain) < 2 {
		return chain
	}
	threshold := domain.DefaultQuotaThreshold
	if policy, ok, err := LoadProjectPolicy(task.ProjectPath); err == nil && ok {
		threshold = policy.QuotaLimit()
	}
	now := time.Now()
	hot := func(c ports.LLMClient) bool {
		q, ok := o.discovery.quotas.current(cooldownKey(c), now)
		return ok && q.Utilization >= threshold
	}
	out := slices.Clone(chain)
	slices.SortStableFunc(out, func(a, b ports.LLMClient) int {
		switch ha, hb := hot(a), hot(b); {
		case ha == hb:
			return 0
		case ha:
			return 1
		}
		return -1
	})
	return out
}
