package services_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

// memUsage is an in-memory usage ledger.
type memUsage struct {
	mu   sync.Mutex
	recs []domain.UsageRecord
	err  error
}

func (m *memUsage) Record(_ context.Context, r domain.UsageRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.recs = append(m.recs, r)
	return nil
}

func (m *memUsage) Summarize(_ context.Context, _ time.Time) ([]domain.UsageSummary, error) {
	return m.SummarizeForProject(context.Background(), time.Time{}, "")
}

func (m *memUsage) SummarizeForProject(_ context.Context, _ time.Time, projectPath string) ([]domain.UsageSummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	out := []domain.UsageSummary{}
	for _, r := range m.recs {
		if projectPath == "" || r.ProjectPath == projectPath {
			out = append(out, domain.UsageSummary{
				Provider:     r.Provider,
				Model:        r.Model,
				Role:         r.Role,
				ProjectPath:  r.ProjectPath,
				Calls:        1,
				InputTokens:  r.InputTokens,
				OutputTokens: r.OutputTokens,
				CachedTokens: r.CachedTokens,
			})
		}
	}
	return out, nil
}

// usageLLM is a scriptLLM that reports usage like the real adapters do.
type usageLLM struct {
	*scriptLLM
	u  domain.Usage
	ok bool
}

func (l *usageLLM) LastUsage() (domain.Usage, bool) { return l.u, l.ok }

var _ ports.UsageReporter = (*usageLLM)(nil)

func TestUsageIsRecordedAfterASuccessfulCall(t *testing.T) {
	llm := &usageLLM{scriptLLM: &scriptLLM{name: "Anthropic", alive: true, model: "claude-x"},
		u: domain.Usage{Model: "claude-x-1", InputTokens: 120, OutputTokens: 30, CachedTokens: 5, CostUSD: 0.01}, ok: true}
	e := newEngine(t, &noopWriter{}, nil, nil, llm)
	ledger := &memUsage{}
	services.WithUsageRepo(ledger)(e.orch)

	e.runOne(e.queue(domain.Task{Instruction: "count", Role: "architect"}), domain.StatusCompleted)
	if len(ledger.recs) != 1 {
		t.Fatalf("want one record, got %+v", ledger.recs)
	}
	r := ledger.recs[0]
	if r.Provider != "Anthropic" || r.Role != "architect" || r.TaskID != "t-count" || r.Model != "claude-x-1" ||
		r.InputTokens != 120 || r.OutputTokens != 30 || r.CachedTokens != 5 || r.At.IsZero() {
		t.Errorf("record: %+v", r)
	}
	sums, err := e.orch.GetUsage(context.Background(), time.Time{})
	if err != nil || len(sums) != 1 || sums[0].Provider != "Anthropic" {
		t.Errorf("GetUsage: %+v %v", sums, err)
	}
}

func TestUsageUsesTheConfiguredModelWhenTheProviderNamesNone(t *testing.T) {
	llm := &usageLLM{scriptLLM: &scriptLLM{name: "P", alive: true, model: "configured"}, u: domain.Usage{InputTokens: 1}, ok: true}
	e := newEngine(t, &noopWriter{}, nil, nil, llm)
	ledger := &memUsage{}
	services.WithUsageRepo(ledger)(e.orch)
	e.runOne(e.queue(domain.Task{Instruction: "x"}), domain.StatusCompleted)
	if len(ledger.recs) != 1 || ledger.recs[0].Model != "configured" {
		t.Errorf("%+v", ledger.recs)
	}
}

func TestNothingIsRecordedWithoutReportedUsageOrAFailedCall(t *testing.T) {
	// a provider that reports nothing
	quiet := &usageLLM{scriptLLM: &scriptLLM{name: "Quiet", alive: true}, ok: false}
	e := newEngine(t, &noopWriter{}, nil, nil, quiet)
	ledger := &memUsage{}
	services.WithUsageRepo(ledger)(e.orch)
	e.runOne(e.queue(domain.Task{Instruction: "a"}), domain.StatusCompleted)
	// a provider that does not implement the capability at all
	plain := &scriptLLM{name: "Plain", alive: true}
	e2 := newEngine(t, &noopWriter{}, nil, nil, plain)
	services.WithUsageRepo(ledger)(e2.orch)
	e2.runOne(e2.queue(domain.Task{Instruction: "b"}), domain.StatusCompleted)
	// a failing call
	failing := &usageLLM{scriptLLM: &scriptLLM{name: "Fail", alive: true, errs: []error{errors.New("boom"), errors.New("boom"), errors.New("boom")}}, u: domain.Usage{InputTokens: 9}, ok: true}
	e3 := newEngine(t, &noopWriter{}, nil, nil, failing)
	services.WithUsageRepo(ledger)(e3.orch)
	e3.runOne(e3.queue(domain.Task{Instruction: "c"}), domain.StatusQueued)
	if len(ledger.recs) != 0 {
		t.Errorf("nothing to record, got %+v", ledger.recs)
	}
}

func TestALedgerErrorNeverFailsTheTask(t *testing.T) {
	llm := &usageLLM{scriptLLM: &scriptLLM{name: "P", alive: true}, u: domain.Usage{InputTokens: 1}, ok: true}
	e := newEngine(t, &noopWriter{}, nil, nil, llm)
	services.WithUsageRepo(&memUsage{err: errors.New("disk full")})(e.orch)
	e.runOne(e.queue(domain.Task{Instruction: "x"}), domain.StatusCompleted)
}

func TestGetUsageWithoutALedgerIsEmpty(t *testing.T) {
	e := newEngine(t, &noopWriter{}, nil, nil, &scriptLLM{name: "P", alive: true})
	sums, err := e.orch.GetUsage(context.Background(), time.Time{})
	if err != nil || sums == nil || len(sums) != 0 {
		t.Errorf("%+v %v", sums, err)
	}
	projSums, err := e.orch.GetUsageForProject(context.Background(), time.Time{}, "/proj")
	if err != nil || projSums == nil || len(projSums) != 0 {
		t.Errorf("%+v %v", projSums, err)
	}
	if err := e.orch.RecordUsage(context.Background(), domain.UsageRecord{}); err != nil {
		t.Errorf("RecordUsage without ledger should be nil, got: %v", err)
	}
}

func TestGetUsageForProjectAndRecordUsage(t *testing.T) {
	ledger := &memUsage{}
	e := newEngine(t, &noopWriter{}, nil, nil, &scriptLLM{name: "P", alive: true})
	services.WithUsageRepo(ledger)(e.orch)

	err := e.orch.RecordUsage(context.Background(), domain.UsageRecord{
		ProjectPath: "/projects/a",
		Provider:    "Claude",
		Usage:       domain.Usage{InputTokens: 50, OutputTokens: 25},
	})
	if err != nil {
		t.Fatalf("RecordUsage failed: %v", err)
	}

	_ = e.orch.RecordUsage(context.Background(), domain.UsageRecord{
		ProjectPath: "/projects/b",
		Provider:    "Claude",
		Usage:       domain.Usage{InputTokens: 100, OutputTokens: 75},
	})

	a, err := e.orch.GetUsageForProject(context.Background(), time.Time{}, "/projects/a")
	if err != nil || len(a) != 1 || a[0].InputTokens != 50 {
		t.Fatalf("expected 1 record for /projects/a: %+v, %v", a, err)
	}

	all, err := e.orch.GetUsageForProject(context.Background(), time.Time{}, "")
	if err != nil || len(all) != 2 {
		t.Fatalf("expected all records when projectPath is empty: %+v, %v", all, err)
	}
}
