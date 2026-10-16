package services_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

func limit(provider string, reset time.Time) *domain.ProviderError {
	return &domain.ProviderError{Provider: provider, Kind: domain.ErrKindRateLimited, Status: 429, ResetAt: reset,
		Msg: provider + ": rate limited (429)"}
}

func TestEveryProviderCoolingDownMakesTheTaskWaitInsteadOfFail(t *testing.T) {
	reset := time.Now().Add(400 * time.Millisecond)
	a := &scriptLLM{name: "anthropic-main", alive: true, errs: []error{limit("anthropic-main", reset)}, replies: []string{"from a"}}
	b := &scriptLLM{name: "ollama-backup", alive: true, errs: []error{limit("ollama-backup", reset.Add(300*time.Millisecond))}, replies: []string{"from b"}}
	e := newEngine(t, &noopWriter{}, nil, nil, a, b)

	id := e.queue(domain.Task{Instruction: "limited", Role: "architect"})
	got := e.runOne(id, domain.StatusWaiting)
	if got.RetryCount != 0 {
		t.Errorf("waiting does not use up a retry: %d", got.RetryCount)
	}
	if got.WaitUntil.Before(reset) || got.WaitUntil.After(reset.Add(3*time.Second)) {
		t.Errorf("the task waits for the earliest reset (%v), WaitUntil = %v", reset, got.WaitUntil)
	}
	for _, want := range []string{"all providers are cooling down", "waiting until"} {
		if !strings.Contains(got.Logs, want) {
			t.Errorf("logs lack %q:\n%s", want, got.Logs)
		}
	}
	if !hasEvent(e.bcast, ports.EventTaskWaiting, id) {
		t.Error("a task.waiting event must be published")
	}
	if q, _ := e.orch.GetQueue(); len(q) != 1 || q[0].Status != domain.StatusWaiting {
		t.Errorf("a waiting task is in the queue: %+v", q)
	}
	if e.orch.ProcessNext() {
		t.Error("nothing is queued while the task waits")
	}

	// Before the time is up nothing is resumed.
	if n, err := e.orch.ResumeWaitingTasks(context.Background(), time.Now()); err != nil || n != 0 {
		t.Fatalf("too early: resumed %d, %v", n, err)
	}
	if e.status(id).Status != domain.StatusWaiting {
		t.Fatal("still waiting")
	}

	// At the reset the task is queued again, without a wait time, and runs on the provider that is back.
	if n, err := e.orch.ResumeWaitingTasks(context.Background(), got.WaitUntil.Add(time.Millisecond)); err != nil || n != 1 {
		t.Fatalf("resume: %d %v", n, err)
	}
	if s := e.status(id); s.Status != domain.StatusQueued || !s.WaitUntil.IsZero() {
		t.Fatalf("after resume: %+v", s)
	}
	time.Sleep(time.Until(reset.Add(350 * time.Millisecond))) // both cooldowns over
	done := e.runOne(id, domain.StatusCompleted)
	if a.calls != 2 || !strings.Contains(done.Logs, "waiting until") {
		t.Errorf("a.calls=%d logs=%s", a.calls, done.Logs)
	}
}

func TestAProviderThatIsNotCoolingDownMeansNoWaiting(t *testing.T) {
	reset := time.Now().Add(time.Hour)
	limited := &scriptLLM{name: "limited", alive: true, errs: []error{limit("limited", reset)}}
	broken := &scriptLLM{name: "broken", alive: true, errs: []error{errInjected, errInjected, errInjected}}
	e := newEngine(t, &noopWriter{}, nil, nil, limited, broken)
	// "broken" failed with an ordinary error, so it is not cooling down: no reason to wait, the normal retry rules apply.
	e.runOne(e.queue(domain.Task{Instruction: "mixed", Role: "architect"}), domain.StatusQueued)
}

func TestResumeWaitingTasksOnlyTouchesDueWaitingTasks(t *testing.T) {
	e := newEngine(t, &noopWriter{}, nil, nil, &scriptLLM{name: "p", alive: true})
	ctx := context.Background()
	now := time.Now()
	for _, tk := range []domain.Task{
		{ID: "due", Status: domain.StatusWaiting, WaitUntil: now.Add(-time.Minute)},
		{ID: "later", Status: domain.StatusWaiting, WaitUntil: now.Add(time.Hour)},
		{ID: "plain", Status: domain.StatusQueued},
		{ID: "no-time", Status: domain.StatusWaiting},
	} {
		tk.ProjectPath = "/proj"
		if err := e.repo.memRepo.Save(ctx, tk); err != nil {
			t.Fatal(err)
		}
	}
	n, err := e.orch.ResumeWaitingTasks(ctx, now)
	if err != nil || n != 2 {
		t.Fatalf("resumed %d, %v (due and the one without a time)", n, err)
	}
	for id, want := range map[string]domain.TaskStatus{"due": domain.StatusQueued, "later": domain.StatusWaiting, "plain": domain.StatusQueued, "no-time": domain.StatusQueued} {
		if got := e.status(id).Status; got != want {
			t.Errorf("%s = %s, want %s", id, got, want)
		}
	}
}

