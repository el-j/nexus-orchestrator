package services_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

func TestSelfHealing_ChatPathUsesHistoryAndPersistsEveryExchange(t *testing.T) {
	llm := &scriptLLM{name: "p", alive: true, replies: []string{"v1", "v2 fixed"}}
	sess := newMemSessionRepo()
	_ = sess.Save(domain.Session{ProjectPath: "/proj", Messages: []domain.Message{{Role: domain.RoleUser, Content: "earlier"}}})
	runner := &scriptedRunner{results: []runResult{{"fail", errors.New("exit 1")}, {"PASS", nil}}}
	e := newEngine(t, &contextWriter{}, sess, runner, llm)
	id := e.queue(domain.Task{Instruction: "chat-heal", TargetFile: "a.go", VerificationCommand: "t"})
	e.runOne(id, domain.StatusCompleted)

	if len(llm.histories) != 2 {
		t.Fatalf("want 2 chat calls, got %d", len(llm.histories))
	}
	if h := llm.histories[1]; h[0].Content != "earlier" || !strings.Contains(h[len(h)-1].Content, "chat-heal") {
		t.Errorf("correction chat must carry prior history and a full correction prompt: %+v", h)
	}
	// earlier + (user, assistant) for the task + (user, assistant) for the correction.
	if saved, _ := sess.GetByProjectPath("/proj"); len(saved.Messages) != 5 {
		t.Errorf("session has %d messages, want 5", len(saved.Messages))
	}
}

func TestSelfHealing_ProviderErrorOnCorrectionTurnWhenRetriesExhausted(t *testing.T) {
	for _, withSession := range []bool{false, true} {
		llm := &scriptLLM{name: "p", alive: true, replies: []string{"v1"}, errs: []error{nil, errInjected}}
		var sess *memSessionRepo
		var e *engineFixture
		runner := &scriptedRunner{results: []runResult{{"fail", errors.New("exit 1")}}}
		if withSession {
			sess = newMemSessionRepo()
			e = newEngine(t, &contextWriter{}, sess, runner, llm)
		} else {
			e = newEngine(t, &contextWriter{}, nil, runner, llm)
		}
		// RetryCount already at the cap (2): no more re-queues, so the failure is final.
		id := e.queue(domain.Task{Instruction: "exhausted", TargetFile: "a.go", VerificationCommand: "t", RetryCount: 2})
		got := e.runOne(id, domain.StatusFailed)
		if !strings.Contains(got.Logs, "failed via p") {
			t.Errorf("withSession=%v: logs = %q", withSession, got.Logs)
		}
	}
}

func TestProcessNext_StatusWriteFailuresAreTolerated(t *testing.T) {
	// NO_PROVIDER path with log/status writes failing.
	e := newEngine(t, &noopWriter{}, nil, nil)
	id := e.queue(domain.Task{Instruction: "np"})
	e.repo.set("UpdateLogs", true)
	e.repo.set("UpdateStatus", true)
	if !e.orch.ProcessNext() {
		t.Fatal("task should still be consumed")
	}
	if got := e.status(id); got.Status != domain.StatusProcessing {
		t.Errorf("status writes failed, so the claimed status remains: %s", got.Status)
	}

	// Context-file failure path with log/status writes failing.
	llm := &scriptLLM{name: "p", alive: true}
	e2 := newEngine(t, &contextWriter{ctxErr: errInjected}, nil, nil, llm)
	id2 := e2.queue(domain.Task{Instruction: "ctx", ContextFiles: []string{"x"}})
	e2.repo.set("UpdateLogs", true)
	e2.repo.set("UpdateStatus", true)
	e2.orch.ProcessNext()
	if llm.calls != 0 || e2.status(id2).Status != domain.StatusProcessing {
		t.Errorf("calls=%d status=%s", llm.calls, e2.status(id2).Status)
	}

	// Session-load failure path likewise.
	e3 := newEngine(t, &noopWriter{}, &failingSessionRepo{err: errInjected}, nil, llm)
	id3 := e3.queue(domain.Task{Instruction: "sess"})
	e3.repo.set("UpdateLogs", true)
	e3.repo.set("UpdateStatus", true)
	e3.orch.ProcessNext()
	if llm.calls != 0 || e3.status(id3).Status != domain.StatusProcessing {
		t.Errorf("calls=%d status=%s", llm.calls, e3.status(id3).Status)
	}
}

