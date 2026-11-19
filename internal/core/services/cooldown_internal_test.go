package services

import (
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

func TestCooldownBook(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	var b cooldownBook
	if _, ok := b.active("a|m", now); ok {
		t.Fatal("nothing is cooling down yet")
	}
	b.mark("a|m", now.Add(time.Minute), domain.ErrKindRateLimited, "a: rate limited (429)")
	c, ok := b.active("a|m", now)
	if !ok || !c.until.Equal(now.Add(time.Minute)) || c.kind != domain.ErrKindRateLimited || c.reason == "" {
		t.Fatalf("active: %+v %v", c, ok)
	}
	if _, ok := b.active("a|other", now); ok {
		t.Error("another model is not cooling down")
	}
	if _, ok := b.active("a|m", now.Add(time.Minute)); ok {
		t.Error("expired at the reset time")
	}
	// A later reset replaces an earlier one; an earlier one does not shorten it.
	b.mark("a|m", now.Add(5*time.Minute), domain.ErrKindQuotaExhausted, "quota")
	b.mark("a|m", now.Add(2*time.Minute), domain.ErrKindRateLimited, "short")
	if c, _ := b.active("a|m", now); !c.until.Equal(now.Add(5*time.Minute)) || c.kind != domain.ErrKindQuotaExhausted {
		t.Errorf("longest wins: %+v", c)
	}
	b.clear("a|m")
	if _, ok := b.active("a|m", now); ok {
		t.Error("cleared")
	}
	b.clear("never-set") // must not panic
	// Earliest end across several keys, for the engine's "wait until" (TASK-587).
	b.mark("x|1", now.Add(10*time.Minute), domain.ErrKindRateLimited, "x")
	b.mark("y|1", now.Add(3*time.Minute), domain.ErrKindRateLimited, "y")
	if u, ok := b.earliest([]string{"x|1", "y|1"}, now); !ok || !u.Equal(now.Add(3*time.Minute)) {
		t.Errorf("earliest: %v %v", u, ok)
	}
	if _, ok := b.earliest([]string{"x|1", "free|1"}, now); ok {
		t.Error("a provider that is free means there is nothing to wait for")
	}
	if _, ok := b.earliest(nil, now); ok {
		t.Error("empty list")
	}
}
