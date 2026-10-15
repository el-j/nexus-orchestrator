package services_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// streamLLM is a scriptLLM that also streams: each reply is delivered in pieces to onChunk.
// failAfter > 0 makes the first call deliver that many pieces and then fail.
type streamLLM struct {
	*scriptLLM
	pieces    []string
	failAfter int
	usedChat  int
}

func (s *streamLLM) ChatStream(ctx context.Context, h []domain.Message, onChunk func(string)) (string, error) {
	if s.failAfter > 0 {
		s.failAfter = 0
		onChunk(s.pieces[0])
		return "", errors.New("connection reset mid-stream")
	}
	for _, p := range s.pieces {
		onChunk(p)
	}
	return strings.Join(s.pieces, ""), nil
}

func (s *streamLLM) Chat(ctx context.Context, h []domain.Message) (string, error) {
	s.usedChat++
	return s.scriptLLM.Chat(ctx, h)
}

// taskEvents returns the stream events (restart and chunk) recorded for one task, in order.
func taskEvents(e *engineFixture, id string) []ports.TaskEvent {
	e.bcast.mu.Lock()
	defer e.bcast.mu.Unlock()
	var out []ports.TaskEvent
	for _, ev := range e.bcast.events {
		if ev.TaskID == id && (ev.Type == ports.EventTaskChunk || ev.Type == ports.EventTaskRestart) {
			out = append(out, ev)
		}
	}
	return out
}

func joinedText(evs []ports.TaskEvent) string {
	var b strings.Builder
	for _, ev := range evs {
		b.WriteString(ev.Text)
	}
	return b.String()
}

func TestStream_ProviderChunksReachSubscribersAndBecomeTheResult(t *testing.T) {
	llm := &streamLLM{scriptLLM: &scriptLLM{name: "live", alive: true}, pieces: []string{"Hel", "lo ", "world"}}
	w := &contextWriter{}
	e := newEngine(t, w, newMemSessionRepo(), nil, llm)
	id := e.queue(domain.Task{Instruction: "stream me", TargetFile: "out.txt"})
	e.runOne(id, domain.StatusCompleted)

	evs := taskEvents(e, id)
	if len(evs) < 2 || evs[0].Type != ports.EventTaskRestart {
		t.Fatalf("an attempt must open with a restart event, got %+v", evs)
	}
	if got := joinedText(evs); got != "Hello world" {
		t.Errorf("streamed text = %q", got)
	}
	for i, ev := range evs[1:] {
		if ev.Type != ports.EventTaskChunk || ev.Seq != i+1 {
			t.Errorf("chunk %d: type=%s seq=%d, want chunk with seq %d", i, ev.Type, ev.Seq, i+1)
		}
	}
	if llm.usedChat != 0 {
		t.Error("a streaming provider must not be called through Chat")
	}
	if len(w.writes) != 1 || !strings.Contains(w.writes[0], "Hello world") {
		t.Errorf("result not written from the streamed text: %v", w.writes)
	}
}

func TestStream_ProviderWithoutStreamingSendsOneChunk(t *testing.T) {
	llm := &scriptLLM{name: "plain", alive: true, replies: []string{"whole answer"}}
	e := newEngine(t, &noopWriter{}, newMemSessionRepo(), nil, llm)
	id := e.queue(domain.Task{Instruction: "no stream"})
	e.runOne(id, domain.StatusCompleted)

	evs := taskEvents(e, id)
	if len(evs) != 2 || evs[0].Type != ports.EventTaskRestart || evs[1].Type != ports.EventTaskChunk || evs[1].Text != "whole answer" {
		t.Errorf("want restart + one chunk with the whole reply, got %+v", evs)
	}
}

