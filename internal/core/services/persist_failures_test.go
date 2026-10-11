package services_test

import (
	"context"
	"errors"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

// When the database refuses every write after a task was claimed, the engine must
// keep going: it logs, still announces the outcome to subscribers, and never
// panics. The row stays PROCESSING (the next start's stale-task sweep recovers it).
func TestEngine_SurvivesPersistenceFailuresWhileSettlingATask(t *testing.T) {
	pass := &mockCommandRunner{}
	fail := &mockCommandRunner{runFunc: func(context.Context, string, string) (string, error) {
		return "boom", errors.New("exit 1")
	}}
	code := "```go\npackage a\n```"
	for name, tc := range map[string]struct {
		task   domain.Task
		writer *contextWriter
		runner *mockCommandRunner
		llm    *scriptLLM
		want   domain.TaskStatus
	}{
		"plain completion":      {domain.Task{Instruction: "a", TargetFile: "a.go"}, &contextWriter{}, nil, &scriptLLM{name: "p", alive: true, replies: []string{code}}, domain.StatusCompleted},
		"write failure":         {domain.Task{Instruction: "b", TargetFile: "a.go"}, &contextWriter{writeErr: errors.New("disk")}, nil, &scriptLLM{name: "p", alive: true, replies: []string{code}}, domain.StatusFailed},
		"verification passes":   {domain.Task{Instruction: "c", TargetFile: "a.go", VerificationCommand: "t"}, &contextWriter{}, pass, &scriptLLM{name: "p", alive: true, replies: []string{code}}, domain.StatusCompleted},
		"verification fails":    {domain.Task{Instruction: "d", TargetFile: "a.go", VerificationCommand: "t"}, &contextWriter{}, fail, &scriptLLM{name: "p", alive: true, replies: []string{code}}, domain.StatusFailed},
		"generation fails":      {domain.Task{Instruction: "e", TargetFile: "a.go"}, &contextWriter{}, nil, &scriptLLM{name: "p", alive: true, errs: []error{errors.New("down")}}, domain.StatusFailed},
		"agent finishes":        {agentTask(func(t *domain.Task) { t.Instruction = "f" }), &contextWriter{}, nil, &scriptLLM{name: "p", alive: true, limit: 100000, replies: []string{tool(`{"tool":"finish"}`)}}, domain.StatusCompleted},
		"agent runs out":        {agentTask(func(t *domain.Task) { t.Instruction = "g"; t.MaxAgentTurns = 1 }), &contextWriter{}, nil, &scriptLLM{name: "p", alive: true, limit: 100000, replies: []string{"thinking"}}, domain.StatusFailed},
		"agent provider failed": {agentTask(func(t *domain.Task) { t.Instruction = "h" }), &contextWriter{}, nil, &scriptLLM{name: "p", alive: true, limit: 100000, errs: []error{errors.New("down")}}, domain.StatusFailed},
		"agent too large":       {agentTask(func(t *domain.Task) { t.Instruction = "i" }), &contextWriter{}, nil, &scriptLLM{name: "p", alive: true, limit: 10}, domain.StatusTooLarge},
	} {
		t.Run(name, func(t *testing.T) {
			var e *engineFixture
			if tc.runner != nil {
				e = newEngine(t, tc.writer, nil, tc.runner, tc.llm)
			} else {
				e = newEngine(t, tc.writer, nil, nil, tc.llm)
			}
			id := e.queue(tc.task)
			for _, op := range []string{"UpdateStatus", "UpdateLogs", "Update", "UpdateStatusIfCurrent"} {
				e.repo.set(op, true)
			}
			if !e.orch.ProcessNext() {
				t.Fatal("ProcessNext found no task")
			}
			if got := e.status(id).Status; got != domain.StatusProcessing {
				t.Errorf("row = %s, want PROCESSING because no write could land", got)
			}
			e.bcast.mu.Lock()
			defer e.bcast.mu.Unlock()
			var last domain.TaskStatus
			for _, ev := range e.bcast.events {
				if ev.TaskID == id {
					last = ev.Status
				}
			}
			if last != tc.want {
				t.Errorf("announced %q, want %q", last, tc.want)
			}
		})
	}
}
