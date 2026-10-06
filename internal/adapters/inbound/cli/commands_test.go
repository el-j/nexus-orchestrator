package cli_test

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/cli"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// scriptedOrch lets a test script the task-lifecycle calls the draft, backlog,
// promote and update commands make, on top of the shared mockOrchestrator.
type scriptedOrch struct {
	mockOrchestrator
	createDraft func(domain.Task) (string, error)
	getBacklog  func(string) ([]domain.Task, error)
	promoteTask func(string) (ports.PromoteResult, error)
	updateTask  func(string, domain.Task) (domain.Task, error)
}

func (s *scriptedOrch) CreateDraft(t domain.Task) (string, error)  { return s.createDraft(t) }
func (s *scriptedOrch) GetBacklog(p string) ([]domain.Task, error) { return s.getBacklog(p) }
func (s *scriptedOrch) PromoteTask(id string) (ports.PromoteResult, error) {
	return s.promoteTask(id)
}
func (s *scriptedOrch) UpdateTask(id string, t domain.Task) (domain.Task, error) {
	return s.updateTask(id, t)
}

// execute runs the CLI and returns stdout and the command error. Output goes
// through cmd.OutOrStdout, so no os.Stdout redirection is needed.
func execute(orch ports.Orchestrator, brain ports.BrainService, args ...string) (string, error) {
	root := cli.NewRootCmd(orch, brain)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	root.SilenceErrors = true
	root.SilenceUsage = true
	err := root.Execute()
	return out.String(), err
}

func TestDraftCmd(t *testing.T) {
	var got domain.Task
	orch := &scriptedOrch{createDraft: func(task domain.Task) (string, error) {
		got = task
		return "draft-1", nil
	}}
	out, err := execute(orch, nil, "draft", "--project", "/p", "--instruction", "do it",
		"--target", "a.go", "--provider", "ollama", "--model", "m", "--priority", "1", "--tags", "a, b")
	if err != nil || !strings.Contains(out, "Draft created: draft-1") {
		t.Fatalf("out=%q err=%v", out, err)
	}
	if got.ProjectPath != "/p" || got.Instruction != "do it" || got.TargetFile != "a.go" ||
		got.ProviderName != "ollama" || got.ModelID != "m" || got.Priority != 1 ||
		len(got.Tags) != 2 || got.Tags[0] != "a" || got.Tags[1] != "b" {
		t.Errorf("task passed to CreateDraft = %+v", got)
	}

	orch.createDraft = func(domain.Task) (string, error) { return "", errors.New("queue full") }
	if _, err := execute(orch, nil, "draft", "--project", "/p", "--instruction", "x"); err == nil || !strings.Contains(err.Error(), "queue full") {
		t.Errorf("error not surfaced: %v", err)
	}
	if _, err := execute(orch, nil, "draft", "--project", "/p"); err == nil {
		t.Error("missing --instruction must be rejected")
	}
}

func TestBacklogCmd(t *testing.T) {
	long := strings.Repeat("x", 80)
	orch := &scriptedOrch{getBacklog: func(string) ([]domain.Task, error) {
		return []domain.Task{
			{ID: "b-1", Priority: 2, Status: domain.StatusBacklog, ProviderName: "ollama", Instruction: long},
		}, nil
	}}
	out, err := execute(orch, nil, "backlog", "--project", "/p")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ID", "Priority", "b-1", "ollama", strings.Repeat("x", 60)} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, strings.Repeat("x", 61)) {
		t.Error("instruction must be truncated to 60 characters")
	}

	orch.getBacklog = func(string) ([]domain.Task, error) { return nil, nil }
	if out, _ := execute(orch, nil, "backlog", "--project", "/p"); !strings.Contains(out, "No backlog items") {
		t.Errorf("empty backlog output = %q", out)
	}

	orch.getBacklog = func(string) ([]domain.Task, error) { return nil, errors.New("db down") }
	if _, err := execute(orch, nil, "backlog", "--project", "/p"); err == nil || !strings.Contains(err.Error(), "db down") {
		t.Errorf("error not surfaced: %v", err)
	}
}

