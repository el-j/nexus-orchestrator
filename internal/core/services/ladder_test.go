package services_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

// ladderProject returns a project directory whose policy has the given body.
func ladderProject(t *testing.T, policy string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".nexus"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, services.PolicyFile), []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

const threeRungs = `{"ladder":{"coder":["Ollama/*","Anthropic/claude-sonnet*","Anthropic/claude-opus*"]}}`

func ladderProviders() (opus, sonnet, local *scriptLLM) {
	opus = &scriptLLM{name: "Anthropic", model: "claude-opus-5-5", alive: true, replies: []string{"from opus"}}
	sonnet = &scriptLLM{name: "Anthropic", model: "claude-sonnet-5-5", alive: true, replies: []string{"from sonnet"}}
	local = &scriptLLM{name: "Ollama", model: "qwen", alive: true, replies: []string{"from local"}}
	return
}

func TestLadder_ATaskStartsOnTheCheapestRung(t *testing.T) {
	opus, sonnet, local := ladderProviders()
	e := newEngine(t, &noopWriter{}, nil, nil, opus, sonnet, local) // registered most expensive first
	dir := ladderProject(t, threeRungs)
	e.runOne(e.queue(domain.Task{Instruction: "cheap", Role: "coder", ProjectPath: dir}), domain.StatusCompleted)
	if local.calls != 1 || sonnet.calls != 0 || opus.calls != 0 {
		t.Errorf("the local model is the first rung: local=%d sonnet=%d opus=%d", local.calls, sonnet.calls, opus.calls)
	}
}

func TestLadder_RolesWithoutALadderKeepTheNormalOrder(t *testing.T) {
	opus, sonnet, local := ladderProviders()
	e := newEngine(t, &noopWriter{}, nil, nil, opus, sonnet, local)
	dir := ladderProject(t, threeRungs)
	e.runOne(e.queue(domain.Task{Instruction: "other role", Role: "tester", ProjectPath: dir}), domain.StatusCompleted)
	if opus.calls+sonnet.calls+local.calls != 1 {
		t.Errorf("one call in total: %d %d %d", opus.calls, sonnet.calls, local.calls)
	}
	// With a "default" ladder the same role is routed.
	dir2 := ladderProject(t, `{"ladder":{"default":["Ollama/*"]}}`)
	o2, s2, l2 := ladderProviders()
	e2 := newEngine(t, &noopWriter{}, nil, nil, o2, s2, l2)
	e2.runOne(e2.queue(domain.Task{Instruction: "default ladder", Role: "tester", ProjectPath: dir2}), domain.StatusCompleted)
	if l2.calls != 1 {
		t.Errorf("the default ladder applies: local=%d", l2.calls)
	}
}

func TestLadder_ProvidersOutsideTheLadderComeAfterItsRungs(t *testing.T) {
	opus, sonnet, local := ladderProviders()
	outsider := &scriptLLM{name: "Gemini", model: "gemini-x", alive: true, replies: []string{"from gemini"}}
	// Only the sonnet rung is listed; the others follow in their normal order.
	e := newEngine(t, &noopWriter{}, nil, nil, outsider, opus, local, sonnet)
	dir := ladderProject(t, `{"ladder":{"coder":["Anthropic/claude-sonnet*"]}}`)
	e.runOne(e.queue(domain.Task{Instruction: "listed first", Role: "coder", ProjectPath: dir}), domain.StatusCompleted)
	if sonnet.calls != 1 || outsider.calls+opus.calls+local.calls != 0 {
		t.Errorf("sonnet=%d others=%d", sonnet.calls, outsider.calls+opus.calls+local.calls)
	}
}

