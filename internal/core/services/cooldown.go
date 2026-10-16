package services

import (
	"sync"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// cooldown is one provider that should be left alone until a time.
type cooldown struct {
	until  time.Time
	kind   domain.ProviderErrorKind
	reason string
}

// cooldownBook remembers which providers hit a limit and until when. The zero value is ready to use
// and every method is safe for concurrent use. A key identifies a provider and its model.
type cooldownBook struct {
	mu sync.Mutex
	m  map[string]cooldown
}

// cooldownKey identifies one provider and model: limits are per model, so the same vendor's
// other model stays available.
func cooldownKey(c ports.LLMClient) string { return c.ProviderName() + "|" + c.ActiveModel() }

// mark records a cooldown. A cooldown that ends later replaces one that ends earlier, never the
// other way round, so a short rate limit does not shorten an exhausted quota.
func (b *cooldownBook) mark(key string, until time.Time, kind domain.ProviderErrorKind, reason string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.m == nil {
		b.m = make(map[string]cooldown)
	}
	if old, ok := b.m[key]; ok && old.until.After(until) {
		return
	}
	b.m[key] = cooldown{until: until, kind: kind, reason: reason}
}

// active returns the cooldown of key when it has not ended at now.
func (b *cooldownBook) active(key string, now time.Time) (cooldown, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	c, ok := b.m[key]
	if !ok || !c.until.After(now) {
		return cooldown{}, false
	}
	return c, true
}

// clear forgets the cooldown of key.
func (b *cooldownBook) clear(key string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.m, key)
}

// earliest returns when the first of keys becomes available again. ok is false when keys is empty
// or one of them is not cooling down: there is then nothing to wait for.
func (b *cooldownBook) earliest(keys []string, now time.Time) (time.Time, bool) {
	var first time.Time
	for _, k := range keys {
		c, ok := b.active(k, now)
		if !ok {
			return time.Time{}, false
		}
		if first.IsZero() || c.until.Before(first) {
			first = c.until
		}
	}
	return first, !first.IsZero()
}

// MarkCooldown records that c hit a limit and should not be used until the given time.
func (s *DiscoveryService) MarkCooldown(c ports.LLMClient, until time.Time, kind domain.ProviderErrorKind, reason string) {
	s.cooldowns.mark(cooldownKey(c), until, kind, reason)
}

// Cooldown returns the active cooldown of c at now, if any.
func (s *DiscoveryService) Cooldown(c ports.LLMClient, now time.Time) (cooldown, bool) {
	return s.cooldowns.active(cooldownKey(c), now)
}

// ClearCooldown forgets the cooldown of c, after a call to it succeeded.
func (s *DiscoveryService) ClearCooldown(c ports.LLMClient) { s.cooldowns.clear(cooldownKey(c)) }
