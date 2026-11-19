package cli_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

type policyScripted struct {
	scriptedOrch
	policy  domain.ProjectPolicy
	found   bool
	getErr  error
	setErr  error
	project string
	saved   *domain.ProjectPolicy
}

func (p *policyScripted) GetProjectPolicy(_ context.Context, project string) (domain.ProjectPolicy, bool, error) {
	p.project = project
	return p.policy, p.found, p.getErr
}

func (p *policyScripted) SetProjectPolicy(_ context.Context, project string, pol domain.ProjectPolicy) error {
	p.project, p.saved = project, &pol
	return p.setErr
}

func TestPolicyCmd_ShowPrintsTheFileOrSaysThereIsNone(t *testing.T) {
	orch := &policyScripted{found: true, policy: domain.ProjectPolicy{QuotaThreshold: 0.8, Budgets: map[string]domain.ProviderBudget{"Claude": {DailyTokens: 300000}}}}
	out, err := execute(orch, nil, "policy", "show", "--project", "/p")
	if err != nil || orch.project != "/p" || !strings.Contains(out, `"quotaThreshold": 0.8`) || !strings.Contains(out, `"dailyTokens": 300000`) {
		t.Fatalf("show: %v\n%s", err, out)
	}
	orch.found = false
	if out, _ := execute(orch, nil, "policy", "show", "--project", "/p"); !strings.Contains(out, "no policy file") {
		t.Errorf("none: %s", out)
	}
	orch.getErr = errors.New("policy: broken")
	if _, err := execute(orch, nil, "policy", "show", "--project", "/p"); err == nil || !strings.Contains(err.Error(), "broken") {
		t.Errorf("error: %v", err)
	}
	if _, err := execute(&scriptedOrch{}, nil, "policy", "show", "--project", "/p"); err == nil || !strings.Contains(err.Error(), "not available") {
		t.Errorf("no manager: %v", err)
	}
	if _, err := execute(orch, nil, "policy", "show"); err == nil {
		t.Error("--project is required")
	}
}

func TestPolicyCmd_SetChangesOnlyTheGivenFields(t *testing.T) {
	orch := &policyScripted{found: true, policy: domain.ProjectPolicy{
		Verify: "go test", ContextBudget: 100,
		Budgets: map[string]domain.ProviderBudget{"Claude": {DailyTokens: 5}, "Gemini": {DailyTokens: 9}},
		Ladder:  map[string][]string{"coder": {"Ollama/*"}, "tester": {"X/*"}},
	}}
	out, err := execute(orch, nil, "policy", "set", "--project", "/p",
		"--quota-threshold", "0.7", "--history-budget", "2000",
		"--budget", "Claude=300000", "--budget", "Gemini=0",
		"--ladder", "coder=Ollama/*,Anthropic/claude-sonnet*", "--ladder", "tester=")
	if err != nil || orch.saved == nil {
		t.Fatalf("set: %v\n%s", err, out)
	}
	s := *orch.saved
	if s.Verify != "go test" || s.ContextBudget != 100 {
		t.Errorf("untouched fields stay: %+v", s)
	}
	if s.QuotaThreshold != 0.7 || s.HistoryBudget != 2000 {
		t.Errorf("flags: %+v", s)
	}
	if s.Budgets["Claude"].DailyTokens != 300000 {
		t.Errorf("budget set: %+v", s.Budgets)
	}
	if _, still := s.Budgets["Gemini"]; still {
		t.Errorf("=0 removes a budget: %+v", s.Budgets)
	}
	if len(s.Ladder["coder"]) != 2 || s.Ladder["coder"][1] != "Anthropic/claude-sonnet*" {
		t.Errorf("ladder set: %+v", s.Ladder)
	}
	if _, still := s.Ladder["tester"]; still {
		t.Errorf("an empty ladder removes the role: %+v", s.Ladder)
	}

	// Nothing given is an error; so are malformed pairs and a refused save.
	if _, err := execute(orch, nil, "policy", "set", "--project", "/p"); err == nil {
		t.Error("nothing to change must be an error")
	}
	for _, args := range [][]string{{"--budget", "Claude"}, {"--budget", "Claude=lots"}, {"--ladder", "coder"}} {
		if _, err := execute(orch, nil, append([]string{"policy", "set", "--project", "/p"}, args...)...); err == nil {
			t.Errorf("%v must be refused", args)
		}
	}
	orch.setErr = errors.New("policy: quotaThreshold must be between 0 and 1")
	if _, err := execute(orch, nil, "policy", "set", "--project", "/p", "--quota-threshold", "3"); err == nil || !strings.Contains(err.Error(), "quotaThreshold") {
		t.Errorf("refused save: %v", err)
	}
	orch.setErr, orch.getErr = nil, errors.New("policy: broken file")
	if _, err := execute(orch, nil, "policy", "set", "--project", "/p", "--quota-threshold", "0.5"); err == nil {
		t.Error("a broken file stops the change")
	}
	if _, err := execute(&scriptedOrch{}, nil, "policy", "set", "--project", "/p", "--quota-threshold", "0.5"); err == nil || !strings.Contains(err.Error(), "not available") {
		t.Errorf("no manager: %v", err)
	}
}

func TestPolicyCmd_SetOnAProjectWithoutAFileStartsFromAnEmptyPolicy(t *testing.T) {
	orch := &policyScripted{}
	if _, err := execute(orch, nil, "policy", "set", "--project", "/p", "--context-budget", "4000"); err != nil || orch.saved == nil || orch.saved.ContextBudget != 4000 {
		t.Fatalf("%v %+v", err, orch.saved)
	}
}
