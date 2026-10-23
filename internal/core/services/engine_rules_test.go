package services_test

// The engine applies the project's rules to every task: its default verification
// command, a briefing plus the rules in the prompt, enforcement of the structured
// rules on agent writes, and a short record of each finished task.

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

type recordingRunner struct {
	mu   sync.Mutex
	cmds []string
}

func (r *recordingRunner) Run(_ context.Context, _, cmd string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cmds = append(r.cmds, cmd)
	return "ok", nil
}

// ruleEngine is newEngine plus a real brain over an in-memory knowledge store.
func ruleEngine(t *testing.T, w ports.FileWriter, sess ports.SessionRepository, runner ports.CommandRunner, llm ports.LLMClient) (*engineFixture, *mockKnowledgeRepo) {
	t.Helper()
	know := &mockKnowledgeRepo{}
	e := newEngine(t, w, sess, runner, llm)
	e.orch.WithBrain(services.NewBrainService(know, nil))
	return e, know
}

func projectWith(t *testing.T, policy string) string {
	t.Helper()
	root := t.TempDir()
	if policy != "" {
		writePolicy(t, root, policy)
	}
	return root
}

func TestRules_PolicyVerifyIsTheDefaultVerificationCommand(t *testing.T) {
	root := projectWith(t, `{"verify":"make check"}`)
	runner := &recordingRunner{}
	e, _ := ruleEngine(t, &noopWriter{}, nil, runner, &scriptLLM{name: "p", alive: true})
	id := e.queue(domain.Task{Instruction: "defaults", ProjectPath: root, TargetFile: "a.go"})
	got := e.runOne(id, domain.StatusCompleted)

	if len(runner.cmds) != 1 || runner.cmds[0] != "make check" {
		t.Errorf("verification ran %v, want [make check]", runner.cmds)
	}
	if got.VerificationCommand != "make check" {
		t.Errorf("the task should record the command it was verified with, got %q", got.VerificationCommand)
	}
}

func TestRules_ATasksOwnVerificationCommandWins(t *testing.T) {
	root := projectWith(t, `{"verify":"make check"}`)
	runner := &recordingRunner{}
	e, _ := ruleEngine(t, &noopWriter{}, nil, runner, &scriptLLM{name: "p", alive: true})
	id := e.queue(domain.Task{Instruction: "own", ProjectPath: root, TargetFile: "a.go", VerificationCommand: "go test ./x"})
	e.runOne(id, domain.StatusCompleted)
	if len(runner.cmds) != 1 || runner.cmds[0] != "go test ./x" {
		t.Errorf("verification ran %v, want the task's own command", runner.cmds)
	}
}

func TestRules_PromptCarriesBriefingAndRulesAheadOfTheTask(t *testing.T) {
	root := projectWith(t, `{"testsFirst":true,"rules":["Keep it small."]}`)
	llm := &scriptLLM{name: "p", alive: true, limit: 100000}
	e, know := ruleEngine(t, &noopWriter{}, nil, nil, llm)
	know.entries = append(know.entries, domain.ProjectKnowledge{
		ProjectPath: root, Kind: domain.KnowledgeArchitecture, Topic: "Layers", Content: "Hexagonal: core never imports adapters.",
	})
	id := e.queue(domain.Task{Instruction: "do the thing", ProjectPath: root, TargetFile: "a.go"})
	e.runOne(id, domain.StatusCompleted)

	p := llm.prompts[0]
	order := []string{"Project Onboarding", "Hexagonal: core never imports adapters.", "## Workflow rules", "Keep it small.", "## Task", "do the thing"}
	last := -1
	for _, want := range order {
		i := strings.Index(p, want)
		if i < 0 || i < last {
			t.Fatalf("prompt must contain %q after the previous part:\n%s", want, p)
		}
		last = i
	}
	if strings.Count(p, "## Workflow rules") != 1 {
		t.Errorf("the rules must appear exactly once (tokens are money):\n%s", p)
	}
}