func TestStream_FailureMidStreamRestartsOnTheNextProvider(t *testing.T) {
	flaky := &streamLLM{scriptLLM: &scriptLLM{name: "flaky", alive: true}, pieces: []string{"par", "tial"}, failAfter: 1}
	backup := &scriptLLM{name: "backup", alive: true, replies: []string{"clean answer"}}
	w := &contextWriter{}
	e := newEngine(t, w, newMemSessionRepo(), nil, flaky, backup)
	id := e.queue(domain.Task{Instruction: "needs backup", TargetFile: "out.txt", Role: "architect"})
	e.runOne(id, domain.StatusCompleted)

	evs := taskEvents(e, id)
	var types []ports.EventType
	for _, ev := range evs {
		types = append(types, ev.Type)
	}
	want := []ports.EventType{ports.EventTaskRestart, ports.EventTaskChunk, ports.EventTaskRestart, ports.EventTaskChunk}
	if len(types) != len(want) {
		t.Fatalf("events = %v, want %v", types, want)
	}
	for i := range want {
		if types[i] != want[i] {
			t.Fatalf("events = %v, want %v", types, want)
		}
	}
	if evs[1].Text != "par" || evs[3].Text != "clean answer" || evs[3].Seq != 1 {
		t.Errorf("partial text must be followed by a restart and a fresh sequence: %+v", evs)
	}
	if len(w.writes) != 1 || strings.Contains(w.writes[0], "par") || !strings.Contains(w.writes[0], "clean answer") {
		t.Errorf("the partial text must not reach the output: %v", w.writes)
	}
}

func TestStream_AgentTurnsStream(t *testing.T) {
	finish := tool(`{"tool":"finish","summary":"done"}`)
	llm := &streamLLM{scriptLLM: &scriptLLM{name: "agent", alive: true}, pieces: []string{finish[:10], finish[10:]}}
	e := newEngine(t, &noopWriter{}, newMemSessionRepo(), nil, llm)
	id := e.queue(domain.Task{Instruction: "agentic stream", Agentic: true})
	e.runOne(id, domain.StatusCompleted)
	if got := joinedText(taskEvents(e, id)); got != finish {
		t.Errorf("agent turn text = %q", got)
	}
}

// Without a session store the engine sends a single prompt through GenerateCode. A provider that
// can stream still streams it; one that cannot sends the whole reply as one chunk.
func TestStream_SinglePromptPathStreamsToo(t *testing.T) {
	llm := &streamLLM{scriptLLM: &scriptLLM{name: "live", alive: true}, pieces: []string{"one ", "shot"}}
	w := &contextWriter{}
	e := newEngine(t, w, nil, nil, llm)
	id := e.queue(domain.Task{Instruction: "no session", TargetFile: "out.txt"})
	e.runOne(id, domain.StatusCompleted)

	evs := taskEvents(e, id)
	if len(evs) < 2 || evs[0].Type != ports.EventTaskRestart || joinedText(evs) != "one shot" {
		t.Fatalf("events = %+v", evs)
	}
	if len(w.writes) != 1 || !strings.Contains(w.writes[0], "one shot") {
		t.Errorf("result not written from the streamed text: %v", w.writes)
	}
}

func TestStream_SinglePromptPathWithoutStreamingSendsOneChunk(t *testing.T) {
	llm := &scriptLLM{name: "plain", alive: true, replies: []string{"whole answer"}}
	e := newEngine(t, &noopWriter{}, nil, nil, llm)
	id := e.queue(domain.Task{Instruction: "no session no stream"})
	e.runOne(id, domain.StatusCompleted)

	evs := taskEvents(e, id)
	if len(evs) != 2 || evs[0].Type != ports.EventTaskRestart || evs[1].Text != "whole answer" {
		t.Errorf("want restart + one chunk, got %+v", evs)
	}
}

func TestStream_SinglePromptFailureStillEmitsTheRestart(t *testing.T) {
	llm := &scriptLLM{name: "broken", alive: true, errs: []error{errors.New("down")}}
	e := newEngine(t, &noopWriter{}, nil, nil, llm)
	id := e.queue(domain.Task{Instruction: "fails"})
	e.runOne(id, domain.StatusQueued) // the first failure re-queues the task for a retry
	if evs := taskEvents(e, id); len(evs) != 1 || evs[0].Type != ports.EventTaskRestart {
		t.Errorf("want only the restart event, got %+v", evs)
	}
}
