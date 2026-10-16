package services_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

func TestFailoverCoolsTheLimitedProviderDown(t *testing.T) {
	limited := &domain.ProviderError{Provider: "anthropic-main", Kind: domain.ErrKindRateLimited, Status: 429,
		ResetAt: time.Now().Add(time.Hour), Msg: "anthropic-main: rate limited (429)"}
	primary := &scriptLLM{name: "anthropic-main", alive: true, errs: []error{limited}, replies: []string{"never used"}}
	backup := &scriptLLM{name: "ollama-backup", alive: true, replies: []string{"from backup"}}
	e := newEngine(t, &noopWriter{}, nil, nil, primary, backup)

	first := e.runOne(e.queue(domain.Task{Instruction: "one", Role: "architect"}), domain.StatusCompleted)
	if !strings.Contains(first.Logs, "[failover") {
		t.Fatalf("first task must fail over: %s", first.Logs)
	}
	second := e.runOne(e.queue(domain.Task{Instruction: "two", Role: "architect"}), domain.StatusCompleted)
	if primary.calls != 1 {
		t.Errorf("the limited provider must be skipped, it was called %d times", primary.calls)
	}
	if !strings.Contains(second.Logs, "anthropic-main is cooling down until") {
		t.Errorf("the skip must be logged: %s", second.Logs)
	}

	infos, _ := e.orch.GetProviders()
	var seen bool
	for _, p := range infos {
		if p.Name == "anthropic-main" {
			seen = true
			if !p.RateLimited || p.RateLimitedUntil.IsZero() || !strings.Contains(p.CooldownReason, "rate limited") {
				t.Errorf("provider info: %+v", p)
			}
		}
		if p.Name == "ollama-backup" && p.RateLimited {
			t.Error("the healthy provider must not be marked")
		}
	}
	if !seen {
		t.Fatal("anthropic-main missing from GetProviders")
	}
}

func TestPlainErrorsDoNotCoolAProviderDown(t *testing.T) {
	flaky := &scriptLLM{name: "flaky", alive: true, errs: []error{errors.New("boom")}}
	backup := &scriptLLM{name: "backup", alive: true}
	e := newEngine(t, &noopWriter{}, nil, nil, flaky, backup)
	e.runOne(e.queue(domain.Task{Instruction: "a", Role: "architect"}), domain.StatusCompleted)
	e.runOne(e.queue(domain.Task{Instruction: "b", Role: "architect"}), domain.StatusCompleted)
	if flaky.calls != 2 {
		t.Errorf("an ordinary error is tried again next time, calls=%d", flaky.calls)
	}
}

func TestSuccessClearsACooldown(t *testing.T) {
	d := services.NewDiscoveryService()
	c := &scriptLLM{name: "p", alive: true, model: "m"}
	d.MarkCooldown(c, time.Now().Add(-time.Second), domain.ErrKindRateLimited, "old")
	if _, ok := d.Cooldown(c, time.Now()); ok {
		t.Fatal("an expired cooldown is not active")
	}
	d.MarkCooldown(c, time.Now().Add(time.Hour), domain.ErrKindRateLimited, "now")
	d.ClearCooldown(c)
	if _, ok := d.Cooldown(c, time.Now()); ok {
		t.Fatal("cleared")
	}
}
