package services_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

// quotaLLM is a scriptLLM that reports the plan's own quota reading like the Claude driver does.
type quotaLLM struct {
	*scriptLLM
	q  domain.QuotaReading
	ok bool
}

func (l *quotaLLM) LastQuota() (domain.QuotaReading, bool) { return l.q, l.ok }

var _ ports.QuotaReporter = (*quotaLLM)(nil)

func reading(util float64, resets time.Time) domain.QuotaReading {
	return domain.QuotaReading{Window: "seven_day", Status: "allowed_warning", Utilization: util, ResetsAt: resets}
}

// warm makes llm report q on a call: a task pinned to it, so the engine reads the reading afterwards.
func warm(e *engineFixture, llm *quotaLLM, q domain.QuotaReading, name string) {
	llm.q, llm.ok = q, true
	e.runOne(e.queue(domain.Task{Instruction: "warm-" + name, Role: "architect", ProviderName: llm.name}), domain.StatusCompleted)
}

// two providers: "Claude (subscription)" first in the chain (frontier role), "Ollama" as the fallback.
func quotaFixture(t *testing.T, primary *quotaLLM) (*engineFixture, *scriptLLM) {
	t.Helper()
	backup := &scriptLLM{name: "ollama-backup", alive: true, replies: []string{"from backup"}}
	return newEngine(t, &noopWriter{}, nil, nil, primary, backup), backup
}

func TestQuotaRouting_AProviderNearItsLimitIsTriedAfterTheOthers(t *testing.T) {
	primary := &quotaLLM{scriptLLM: &scriptLLM{name: "anthropic-main", alive: true, replies: []string{"from primary"}}}
	e, backup := quotaFixture(t, primary)
	e.runOne(e.queue(domain.Task{Instruction: "blind", Role: "architect"}), domain.StatusCompleted)
	if primary.calls != 1 {
		t.Fatalf("with no reading the chain order stands, primary calls=%d", primary.calls)
	}
	warm(e, primary, reading(0.95, time.Now().Add(48*time.Hour)), "hot") // primary call 2, now known to be at 95 %
	e.runOne(e.queue(domain.Task{Instruction: "routed", Role: "architect"}), domain.StatusCompleted)
	if primary.calls != 2 || backup.calls != 1 {
		t.Errorf("a provider at 95%% must come after the others: primary=%d backup=%d", primary.calls, backup.calls)
	}
}

