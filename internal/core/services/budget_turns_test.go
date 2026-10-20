package services_test

import (
	"errors"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

// heavyLLM is a provider whose every call reports 1200 tokens, more than the 1000 a day the tests allow.
func heavyLLM(name string, replies ...string) *usageLLM {
	return &usageLLM{scriptLLM: &scriptLLM{name: name, alive: true, replies: replies},
		u: domain.Usage{InputTokens: 1000, OutputTokens: 200}, ok: true}
}

// A provider that used up its budget during the first attempt must not be asked for the correction turn.
func TestBudget_ACorrectionTurnMovesToAProviderWithBudgetLeft(t *testing.T) {
	a := heavyLLM("anthropic-main", "draft")
	b := heavyLLM("ollama-backup", "fixed")
	runner := &scriptedRunner{results: []runResult{{"fail", errors.New("exit 1")}, {"PASS", nil}}}
	e := newEngine(t, &contextWriter{}, nil, runner, a, b)
	services.WithUsageRepo(&memUsage{})(e.orch)
	dir := ladderProject(t, `{"budgets":{"anthropic-main":{"dailyTokens":1000}}}`)

	got := e.runOne(e.queue(domain.Task{Instruction: "x", ProjectPath: dir, TargetFile: "f.go", VerificationCommand: "go test"}), domain.StatusCompleted)
	if a.calls != 1 || b.calls != 1 || !strings.Contains(got.Logs, "daily budget of 1000 tokens") {
		t.Errorf("the correction goes to the other provider: a=%d b=%d logs=%s", a.calls, b.calls, got.Logs)
	}
}

// With nobody left, the task waits for the budget to reset instead of spending past it or failing.
func TestBudget_ACorrectionTurnWithNoProviderLeftWaits(t *testing.T) {
	a := heavyLLM("anthropic-main", "draft", "never asked")
	runner := &scriptedRunner{results: []runResult{{"fail", errors.New("exit 1")}, {"PASS", nil}}}
	e := newEngine(t, &contextWriter{}, nil, runner, a)
	services.WithUsageRepo(&memUsage{})(e.orch)
	dir := ladderProject(t, `{"budgets":{"anthropic-main":{"dailyTokens":1000}}}`)

	got := e.runOne(e.queue(domain.Task{Instruction: "x", ProjectPath: dir, TargetFile: "f.go", VerificationCommand: "go test"}), domain.StatusWaiting)
	if a.calls != 1 || got.WaitUntil.IsZero() {
		t.Errorf("a=%d waitUntil=%v", a.calls, got.WaitUntil)
	}
}