func TestLadder_ACooledDownRungMovesTheTaskUpTheLadder(t *testing.T) {
	opus, sonnet, local := ladderProviders()
	local.errs = []error{limit("Ollama", time.Now().Add(time.Hour))}
	e := newEngine(t, &noopWriter{}, nil, nil, opus, sonnet, local)
	dir := ladderProject(t, threeRungs)
	e.runOne(e.queue(domain.Task{Instruction: "first", Role: "coder", ProjectPath: dir}), domain.StatusCompleted)
	if sonnet.calls != 1 || opus.calls != 0 {
		t.Errorf("the limited first rung hands over to the second: sonnet=%d opus=%d", sonnet.calls, opus.calls)
	}
	e.runOne(e.queue(domain.Task{Instruction: "second", Role: "coder", ProjectPath: dir}), domain.StatusCompleted)
	if local.calls != 1 || sonnet.calls != 2 {
		t.Errorf("the limited rung is skipped until its reset: local=%d sonnet=%d", local.calls, sonnet.calls)
	}
}

func TestLadder_APinnedProviderIsNotReordered(t *testing.T) {
	opus, sonnet, local := ladderProviders()
	e := newEngine(t, &noopWriter{}, nil, nil, opus, sonnet, local)
	dir := ladderProject(t, threeRungs)
	e.runOne(e.queue(domain.Task{Instruction: "pinned", Role: "coder", ProjectPath: dir, ProviderName: "Anthropic"}), domain.StatusCompleted)
	if local.calls != 0 {
		t.Errorf("a task that names its provider keeps it: local=%d", local.calls)
	}
}

func TestLadder_AFailedVerificationAsksTheNextRungUp(t *testing.T) {
	opus, sonnet, local := ladderProviders()
	local.replies = []string{"local broken"}
	sonnet.replies = []string{"sonnet still broken"}
	opus.replies = []string{"opus fixed"}
	runner := &scriptedRunner{results: []runResult{
		{"error A", errors.New("exit 1")}, {"error B", errors.New("exit 1")}, {"PASS", nil},
	}}
	e := newEngine(t, &contextWriter{}, nil, runner, opus, sonnet, local)
	dir := ladderProject(t, threeRungs)
	id := e.queue(domain.Task{Instruction: "implement Foo", Role: "coder", ProjectPath: dir, TargetFile: "foo.go",
		VerificationCommand: "go test", MaxCorrectionTurns: 3})
	got := e.runOne(id, domain.StatusCompleted)
	if local.calls != 1 || sonnet.calls != 1 || opus.calls != 1 {
		t.Fatalf("each failed verification goes one rung up: local=%d sonnet=%d opus=%d", local.calls, sonnet.calls, opus.calls)
	}
	if !strings.Contains(sonnet.prompts[0], "error A") || !strings.Contains(opus.prompts[0], "error B") {
		t.Errorf("each rung gets the failing output:\nsonnet: %s\nopus: %s", sonnet.prompts[0], opus.prompts[0])
	}
	for _, want := range []string{"asking Anthropic/claude-sonnet-5-5", "asking Anthropic/claude-opus-5-5"} {
		if !strings.Contains(got.Logs, want) {
			t.Errorf("logs lack %q:\n%s", want, got.Logs)
		}
	}
}

func TestLadder_TheTopRungKeepsCorrectingItself(t *testing.T) {
	opus, sonnet, local := ladderProviders()
	runner := &scriptedRunner{results: []runResult{{"e1", errors.New("x")}, {"e2", errors.New("x")}, {"e3", errors.New("x")}, {"PASS", nil}}}
	e := newEngine(t, &contextWriter{}, nil, runner, opus, sonnet, local)
	dir := ladderProject(t, `{"ladder":{"coder":["Ollama/*","Anthropic/claude-sonnet*"]}}`)
	id := e.queue(domain.Task{Instruction: "Foo", Role: "coder", ProjectPath: dir, TargetFile: "foo.go",
		VerificationCommand: "go test", MaxCorrectionTurns: 3})
	e.runOne(id, domain.StatusCompleted)
	if local.calls != 1 || sonnet.calls != 3 || opus.calls != 0 {
		t.Errorf("past the last listed rung the same model corrects again: local=%d sonnet=%d opus=%d", local.calls, sonnet.calls, opus.calls)
	}
}