func TestFallbackChain_ExhaustionWhenWritesAndRetryUpdateFail(t *testing.T) {
	llm := &scriptLLM{name: "p", alive: true, errs: []error{errInjected, errInjected, errInjected}}
	e := newEngine(t, &noopWriter{}, nil, nil, llm)
	id := e.queue(domain.Task{Instruction: "retry-update-fails", RetryCount: 0})
	e.repo.set("Update", true) // requeueForRetry cannot persist -> falls through to FAILED
	e.repo.set("UpdateLogs", true)
	e.repo.set("UpdateStatus", true)
	e.orch.ProcessNext()
	if llm.calls != 1 {
		t.Errorf("calls = %d", llm.calls)
	}
	_ = id
}

func TestFallbackChain_TooLargeWithWriteFailuresStillEmitsEvent(t *testing.T) {
	small := &scriptLLM{name: "tiny", alive: true, limit: 600}
	e := newEngine(t, &noopWriter{}, nil, nil, small)
	e.queue(domain.Task{Instruction: strings.Repeat("word ", 400)})
	e.repo.set("UpdateLogs", true)
	e.repo.set("UpdateStatus", true)
	e.orch.ProcessNext()
	found := false
	for _, ev := range e.bcast.taskEvents() {
		if ev.Status == domain.StatusTooLarge {
			found = true
		}
	}
	if !found {
		t.Error("too-large event must be emitted even when persistence fails")
	}
}

func TestWriteAndVerify_PassFirstTimeAndWriteBeforeVerifyFailure(t *testing.T) {
	llm := &scriptLLM{name: "p", alive: true, replies: []string{"code"}}
	runner := &scriptedRunner{results: []runResult{{"PASS", nil}}}
	e := newEngine(t, &contextWriter{}, nil, runner, llm)
	id := e.queue(domain.Task{Instruction: "first-pass", TargetFile: "a.go", VerificationCommand: "t"})
	got := e.runOne(id, domain.StatusCompleted)
	if !strings.Contains(got.Logs, "verification passed (t)") || got.VerificationOutput != "PASS" {
		t.Errorf("logs=%q out=%q", got.Logs, got.VerificationOutput)
	}
}

func TestSubmitAndPromote_RepositoryAndAdmissionErrors(t *testing.T) {
	orch, repo, _ := newFaultyOrch(t)
	repo.set("Save", true)
	if _, err := orch.SubmitTask(domain.Task{Instruction: "x", ProjectPath: "/p"}); !errors.Is(err, errInjected) {
		t.Errorf("save failure: %v", err)
	}
	repo.set("Save", false)

	// Admission: GetPending failure.
	repo.set("GetPending", true)
	if _, err := orch.SubmitTask(domain.Task{Instruction: "x", ProjectPath: "/p"}); !errors.Is(err, errInjected) {
		t.Errorf("pending failure: %v", err)
	}
	repo.set("GetPending", false)

	// Admission: "execute" needs a completed plan; lookup failure surfaces.
	byProject := &projectFailRepo{faultyRepo: newFaultyRepo()}
	o2 := newOrchOver(t, byProject)
	if _, err := o2.SubmitTask(domain.Task{Instruction: "x", ProjectPath: "/p", Command: domain.CommandExecute}); !errors.Is(err, errInjected) {
		t.Errorf("project lookup failure: %v", err)
	}
}

type projectFailRepo struct{ *faultyRepo }

func (p *projectFailRepo) GetByProjectPath(string) ([]domain.Task, error) { return nil, errInjected }

func TestClaimTask_RefetchFailureAndAppendRoutedFailure(t *testing.T) {
	// Second GetByID (after the status flip) fails.
	inner := newFaultyRepo()
	seedTask(t, inner, "t", domain.StatusQueued)
	repo := &secondGetFailsRepo{faultyRepo: inner}
	orch := newOrchOver(t, repo)
	aiRepo := newMemAISessionRepo()
	orch.SetAISessionRepo(aiRepo)
	seedActiveSession(t, aiRepo, "s")
	if _, err := orch.ClaimTask(bg, "t", "s"); !errors.Is(err, errInjected) {
		t.Errorf("refetch failure: %v", err)
	}

	// AppendRoutedTaskID failing is logged but the claim still succeeds.
	orch2, repo2, ai2 := newFaultyOrch(t)
	if err := ai2.SaveAISession(bg, domain.AISession{ID: "s2", Status: domain.SessionStatusActive}); err != nil {
		t.Fatal(err)
	}
	orch2.SetAISessionRepo(&appendFailAIRepo{memAISessionRepo: ai2})
	seedTask(t, repo2, "t2", domain.StatusQueued)
	if got, err := orch2.ClaimTask(bg, "t2", "s2"); err != nil || got.Status != domain.StatusProcessing {
		t.Errorf("claim must succeed despite routed-id failure: %+v %v", got, err)
	}
}

type secondGetFailsRepo struct {
	*faultyRepo
	gets int
}

