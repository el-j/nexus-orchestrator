package services

import (
	"testing"
	"time"
)

func TestStaleTTL_BelowThreshold(t *testing.T) {
	// 0, 1, 2 consecutive failures → healthCacheTTL
	for _, n := range []int{0, 1, 2} {
		h := &providerHealth{consecutiveFails: n}
		if got := staleTTL(h); got != healthCacheTTL {
			t.Errorf("consecutiveFails=%d: expected %v, got %v", n, healthCacheTTL, got)
		}
	}
}

func TestStaleTTL_AtThreshold(t *testing.T) {
	// Exactly 3 consecutive failures: exp=0 → still healthCacheTTL
	h := &providerHealth{consecutiveFails: 3}
	if got := staleTTL(h); got != healthCacheTTL {
		t.Errorf("consecutiveFails=3: expected %v, got %v", healthCacheTTL, got)
	}
}

func TestStaleTTL_CircuitBreakerGrows(t *testing.T) {
	// 4+ consecutive failures should double the TTL each extra step.
	prev := staleTTL(&providerHealth{consecutiveFails: 3})
	for n := 4; n <= 8; n++ {
		h := &providerHealth{consecutiveFails: n}
		got := staleTTL(h)
		if got <= prev {
			t.Errorf("consecutiveFails=%d: expected TTL > %v, got %v", n, prev, got)
		}
		if got > maxBackoffTTL {
			t.Errorf("consecutiveFails=%d: TTL %v exceeds maxBackoffTTL %v", n, got, maxBackoffTTL)
		}
		prev = got
	}
}

func TestStaleTTL_CappedAtMaxBackoff(t *testing.T) {
	// Very large fail count must still return maxBackoffTTL.
	h := &providerHealth{consecutiveFails: 100}
	if got := staleTTL(h); got != maxBackoffTTL {
		t.Errorf("expected maxBackoffTTL=%v, got %v", maxBackoffTTL, got)
	}
}

func TestStaleTTL_ValueAt4Fails(t *testing.T) {
	// At 4 consecutive failures the TTL must exceed 30 s (the base healthCacheTTL).
	h := &providerHealth{consecutiveFails: 4}
	got := staleTTL(h)
	if got <= 30*time.Second {
		t.Errorf("expected TTL > 30s after 4 consecutive failures, got %v", got)
	}
}