func TestPromoteCmd(t *testing.T) {
	orch := &scriptedOrch{}

	orch.promoteTask = func(string) (ports.PromoteResult, error) {
		return ports.PromoteResult{Promoted: true, Warning: "queue nearly full"}, nil
	}
	out, err := execute(orch, nil, "promote", "t-1")
	if err != nil || !strings.Contains(out, "Task t-1 promoted to queue") || !strings.Contains(out, "Warning: queue nearly full") {
		t.Fatalf("out=%q err=%v", out, err)
	}

	orch.promoteTask = func(string) (ports.PromoteResult, error) {
		return ports.PromoteResult{Promoted: true}, nil
	}
	if out, _ := execute(orch, nil, "promote", "t-1"); strings.Contains(out, "Warning") {
		t.Errorf("no warning expected, got %q", out)
	}

	// A missing task must be a failure (non-zero exit), whether the error wraps
	// domain.ErrNotFound or only carries "not found" text (HTTP client).
	for name, perr := range map[string]error{
		"sentinel": fmt.Errorf("promote: %w", domain.ErrNotFound),
		"text":     errors.New("daemon: task not found"),
	} {
		orch.promoteTask = func(string) (ports.PromoteResult, error) { return ports.PromoteResult{}, perr }
		if _, err := execute(orch, nil, "promote", "ghost"); err == nil || !strings.Contains(err.Error(), "task ghost not found") {
			t.Errorf("%s: want a not-found error, got %v", name, err)
		}
	}

	orch.promoteTask = func(string) (ports.PromoteResult, error) { return ports.PromoteResult{}, errors.New("boom") }
	if _, err := execute(orch, nil, "promote", "t-1"); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("generic error not surfaced: %v", err)
	}
	if _, err := execute(orch, nil, "promote"); err == nil {
		t.Error("promote without an id must be rejected")
	}
}

func TestUpdateCmd(t *testing.T) {
	var gotID string
	var got domain.Task
	orch := &scriptedOrch{updateTask: func(id string, u domain.Task) (domain.Task, error) {
		gotID, got = id, u
		return domain.Task{ID: id, Status: domain.StatusQueued, Priority: u.Priority}, nil
	}}

	out, err := execute(orch, nil, "update", "t-9", "--instruction", "new", "--provider", "p",
		"--model", "m", "--priority", "3", "--tags", "x, y")
	if err != nil || !strings.Contains(out, "Task t-9 updated: status=QUEUED, priority=3") {
		t.Fatalf("out=%q err=%v", out, err)
	}
	if gotID != "t-9" || got.Instruction != "new" || got.ProviderName != "p" || got.ModelID != "m" ||
		got.Priority != 3 || len(got.Tags) != 2 || got.Tags[1] != "y" {
		t.Errorf("update = %q %+v", gotID, got)
	}

	// Only flags that were set are sent.
	if _, err := execute(orch, nil, "update", "t-9", "--priority", "1"); err != nil {
		t.Fatal(err)
	}
	if got.Instruction != "" || got.ProviderName != "" || got.ModelID != "" || got.Tags != nil || got.Priority != 1 {
		t.Errorf("unset flags leaked into update: %+v", got)
	}

	orch.updateTask = func(string, domain.Task) (domain.Task, error) { return domain.Task{}, errors.New("nope") }
	if _, err := execute(orch, nil, "update", "t-9"); err == nil || !strings.Contains(err.Error(), "nope") {
		t.Errorf("error not surfaced: %v", err)
	}
}

func TestBrainCmds_OutputAndErrors(t *testing.T) {
	brain := &mockBrainService{
		getStatusResult:  domain.BrainStatus{ProjectPath: "/p", Initialized: true, EntryCount: 2},
		listResult:       []domain.ProjectKnowledge{{ID: "k1", Topic: "layers"}},
		getContextResult: domain.ContextResponse{TokensUsed: 7},
		getFileMapResult: []string{"a.go", "b.go"},
	}
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"status", []string{"brain", "status", "--project", "/p"}, `"entryCount": 2`},
		{"list", []string{"brain", "list", "--project", "/p", "--kind", "learning"}, `"topic": "layers"`},
		{"context", []string{"brain", "context", "--project", "/p", "--max-tokens", "50"}, `"tokensUsed": 7`},
		{"delete", []string{"brain", "delete", "--id", "k1"}, "deleted k1"},
		{"file-map", []string{"brain", "file-map", "--project", "/p"}, "a.go\nb.go"},
	}
	for _, tc := range cases {
		out, err := execute(&mockOrchestrator{}, brain, tc.args...)
		if err != nil || !strings.Contains(out, tc.want) {
			t.Errorf("%s: out=%q err=%v, want %q", tc.name, out, err, tc.want)
		}
	}

	if out, _ := execute(&mockOrchestrator{}, &mockBrainService{}, "brain", "file-map", "--project", "/p"); !strings.Contains(out, "no file map entries") {
		t.Errorf("empty file map output = %q", out)
	}

	boom := errors.New("kaboom")
	failing := &mockBrainService{listErr: boom, deleteErr: boom, getContextErr: boom, getFileMapErr: boom}
	for _, args := range [][]string{
		{"brain", "list", "--project", "/p"},
		{"brain", "delete", "--id", "k1"},
		{"brain", "context", "--project", "/p"},
		{"brain", "file-map", "--project", "/p"},
	} {
		if _, err := execute(&mockOrchestrator{}, failing, args...); err == nil || !strings.Contains(err.Error(), "kaboom") {
			t.Errorf("%v: error not surfaced: %v", args, err)
		}
	}
}
