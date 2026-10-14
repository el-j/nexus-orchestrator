package services_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

func writePolicy(t *testing.T, root, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".nexus"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, services.PolicyFile), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A fresh session must be told the project's rules first, where a tight token budget cannot cut them.
func TestOnboardingLeadsWithTheProjectRules(t *testing.T) {
	root := t.TempDir()
	writePolicy(t, root, `{"verify":"make check","maxFileLines":250,"testsFirst":true,"rules":["Prefer small PRs."]}`)
	repo := &mockKnowledgeRepo{entries: []domain.ProjectKnowledge{{
		ProjectPath: filepath.Clean(root), Kind: domain.KnowledgeArchitecture, Topic: "Layers",
		Content: strings.Repeat("a long architecture paragraph ", 40),
	}}}
	b := services.NewBrainService(repo, nil)

	out, err := b.GetOnboardingContext(context.Background(), root, 200)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"## Workflow rules", "`make check`", "250 lines", "Tests first", "Prefer small PRs."} {
		if !strings.Contains(out, want) {
			t.Errorf("onboarding lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "## Verification Commands") {
		t.Errorf("a guessed verification list must not contradict the policy:\n%s", out)
	}
	rules, arch := strings.Index(out, "## Workflow rules"), strings.Index(out, "## Architectural Overview")
	if arch >= 0 && rules > arch {
		t.Errorf("rules must come before architecture so truncation cuts the tail, not the rules")
	}
}

func TestOnboardingReportsABrokenPolicy(t *testing.T) {
	root := t.TempDir()
	writePolicy(t, root, `{"maxFileLine": 1}`)
	out, err := services.NewBrainService(&mockKnowledgeRepo{}, nil).GetOnboardingContext(context.Background(), root, 800)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "policy file is invalid") {
		t.Errorf("a broken policy must be reported, not skipped:\n%s", out)
	}
}

func TestOnboardingWithoutAPolicyKeepsTheGuessedCommands(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module x"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _ := services.NewBrainService(&mockKnowledgeRepo{}, nil).GetOnboardingContext(context.Background(), root, 800)
	if !strings.Contains(out, "## Verification Commands") || strings.Contains(out, "## Workflow rules") {
		t.Errorf("without a policy the old behaviour must stay:\n%s", out)
	}
}

func TestInitProjectCreatesAPolicyOnce(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module x"), 0o644); err != nil {
		t.Fatal(err)
	}
	b := services.NewBrainService(&mockKnowledgeRepo{}, nil)
	if _, err := b.InitProject(context.Background(), root, ""); err != nil {
		t.Fatal(err)
	}
	p, ok, err := services.LoadProjectPolicy(root)
	if err != nil || !ok || p.Verify != "go vet ./... && go test ./..." {
		t.Fatalf("init must create a usable default policy: ok=%v err=%v p=%+v", ok, err, p)
	}
	writePolicy(t, root, `{"verify":"mine"}`)
	if _, err := b.InitProject(context.Background(), root, ""); err != nil {
		t.Fatal(err)
	}
	if p, _, _ := services.LoadProjectPolicy(root); p.Verify != "mine" {
		t.Errorf("init overwrote the project's own policy: %+v", p)
	}
}

func TestInitProjectSurvivesAnUnwritablePolicyLocation(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".nexus"), []byte("a file"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := services.NewBrainService(&mockKnowledgeRepo{}, nil).InitProject(context.Background(), root, ""); err != nil {
		t.Errorf("a policy that cannot be written must not fail the knowledge init: %v", err)
	}
}