func TestRules_SmallContextModelsGetRulesButNoBriefing(t *testing.T) {
	root := projectWith(t, `{"rules":["Keep it small."]}`)
	llm := &scriptLLM{name: "tiny", alive: true, limit: 2000}
	e, _ := ruleEngine(t, &noopWriter{}, nil, nil, llm)
	id := e.queue(domain.Task{Instruction: "small model", ProjectPath: root, TargetFile: "a.go"})
	e.runOne(id, domain.StatusCompleted)
	p := llm.prompts[0]
	if strings.Contains(p, "Project Onboarding") || !strings.Contains(p, "Keep it small.") {
		t.Errorf("a 2000-token model must get the rules only:\n%s", p)
	}
}

func TestRules_NoPolicyNoBrainLeavesThePromptAlone(t *testing.T) {
	llm := &scriptLLM{name: "p", alive: true}
	e := newEngine(t, &noopWriter{}, nil, nil, llm)
	id := e.queue(domain.Task{Instruction: "untouched", ProjectPath: t.TempDir(), TargetFile: "a.go"})
	e.runOne(id, domain.StatusCompleted)
	if llm.prompts[0] != "untouched" {
		t.Errorf("prompt = %q, want the bare instruction", llm.prompts[0])
	}
}

func TestRules_ABrokenPolicyFailsTheTaskLoudly(t *testing.T) {
	root := projectWith(t, `{"maxFileLine": 5}`)
	llm := &scriptLLM{name: "p", alive: true}
	e, _ := ruleEngine(t, &noopWriter{}, nil, nil, llm)
	id := e.queue(domain.Task{Instruction: "broken", ProjectPath: root, TargetFile: "a.go"})
	got := e.runOne(id, domain.StatusFailed)
	if !strings.Contains(got.Logs, "policy") || llm.calls != 0 {
		t.Errorf("logs=%q modelCalls=%d: must fail with a policy message before any model call", got.Logs, llm.calls)
	}
}

func TestRules_AgentMustWriteATestBeforeSource(t *testing.T) {
	root := projectWith(t, `{"testsFirst":true}`)
	fs := newMemFS(map[string]string{})
	llm := &scriptLLM{name: "p", alive: true, limit: 100000, replies: []string{
		tool(`{"tool":"write_file","path":"a.go","content":"package a"}`),
		tool(`{"tool":"write_file","path":"a_test.go","content":"package a"}`) + tool(`{"tool":"write_file","path":"a.go","content":"package a"}`),
		tool(`{"tool":"finish","summary":"done"}`),
	}}
	e, _ := ruleEngine(t, fs, nil, nil, llm)
	id := e.queue(agentTask(func(t *domain.Task) { t.ProjectPath = root }))
	e.runOne(id, domain.StatusCompleted)

	if want := []string{"a_test.go", "a.go"}; strings.Join(fs.order, ",") != strings.Join(want, ",") {
		t.Errorf("write order = %v, want the test first and the refused write not recorded: %v", fs.order, want)
	}
	h := llm.histories[1]
	if !strings.Contains(h[len(h)-1].Content, "tests first") {
		t.Errorf("the model was not told why its write was refused: %q", h[len(h)-1].Content)
	}
}

func TestRules_AgentEditOfSourceFollowsTheSameOrder(t *testing.T) {
	root := projectWith(t, `{"testsFirst":true}`)
	fs := newMemFS(map[string]string{"a.go": "package a // old"})
	llm := &scriptLLM{name: "p", alive: true, limit: 100000, replies: []string{
		tool(`{"tool":"edit_file","path":"a.go","old":"old","new":"new"}`),
		tool(`{"tool":"finish","summary":"gave up"}`),
	}}
	e, _ := ruleEngine(t, fs, nil, nil, llm)
	id := e.queue(agentTask(func(t *domain.Task) { t.ProjectPath = root }))
	e.runOne(id, domain.StatusCompleted)
	if fs.files["a.go"] != "package a // old" {
		t.Errorf("a refused edit must leave the file alone, got %q", fs.files["a.go"])
	}
}

