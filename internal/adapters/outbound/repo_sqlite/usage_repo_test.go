package repo_sqlite_test

import (
	"context"
	"testing"
	"time"

	repo_sqlite "nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

var _ ports.UsageRepository = (*repo_sqlite.UsageRepo)(nil)

func TestUsageRepo_RecordAndSummarize(t *testing.T) {
	r := newTestRepo(t)
	u := repo_sqlite.NewUsageRepo(r)
	ctx := context.Background()
	t0 := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	rec := func(at time.Time, provider, model, role string, in, out, cached int, cost float64) domain.UsageRecord {
		return domain.UsageRecord{At: at, TaskID: "t", Provider: provider, Role: role,
			Usage: domain.Usage{Model: model, InputTokens: in, OutputTokens: out, CachedTokens: cached, CostUSD: cost}}
	}
	for _, x := range []domain.UsageRecord{
		rec(t0.Add(-48*time.Hour), "Anthropic", "opus", "architect", 1000, 1000, 0, 1), // too old for the window below
		rec(t0.Add(-1*time.Hour), "Anthropic", "opus", "architect", 100, 50, 10, 0.5),
		rec(t0.Add(-30*time.Minute), "Anthropic", "opus", "architect", 200, 70, 20, 0.25),
		rec(t0.Add(-20*time.Minute), "Ollama", "qwen", "tester", 5000, 300, 0, 0),
		rec(t0.Add(-10*time.Minute), "Anthropic", "haiku", "", 10, 5, 0, 0.01),
	} {
		if err := u.Record(ctx, x); err != nil {
			t.Fatal(err)
		}
	}
	got, err := u.Summarize(ctx, t0.Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 groups, got %+v", got)
	}
	if got[0].Provider != "Ollama" || got[0].Calls != 1 || got[0].InputTokens != 5000 {
		t.Errorf("largest consumer first: %+v", got[0])
	}
	opus := got[1]
	if opus.Provider != "Anthropic" || opus.Model != "opus" || opus.Role != "architect" || opus.Calls != 2 ||
		opus.InputTokens != 300 || opus.OutputTokens != 120 || opus.CachedTokens != 30 || opus.CostUSD != 0.75 {
		t.Errorf("opus group: %+v", opus)
	}
	if all, _ := u.Summarize(ctx, time.Time{}); len(all) != 3 || all[0].InputTokens != 5000 {
		// the zero time means "everything": the old call joins the opus group
		var opusIn int
		for _, g := range all {
			if g.Model == "opus" {
				opusIn = g.InputTokens
			}
		}
		if opusIn != 1300 {
			t.Errorf("everything: %+v", all)
		}
	}
	empty, err := u.Summarize(ctx, t0.Add(time.Hour))
	if err != nil || len(empty) != 0 {
		t.Errorf("nothing after the last call: %+v %v", empty, err)
	}
}

func TestUsageRepo_ADefaultTimeIsFilledIn(t *testing.T) {
	r := newTestRepo(t)
	u := repo_sqlite.NewUsageRepo(r)
	if err := u.Record(context.Background(), domain.UsageRecord{Provider: "P", Usage: domain.Usage{InputTokens: 1}}); err != nil {
		t.Fatal(err)
	}
	got, err := u.Summarize(context.Background(), time.Now().Add(-time.Minute))
	if err != nil || len(got) != 1 {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestUsageRepo_ClosedDatabaseIsAnError(t *testing.T) {
	u := repo_sqlite.NewUsageRepo(closedRepo(t))
	if err := u.Record(context.Background(), domain.UsageRecord{Provider: "P"}); err == nil {
		t.Error("Record on a closed database")
	}
	if _, err := u.Summarize(context.Background(), time.Time{}); err == nil {
		t.Error("Summarize on a closed database")
	}
}