func TestQuotaRouting_ThePolicyThresholdDecides(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".nexus"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(body string) {
		if err := os.WriteFile(filepath.Join(project, services.PolicyFile), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	primary := &quotaLLM{scriptLLM: &scriptLLM{name: "anthropic-main", alive: true, replies: []string{"p"}}}
	e, backup := quotaFixture(t, primary)
	warm(e, primary, reading(0.95, time.Now().Add(time.Hour)), "x") // primary call 1

	write(`{"quotaThreshold":0.99}`)
	e.runOne(e.queue(domain.Task{Instruction: "lenient", Role: "architect", ProjectPath: project}), domain.StatusCompleted)
	if primary.calls != 2 || backup.calls != 0 {
		t.Errorf("at 95%% with a 0.99 threshold the primary is still first: primary=%d backup=%d", primary.calls, backup.calls)
	}
	write(`{"quotaThreshold":0.5}`)
	e.runOne(e.queue(domain.Task{Instruction: "strict", Role: "architect", ProjectPath: project}), domain.StatusCompleted)
	if primary.calls != 2 || backup.calls != 1 {
		t.Errorf("at 95%% with a 0.5 threshold the backup goes first: primary=%d backup=%d", primary.calls, backup.calls)
	}
}

func TestQuotaRouting_AStaleReadingAndAPinnedProviderAreIgnored(t *testing.T) {
	primary := &quotaLLM{scriptLLM: &scriptLLM{name: "anthropic-main", alive: true, replies: []string{"p"}}}
	e, backup := quotaFixture(t, primary)
	// The window reset an hour ago: the old reading says nothing about now.
	warm(e, primary, reading(0.99, time.Now().Add(-time.Hour)), "stale") // primary call 1
	e.runOne(e.queue(domain.Task{Instruction: "stale", Role: "architect"}), domain.StatusCompleted)
	if primary.calls != 2 || backup.calls != 0 {
		t.Errorf("a reading past its reset is ignored: primary=%d backup=%d", primary.calls, backup.calls)
	}
	// A task that names its provider gets it, whatever the reading.
	warm(e, primary, reading(0.99, time.Now().Add(time.Hour)), "hot") // primary call 3
	e.runOne(e.queue(domain.Task{Instruction: "pinned", Role: "architect", ProviderName: "anthropic-main"}), domain.StatusCompleted)
	if primary.calls != 4 {
		t.Errorf("a pinned provider is used: primary=%d", primary.calls)
	}
}

func TestQuotaRouting_WhenEveryProviderIsNearItsLimitTheOrderStands(t *testing.T) {
	primary := &quotaLLM{scriptLLM: &scriptLLM{name: "anthropic-main", alive: true, replies: []string{"p"}}}
	backup := &quotaLLM{scriptLLM: &scriptLLM{name: "ollama-backup", alive: true, replies: []string{"b"}}}
	e := newEngine(t, &noopWriter{}, nil, nil, primary, backup)
	warm(e, primary, reading(0.95, time.Now().Add(time.Hour)), "a") // primary call 1
	warm(e, backup, reading(0.97, time.Now().Add(time.Hour)), "b")
	e.runOne(e.queue(domain.Task{Instruction: "all hot", Role: "architect"}), domain.StatusCompleted)
	if primary.calls != 2 || backup.calls != 1 {
		t.Errorf("nothing is better, so nothing is reordered: primary=%d backup=%d", primary.calls, backup.calls)
	}
}

func TestQuotaIsReadAfterACallAndShownInTheProviderList(t *testing.T) {
	primary := &quotaLLM{scriptLLM: &scriptLLM{name: "anthropic-main", alive: true, replies: []string{"p"}}}
	e := newEngine(t, &noopWriter{}, nil, nil, primary)
	warm(e, primary, reading(0.42, time.Now().Add(72*time.Hour)), "show")
	infos, err := e.orch.GetProviders()
	if err != nil || len(infos) != 1 || infos[0].Quota == nil {
		t.Fatalf("providers: %+v %v", infos, err)
	}
	if q := infos[0].Quota; q.Utilization != 0.42 || q.Window != "seven_day" {
		t.Errorf("quota: %+v", q)
	}
	// Once the window has reset the reading is no longer shown.
	warm(e, primary, reading(0.42, time.Now().Add(40*time.Millisecond)), "soon")
	time.Sleep(80 * time.Millisecond)
	if infos, _ := e.orch.GetProviders(); infos[0].Quota != nil {
		t.Errorf("a reading past its reset must disappear: %+v", infos[0].Quota)
	}
}

func TestAQuotaRejectedCallPutsTheProviderOnCooldownUntilTheReset(t *testing.T) {
	reset := time.Now().Add(3 * time.Hour)
	limited := &domain.ProviderError{Provider: "anthropic-main", Kind: domain.ErrKindQuotaExhausted, ResetAt: reset, Msg: "anthropic-main: seven_day limit reached"}
	primary := &quotaLLM{scriptLLM: &scriptLLM{name: "anthropic-main", alive: true, errs: []error{limited}},
		q: domain.QuotaReading{Window: "seven_day", Status: "rejected", Utilization: 1, ResetsAt: reset}, ok: true}
	e, backup := quotaFixture(t, primary)
	got := e.runOne(e.queue(domain.Task{Instruction: "limit", Role: "architect"}), domain.StatusCompleted)
	if !strings.Contains(got.Logs, "failover") || backup.calls != 1 {
		t.Fatalf("must fail over: %s", got.Logs)
	}
	infos, _ := e.orch.GetProviders()
	for _, p := range infos {
		if p.Name == "anthropic-main" && (!p.RateLimited || p.RateLimitedUntil.Sub(reset).Abs() > time.Second) {
			t.Errorf("cooldown must end at the provider's reset time: %+v", p)
		}
	}
}