func TestRules_AgentCannotExceedTheFileSizeLimit(t *testing.T) {
	root := projectWith(t, `{"maxFileLines":3}`)
	fs := newMemFS(map[string]string{})
	big := strings.Repeat("x\\n", 10)
	llm := &scriptLLM{name: "p", alive: true, limit: 100000, replies: []string{
		tool(`{"tool":"write_file","path":"big.go","content":"` + big + `"}`),
		tool(`{"tool":"write_file","path":"a.go","content":"1\n2\n"}`) + tool(`{"tool":"finish","summary":"split"}`),
	}}
	e, _ := ruleEngine(t, fs, nil, nil, llm)
	id := e.queue(agentTask(func(t *domain.Task) { t.ProjectPath = root }))
	e.runOne(id, domain.StatusCompleted)

	if _, wrote := fs.files["big.go"]; wrote {
		t.Error("an over-long file must not be written")
	}
	h := llm.histories[1]
	if !strings.Contains(h[len(h)-1].Content, "the limit is 3") {
		t.Errorf("the model must be told the limit: %q", h[len(h)-1].Content)
	}
}

func TestRules_ADeclinedWriteDoesNotCountAsAWrittenFile(t *testing.T) {
	root := projectWith(t, `{"testsFirst":true}`)
	llm := &scriptLLM{name: "p", alive: true, limit: 100000, replies: []string{
		tool(`{"tool":"write_file","path":"a.go","content":"x"}`) + tool(`{"tool":"finish","summary":"done"}`),
	}}
	e, _ := ruleEngine(t, newMemFS(map[string]string{}), nil, nil, llm)
	id := e.queue(agentTask(func(t *domain.Task) { t.ProjectPath = root }))
	got := e.runOne(id, domain.StatusCompleted)
	if !strings.Contains(got.Logs, "0 file(s) written") {
		t.Errorf("logs = %q", got.Logs)
	}
}

func TestRules_FinishedTasksAreRememberedAndPruned(t *testing.T) {
	root := projectWith(t, `{"remember":true}`)
	llm := &scriptLLM{name: "p", alive: true}
	e, know := ruleEngine(t, &noopWriter{}, nil, nil, llm)
	for i := 0; i < services.MaxTaskMemories+5; i++ {
		id := e.queue(domain.Task{Instruction: fmt.Sprintf("job %d", i), ProjectPath: root, TargetFile: "a.go"})
		e.runOne(id, domain.StatusCompleted)
	}
	var mine []domain.ProjectKnowledge
	for _, k := range know.entries {
		if k.Source == services.TaskMemorySource {
			mine = append(mine, k)
		}
	}
	if len(mine) != services.MaxTaskMemories {
		t.Fatalf("kept %d task memories, want exactly %d", len(mine), services.MaxTaskMemories)
	}
	newest := mine[len(mine)-1]
	if newest.Kind != domain.KnowledgeLearning || !strings.Contains(newest.Content, "job 34") || !strings.Contains(newest.Content, "Completed") {
		t.Errorf("newest memory = %+v", newest)
	}
	for _, k := range mine {
		if strings.Contains(k.Content, "job 0 ") || strings.HasSuffix(k.Content, "job 0") {
			t.Errorf("the oldest memory should have been pruned: %q", k.Content)
		}
	}
}

func TestRules_FailureIsRememberedToo(t *testing.T) {
	root := projectWith(t, `{"remember":true}`)
	llm := &scriptLLM{name: "p", alive: true, errs: []error{fmt.Errorf("boom"), fmt.Errorf("boom"), fmt.Errorf("boom")}}
	e, know := ruleEngine(t, &noopWriter{}, nil, nil, llm)
	id := e.queue(domain.Task{Instruction: "doomed", ProjectPath: root, TargetFile: "a.go"})
	for e.status(id).Status != domain.StatusFailed {
		e.orch.ProcessNext()
	}
	var found bool
	for _, k := range know.entries {
		found = found || (k.Source == services.TaskMemorySource && strings.Contains(k.Content, "Failed"))
	}
	if !found {
		t.Errorf("a failed task must leave a memory so the next session avoids repeating it: %+v", know.entries)
	}
}

