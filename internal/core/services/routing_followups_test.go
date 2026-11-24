package services_test

import (
	"errors"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

// Subscriptions are flat-rate: a plan model is not penalised for cost, so a routine task goes to the
// small model of the plan and real work to the strong one of the same plan.
func TestFlatRateProvidersCostNothingInTheRouting(t *testing.T) {
	strong := &mockLLMClient{alive: true, name: "vscode-copilot-claude-sonnet-5.5", activeModel: "claude-sonnet-5.5"}
	paid := &mockLLMClient{alive: true, name: "Anthropic API", activeModel: "claude-sonnet-5.5"}
	profile := services.EstimateTaskAffinity(domain.Task{Instruction: "implement the feature"}, 0)
	flat, metered := services.ScoreProviderAffinity(strong, profile), services.ScoreProviderAffinity(paid, profile)
	if flat.CostScore != 0 || metered.CostScore != 1 {
		t.Errorf("cost: flat-rate %v, metered %v", flat.CostScore, metered.CostScore)
	}
	if flat.TotalAffinity <= metered.TotalAffinity {
		t.Errorf("a flat-rate model must rank above the same metered one")
	}
	for _, name := range []string{"Claude (subscription)", "Antigravity (agy)", "vscode-copilot-gpt-6-sol"} {
		c := &mockLLMClient{alive: true, name: name, activeModel: "claude-sonnet-5.5"}
		if services.ScoreProviderAffinity(c, profile).CostScore != 0 {
			t.Errorf("%s is flat-rate", name)
		}
	}
}

// A task without a target file that has a check to pass cannot be satisfied by text: it runs as an
// agent that edits the project, and the check decides.
func TestATaskWithoutTargetFileAndACheckRunsAsAnAgent(t *testing.T) {
	llm := &scriptLLM{name: "Anthropic", alive: true, limit: 100000, replies: []string{
		tool(`{"tool":"write_file","path":"a.go","content":"package a"}`),
		tool(`{"tool":"finish","summary":"done"}`),
	}}
	runner := &scriptedRunner{results: []runResult{{"PASS", nil}}}
	fs := newMemFS(map[string]string{})
	e := newEngine(t, fs, nil, runner, llm)
	id := e.queue(domain.Task{Instruction: "no target", VerificationCommand: "check"})
	got := e.runOne(id, domain.StatusCompleted)
	if _, ok := fs.files["a.go"]; !ok {
		t.Errorf("the model did not get to edit the project: logs %q", got.Logs)
	}
}

// The project's default check is for tasks that change code: a plan task does not inherit it, and a task
// that says its own check keeps it.
func TestThePolicyCheckIsNotAppliedToPlanTasks(t *testing.T) {
	dir := ladderProject(t, `{"version":1,"verify":"cargo fmt --check"}`)
	llm := &mockLLMClient{alive: true, name: "p", code: "plan text"}
	runner := &scriptedRunner{results: []runResult{{"red", errors.New("exit 1")}}}
	e := newEngine(t, newMemFS(map[string]string{}), nil, runner, llm)
	id := e.queue(domain.Task{Instruction: "plan it", Command: domain.CommandPlan, ProjectPath: dir})
	got := e.runOne(id, domain.StatusCompleted)
	if strings.Contains(got.Logs, "verification") {
		t.Errorf("a plan task was verified: %q", got.Logs)
	}
	var _ ports.LLMClient = llm
}