func TestLadder_NoLadderMeansTheSameModelCorrectsItself(t *testing.T) {
	llm := &scriptLLM{name: "p", alive: true, replies: []string{"v1", "v2"}}
	runner := &scriptedRunner{results: []runResult{{"e", errors.New("x")}, {"PASS", nil}}}
	e := newEngine(t, &contextWriter{}, nil, runner, llm)
	e.runOne(e.queue(domain.Task{Instruction: "Foo", TargetFile: "foo.go", VerificationCommand: "go test"}), domain.StatusCompleted)
	if llm.calls != 2 {
		t.Errorf("calls = %d", llm.calls)
	}
}

func TestPreviewRouting_ShowsTheOrderTheLadderCooldownsAndQuotaGive(t *testing.T) {
	opus, sonnet, local := ladderProviders()
	hot := &quotaLLM{scriptLLM: &scriptLLM{name: "Gemini", model: "gemini-x", alive: true, replies: []string{"h"}}}
	e := newEngine(t, &noopWriter{}, nil, nil, opus, sonnet, local, hot)
	dir := ladderProject(t, `{"quotaThreshold":0.8,"ladder":{"coder":["Ollama/*","Gemini/*","Anthropic/claude-sonnet*"]}}`)
	local.errs = []error{limit("Ollama", time.Now().Add(time.Hour))}
	e.runOne(e.queue(domain.Task{Instruction: "limit the local one", Role: "coder", ProjectPath: dir}), domain.StatusCompleted)
	warm(e, hot, reading(0.93, time.Now().Add(time.Hour)), "hot")

	p, err := e.orch.PreviewRouting(t.Context(), dir, "coder")
	if err != nil {
		t.Fatal(err)
	}
	if p.QuotaThreshold != 0.8 || len(p.Ladder) != 3 || p.Role != "coder" || len(p.Order) != 4 {
		t.Fatalf("preview: %+v", p)
	}
	// Rung order is local, gemini, sonnet, then the one outside the ladder; the hot gemini goes behind the
	// others, and the cooling local one is still listed (with its cooldown) in its rung place.
	got := []string{}
	for _, en := range p.Order {
		got = append(got, en.Provider+"/"+en.Model)
	}
	want := []string{"Ollama/qwen", "Anthropic/claude-sonnet-5-5", "Anthropic/claude-opus-5-5", "Gemini/gemini-x"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
	if p.Order[0].CoolingUntil.IsZero() || p.Order[0].Rung != 0 || p.Order[1].Rung != 2 || p.Order[2].Rung != -1 {
		t.Errorf("entries: %+v", p.Order)
	}
	if hotEntry := p.Order[3]; hotEntry.Utilization == nil || *hotEntry.Utilization != 0.93 || hotEntry.Position != 4 {
		t.Errorf("the hot provider shows its reading: %+v", hotEntry)
	}
	if p.Order[1].Utilization != nil {
		t.Error("a provider that reports no quota shows none")
	}
}

func TestPreviewRouting_WithoutProvidersOrAPolicy(t *testing.T) {
	e := newEngine(t, &noopWriter{}, nil, nil, &scriptLLM{name: "P", alive: true, model: "m"})
	p, err := e.orch.PreviewRouting(t.Context(), t.TempDir(), "")
	if err != nil || len(p.Order) != 1 || p.Ladder != nil || p.QuotaThreshold != domain.DefaultQuotaThreshold {
		t.Errorf("%+v %v", p, err)
	}
	// The ladder is shown also when only one provider is alive.
	dir := ladderProject(t, `{"ladder":{"coder":["P/*"]}}`)
	if p, err := e.orch.PreviewRouting(t.Context(), dir, "coder"); err != nil || len(p.Ladder) != 1 || p.Order[0].Rung != 0 {
		t.Errorf("one provider: %+v %v", p, err)
	}
	none := newEngine(t, &noopWriter{}, nil, nil, &scriptLLM{name: "Down", alive: false})
	if _, err := none.orch.PreviewRouting(t.Context(), t.TempDir(), "coder"); err == nil {
		t.Error("no live provider is an error")
	}
}