func TestRules_NothingIsRememberedUnlessThePolicyAsks(t *testing.T) {
	for name, policy := range map[string]string{"no policy": "", "remember off": `{"verify":"x"}`} {
		t.Run(name, func(t *testing.T) {
			root := projectWith(t, policy)
			e, know := ruleEngine(t, &noopWriter{}, nil, nil, &scriptLLM{name: "p", alive: true})
			id := e.queue(domain.Task{Instruction: "quiet", ProjectPath: root, TargetFile: "a.go"})
			if policy == "" {
				e.runOne(id, domain.StatusCompleted)
			} else {
				e.orch.ProcessNext()
			}
			if len(know.entries) != 0 {
				t.Errorf("unexpected memory: %+v", know.entries)
			}
		})
	}
}

func TestRules_LongSessionHistoryIsCappedToTheNewestMessages(t *testing.T) {
	root := projectWith(t, "")
	sess := newMemSessionRepo()
	for i := 0; i < 60; i++ {
		role := domain.RoleUser
		if i%2 == 1 {
			role = domain.RoleAssistant
		}
		_ = sess.AppendMessage(context.Background(), root, domain.Message{Role: role, Content: fmt.Sprintf("m%02d", i)})
	}
	llm := &scriptLLM{name: "p", alive: true}
	e := newEngine(t, &noopWriter{}, sess, nil, llm)
	id := e.queue(domain.Task{Instruction: "capped", ProjectPath: root, TargetFile: "a.go"})
	e.runOne(id, domain.StatusCompleted)

	h := llm.histories[0]
	if len(h) != services.MaxHistoryMessages+1 {
		t.Fatalf("model saw %d messages, want %d history + the new prompt", len(h), services.MaxHistoryMessages)
	}
	if h[0].Role != domain.RoleUser {
		t.Errorf("a trimmed history must still start with a user message, got %s", h[0].Role)
	}
	if h[len(h)-2].Content != "m59" {
		t.Errorf("the newest history message must be kept, got %q", h[len(h)-2].Content)
	}
}

// plainBrain hides the optional GetProjectBrief and lets a test break single operations.
type plainBrain struct {
	ports.BrainService
	onboardErr, ingestErr, listErr, deleteErr error
	onboardCalls                              int
	deleted                                   int
}

func (b *plainBrain) GetOnboardingContext(ctx context.Context, p string, n int) (string, error) {
	b.onboardCalls++
	if b.onboardErr != nil {
		return "", b.onboardErr
	}
	return b.BrainService.GetOnboardingContext(ctx, p, n)
}

func (b *plainBrain) IngestKnowledge(ctx context.Context, k domain.ProjectKnowledge) (domain.ProjectKnowledge, error) {
	if b.ingestErr != nil {
		return domain.ProjectKnowledge{}, b.ingestErr
	}
	return b.BrainService.IngestKnowledge(ctx, k)
}

func (b *plainBrain) ListKnowledge(ctx context.Context, p, kind string) ([]domain.ProjectKnowledge, error) {
	if b.listErr != nil {
		return nil, b.listErr
	}
	return b.BrainService.ListKnowledge(ctx, p, kind)
}

func (b *plainBrain) DeleteKnowledge(ctx context.Context, id string) error {
	b.deleted++
	if b.deleteErr != nil {
		return b.deleteErr
	}
	return b.BrainService.DeleteKnowledge(ctx, id)
}

func plainEngine(t *testing.T, b *plainBrain, llm ports.LLMClient) *engineFixture {
	t.Helper()
	b.BrainService = services.NewBrainService(&mockKnowledgeRepo{}, nil)
	e := newEngine(t, &noopWriter{}, nil, nil, llm)
	e.orch.WithBrain(b)
	return e
}

func TestRules_AnyBrainWithoutABriefMethodStillBriefsTheModel(t *testing.T) {
	root := projectWith(t, "")
	b := &plainBrain{}
	llm := &scriptLLM{name: "p", alive: true, limit: 100000}
	e := plainEngine(t, b, llm)
	e.runOne(e.queue(domain.Task{Instruction: "fallback", ProjectPath: root, TargetFile: "a.go"}), domain.StatusCompleted)
	if b.onboardCalls != 1 || !strings.Contains(llm.prompts[0], "Project Onboarding") {
		t.Errorf("onboarding calls=%d prompt=%q", b.onboardCalls, llm.prompts[0])
	}
}