func (r *secondGetFailsRepo) GetByID(id string) (domain.Task, error) {
	r.gets++
	if r.gets >= 2 {
		return domain.Task{}, errInjected
	}
	return r.faultyRepo.GetByID(id)
}

type appendFailAIRepo struct{ *memAISessionRepo }

func (a *appendFailAIRepo) AppendRoutedTaskID(_ context.Context, _, _ string) error {
	return errInjected
}

func TestTriggerScanAndPromoteProvider_ErrorBranches(t *testing.T) {
	orch, _ := newProviderOrch(t, nil, nil)
	orch.WithSystemScanner(&mockScanner{err: errInjected})
	if _, err := orch.TriggerScan(bg); !errors.Is(err, errInjected) {
		t.Errorf("scan failure: %v", err)
	}

	// Promote with a config repo whose save fails.
	repo := newMemProviderConfigRepo()
	repo.saveErr = errInjected
	o2, _ := newProviderOrch(t, repo, namedFactory)
	o2.WithSystemScanner(&mockScanner{results: []domain.DiscoveredProvider{{ID: "d", Name: "n", BaseURL: "http://x", Status: domain.DiscoveryStatusReachable}}})
	if _, err := o2.TriggerScan(bg); err != nil {
		t.Fatal(err)
	}
	if err := o2.PromoteProvider(bg, "d"); !errors.Is(err, errInjected) {
		t.Errorf("persist failure: %v", err)
	}

	// Promote without a config repo whose factory fails.
	o3, _ := newProviderOrch(t, nil, func(domain.ProviderConfig) (ports.LLMClient, error) { return nil, errInjected })
	o3.WithSystemScanner(&mockScanner{results: []domain.DiscoveredProvider{{ID: "d", Name: "n", BaseURL: "http://x", Status: domain.DiscoveryStatusReachable}}})
	_, _ = o3.TriggerScan(bg)
	if err := o3.PromoteProvider(bg, "d"); !errors.Is(err, errInjected) {
		t.Errorf("factory failure: %v", err)
	}
}

func TestRequeueForRetry_StopsAtMaxRetries(t *testing.T) {
	llm := &scriptLLM{name: "p", alive: true, errs: []error{errInjected, errInjected}}
	e := newEngine(t, &noopWriter{}, nil, nil, llm)
	id := e.queue(domain.Task{Instruction: "cap", RetryCount: 2})
	got := e.runOne(id, domain.StatusFailed)
	if got.RetryCount != 2 {
		t.Errorf("retry count must not exceed the cap, got %d", got.RetryCount)
	}
}

func TestSearchKnowledge_UsesFixedFetchWindow(t *testing.T) {
	var entries []domain.ProjectKnowledge
	for i := 0; i < 30; i++ {
		entries = append(entries, domain.ProjectKnowledge{Topic: "t", TokenCount: 1, ProjectPath: "/p"})
	}
	b := services.NewBrainService(&mockKnowledgeRepo{entries: entries}, nil)
	if got, _ := b.SearchKnowledge(bg, "/p", "q", 20); len(got) != 20 {
		t.Errorf("item limit 20: got %d", len(got))
	}
	if got, _ := b.SearchKnowledge(bg, "/p", "q", 400); len(got) != 20 {
		t.Errorf("a token budget still sees at most the 20 fetched hits: got %d", len(got))
	}
}

func TestIngestFromFile_SectionHeadingWithoutBodyAtEOF(t *testing.T) {
	repo := &mockKnowledgeRepo{}
	b := services.NewBrainService(repo, nil)
	md := writeMD(t, "# T\nA sufficiently long preamble paragraph here.\n## Dangling heading") // no trailing newline
	if n, err := b.IngestFromFile(bg, "/p", md); err != nil || n != 1 {
		t.Errorf("n=%d err=%v", n, err)
	}
}

func TestGetFileMap_FocusFilter(t *testing.T) {
	repo := &mockKnowledgeRepo{entries: []domain.ProjectKnowledge{
		{Kind: domain.KnowledgeFileMap, Topic: "Auth module", Content: "auth.go", ProjectPath: "/p"},
		{Kind: domain.KnowledgeFileMap, Topic: "DB module", Content: "db.go", ProjectPath: "/p"},
	}}
	b := services.NewBrainService(repo, nil)
	if got, _ := b.GetFileMap(bg, "/p", "AUTH"); len(got) != 1 || got[0] != "auth.go" {
		t.Errorf("focus filter: %v", got)
	}
	if got, _ := b.GetFileMap(bg, "/p", ""); len(got) != 2 {
		t.Errorf("no focus: %v", got)
	}
}
