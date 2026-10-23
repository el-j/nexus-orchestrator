package services_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

func newPolicyAdmin(t *testing.T) *services.OrchestratorService {
	t.Helper()
	return newEngine(t, &noopWriter{}, nil, nil, &scriptLLM{name: "p", alive: true}).orch
}

func TestPolicyAdmin_SetThenGetRoundTripsAndCreatesTheFile(t *testing.T) {
	o := newPolicyAdmin(t)
	dir := t.TempDir()
	if _, found, err := o.GetProjectPolicy(t.Context(), dir); err != nil || found {
		t.Fatalf("no file yet: found=%v err=%v", found, err)
	}
	want := domain.ProjectPolicy{
		Verify:         "go test ./...",
		QuotaThreshold: 0.8,
		ContextBudget:  4000,
		Ladder:         map[string][]string{"coder": {"Ollama/*", "Anthropic/claude-sonnet*"}},
		Budgets:        map[string]domain.ProviderBudget{"Anthropic": {DailyTokens: 300000}},
	}
	if err := o.SetProjectPolicy(t.Context(), dir, want); err != nil {
		t.Fatal(err)
	}
	got, found, err := o.GetProjectPolicy(t.Context(), dir)
	if err != nil || !found {
		t.Fatalf("found=%v err=%v", found, err)
	}
	if got.QuotaThreshold != 0.8 || got.Budgets["Anthropic"].DailyTokens != 300000 || len(got.Ladder["coder"]) != 2 || got.Verify != "go test ./..." {
		t.Errorf("round trip: %+v", got)
	}
	// The daemon's own loader reads what was written (so the engine honours it).
	if _, ok, err := services.LoadProjectPolicy(dir); err != nil || !ok {
		t.Errorf("LoadProjectPolicy: ok=%v err=%v", ok, err)
	}
}

func TestPolicyAdmin_RefusesInvalidPoliciesAndUnsafeTargetsAndLeavesTheFileAlone(t *testing.T) {
	o := newPolicyAdmin(t)
	dir := ladderProject(t, `{"verify":"keep me"}`)
	file := filepath.Join(dir, services.PolicyFile)

	for name, bad := range map[string]domain.ProjectPolicy{
		"threshold": {QuotaThreshold: 2},
		"budget":    {Budgets: map[string]domain.ProviderBudget{"X": {DailyTokens: -1}}},
		"ladder":    {Ladder: map[string][]string{"coder": {""}}},
	} {
		if err := o.SetProjectPolicy(t.Context(), dir, bad); err == nil {
			t.Errorf("%s: an invalid policy must be refused", name)
		}
	}
	if b, _ := os.ReadFile(file); !strings.Contains(string(b), "keep me") {
		t.Errorf("a refused policy must not touch the file: %s", b)
	}

	for _, path := range []string{"", "relative/dir", filepath.Join(dir, "missing")} {
		if err := o.SetProjectPolicy(t.Context(), path, domain.ProjectPolicy{}); err == nil {
			t.Errorf("project path %q must be refused", path)
		}
		if _, _, err := o.GetProjectPolicy(t.Context(), path); err == nil {
			t.Errorf("reading project path %q must be refused", path)
		}
	}
	// A symlinked policy file is not followed.
	linked := t.TempDir()
	_ = os.MkdirAll(filepath.Join(linked, ".nexus"), 0o755)
	target := filepath.Join(t.TempDir(), "elsewhere.json")
	_ = os.WriteFile(target, []byte("{}"), 0o644)
	_ = os.Symlink(target, filepath.Join(linked, services.PolicyFile))
	if err := o.SetProjectPolicy(t.Context(), linked, domain.ProjectPolicy{Verify: "x"}); err != nil {
		t.Fatalf("the link is replaced, not followed: %v", err)
	}
	if b, _ := os.ReadFile(target); string(b) != "{}" {
		t.Errorf("the file behind the link must stay untouched: %s", b)
	}
}

func TestPolicyAdmin_AFileThatCannotBeReadIsReportedOnGet(t *testing.T) {
	o := newPolicyAdmin(t)
	dir := ladderProject(t, `{"nonsense":true}`)
	if _, _, err := o.GetProjectPolicy(t.Context(), dir); err == nil {
		t.Error("a broken policy file must be an error, not an empty policy")
	}
}

func TestPolicyAdmin_WhenTheNexusFolderCannotBeMadeTheErrorIsReported(t *testing.T) {
	o := newPolicyAdmin(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".nexus"), []byte("a file, not a folder"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := o.SetProjectPolicy(t.Context(), dir, domain.ProjectPolicy{Verify: "x"}); err == nil {
		t.Error("a file where the .nexus folder belongs must be an error")
	}
	// A directory where the policy file belongs cannot be replaced by the rename either.
	dir2 := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir2, services.PolicyFile, "inner"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := o.SetProjectPolicy(t.Context(), dir2, domain.ProjectPolicy{Verify: "x"}); err == nil {
		t.Error("a non-empty directory at the policy path must be an error")
	}
}