func TestRules_ABriefingFailureNeverBlocksTheTask(t *testing.T) {
	root := projectWith(t, "")
	llm := &scriptLLM{name: "p", alive: true, limit: 100000}
	e := plainEngine(t, &plainBrain{onboardErr: fmt.Errorf("brain down")}, llm)
	e.runOne(e.queue(domain.Task{Instruction: "no brief", ProjectPath: root, TargetFile: "a.go"}), domain.StatusCompleted)
	if llm.prompts[0] != "no brief" {
		t.Errorf("prompt = %q, want the bare instruction", llm.prompts[0])
	}
}

func TestRules_MemoryFailuresNeverBlockTheTask(t *testing.T) {
	cases := map[string]*plainBrain{
		"ingest fails": {ingestErr: fmt.Errorf("disk full")},
		"list fails":   {listErr: fmt.Errorf("db locked")},
	}
	for name, b := range cases {
		t.Run(name, func(t *testing.T) {
			root := projectWith(t, `{"remember":true}`)
			e := plainEngine(t, b, &scriptLLM{name: "p", alive: true})
			e.runOne(e.queue(domain.Task{Instruction: "resilient", ProjectPath: root, TargetFile: "a.go"}), domain.StatusCompleted)
		})
	}
}

func TestRules_PruneToleratesDeleteErrors(t *testing.T) {
	root := projectWith(t, `{"remember":true}`)
	b := &plainBrain{deleteErr: fmt.Errorf("cannot delete")}
	e := plainEngine(t, b, &scriptLLM{name: "p", alive: true})
	for i := 0; i < services.MaxTaskMemories+2; i++ {
		e.runOne(e.queue(domain.Task{Instruction: fmt.Sprintf("n%d", i), ProjectPath: root, TargetFile: "a.go"}), domain.StatusCompleted)
	}
	if b.deleted == 0 {
		t.Error("pruning should have tried to delete the oldest entries")
	}
}

func TestRules_ProjectBriefIsByteStableAcrossTasksAndInjectsFocusedKnowledge(t *testing.T) {
	root := projectWith(t, `{"rules":["Follow hexagonal architecture."]}`)
	know := &mockKnowledgeRepo{
		entries: []domain.ProjectKnowledge{
			{
				ProjectPath:    root,
				Kind:           domain.KnowledgeLearning,
				Topic:          "Authentication",
				Content:        "Always validate Bearer tokens in middleware using subtle.ConstantTimeCompare.",
				RelevanceScore: 0.9,
			},
		},
	}
	llm := &scriptLLM{name: "p", alive: true, limit: 100000}
	e := newEngine(t, &noopWriter{}, nil, nil, llm)
	brain := services.NewBrainService(know, e.repo)
	e.orch.WithBrain(brain)

	briefBefore, err := brain.GetProjectBrief(context.Background(), root, 800)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(briefBefore, "- **Tasks:**") {
		t.Errorf("GetProjectBrief must omit volatile task counters for prompt cache stability:\n%s", briefBefore)
	}
	onboard, err := brain.GetOnboardingContext(context.Background(), root, 800)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(onboard, "- **Tasks:**") {
		t.Errorf("GetOnboardingContext must still include live task counters:\n%s", onboard)
	}

	id := e.queue(domain.Task{Instruction: "Update Authentication middleware token check", ProjectPath: root, TargetFile: "auth.go"})
	e.runOne(id, domain.StatusCompleted)

	briefAfter, err := brain.GetProjectBrief(context.Background(), root, 800)
	if err != nil {
		t.Fatal(err)
	}
	if briefBefore != briefAfter {
		t.Errorf("GetProjectBrief changed after task completion:\nbefore:\n%s\nafter:\n%s", briefBefore, briefAfter)
	}
	if !strings.Contains(llm.prompts[0], "subtle.ConstantTimeCompare") {
		t.Errorf("prompt must include focused knowledge matching task instruction:\n%s", llm.prompts[0])
	}
}