func hasEvent(b *capturingBroadcaster, typ ports.EventType, taskID string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ev := range b.events {
		if ev.Type == typ && ev.TaskID == taskID {
			return true
		}
	}
	return false
}

func TestAWaitingTaskIsQueuedAgainByItsOwnTimer(t *testing.T) {
	reset := time.Now().Add(20 * time.Millisecond)
	only := &scriptLLM{name: "only", alive: true, errs: []error{limit("only", reset)}, replies: []string{"back"}}
	e := newEngine(t, &noopWriter{}, nil, nil, only)
	id := e.queue(domain.Task{Instruction: "timer"})
	e.runOne(id, domain.StatusWaiting)
	deadline := time.Now().Add(4 * time.Second) // reset + the one second grace
	for e.status(id).Status != domain.StatusQueued {
		if time.Now().After(deadline) {
			t.Fatalf("the timer must queue the task again, status = %s", e.status(id).Status)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !e.status(id).WaitUntil.IsZero() {
		t.Error("the wait time is cleared")
	}
}

func TestAHoldThatCannotBeSavedFallsBackToTheNormalRetryRules(t *testing.T) {
	only := &scriptLLM{name: "only", alive: true, errs: []error{limit("only", time.Now().Add(time.Hour))}}
	e := newEngine(t, &noopWriter{}, nil, nil, only)
	id := e.queue(domain.Task{Instruction: "unsaveable"})
	e.repo.set("Update", true)
	if !e.orch.ProcessNext() {
		t.Fatal("no task claimed")
	}
	e.repo.set("Update", false)
	if got := e.status(id); got.Status == domain.StatusWaiting {
		t.Errorf("without a saved hold the task is not waiting: %s", got.Status)
	}
}

func TestResumeWaitingTasksReportsAndSurvivesRepositoryErrors(t *testing.T) {
	e := newEngine(t, &noopWriter{}, nil, nil, &scriptLLM{name: "p", alive: true})
	ctx := context.Background()
	e.repo.set("GetAll", true)
	if _, err := e.orch.ResumeWaitingTasks(ctx, time.Now()); err == nil {
		t.Error("a listing failure is an error")
	}
	e.repo.set("GetAll", false)

	if err := e.repo.memRepo.Save(ctx, domain.Task{ID: "w", ProjectPath: "/proj", Status: domain.StatusWaiting, WaitUntil: time.Now().Add(-time.Second)}); err != nil {
		t.Fatal(err)
	}
	e.repo.set("UpdateStatusIfCurrent", true)
	if n, _ := e.orch.ResumeWaitingTasks(ctx, time.Now()); n != 0 || e.status("w").Status != domain.StatusWaiting {
		t.Errorf("a failed status change resumes nothing: %d", n)
	}
	e.repo.set("UpdateStatusIfCurrent", false)
	e.repo.set("Update", true) // clearing the wait time fails: the task is queued all the same
	if n, _ := e.orch.ResumeWaitingTasks(ctx, time.Now()); n != 1 || e.status("w").Status != domain.StatusQueued {
		t.Errorf("queued even if the wait time could not be cleared: %d %s", n, e.status("w").Status)
	}
}

func TestAWaitingTaskCanBeCancelled(t *testing.T) {
	only := &scriptLLM{name: "only", alive: true, errs: []error{limit("only", time.Now().Add(time.Hour))}}
	e := newEngine(t, &noopWriter{}, nil, nil, only)
	id := e.queue(domain.Task{Instruction: "cancel while waiting"})
	e.runOne(id, domain.StatusWaiting)
	if err := e.orch.CancelTask(id); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if got := e.status(id).Status; got != domain.StatusCancelled {
		t.Errorf("status = %s", got)
	}
	// A cancelled task is never resumed.
	if n, _ := e.orch.ResumeWaitingTasks(context.Background(), time.Now().Add(2*time.Hour)); n != 0 {
		t.Errorf("resumed %d", n)
	}
}
