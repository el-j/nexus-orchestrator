package services_test

import (
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

var _ ports.TaskEventSource = (*services.OrchestratorService)(nil)

// next returns the next event or fails the test after a second.
func next(t *testing.T, ch <-chan ports.TaskEvent) ports.TaskEvent {
	t.Helper()
	select {
	case ev, ok := <-ch:
		if !ok {
			t.Fatal("channel closed")
		}
		return ev
	case <-time.After(time.Second):
		t.Fatal("no event")
		return ports.TaskEvent{}
	}
}

func TestSubscribeTaskEvents_SeesLifecycleAndStreamEvents(t *testing.T) {
	llm := &streamLLM{scriptLLM: &scriptLLM{name: "live", alive: true}, pieces: []string{"hi"}}
	e := newEngine(t, &noopWriter{}, newMemSessionRepo(), nil, llm)
	events, cancel := e.orch.SubscribeTaskEvents()
	defer cancel()
	id := e.queue(domain.Task{Instruction: "tap"})
	e.runOne(id, domain.StatusCompleted)

	var types []ports.EventType
	for len(events) > 0 {
		types = append(types, (<-events).Type)
	}
	has := func(want ports.EventType) bool {
		for _, ty := range types {
			if ty == want {
				return true
			}
		}
		return false
	}
	if !has(ports.EventTaskRestart) || !has(ports.EventTaskChunk) || !has(ports.EventTaskCompleted) {
		t.Errorf("events seen: %v", types)
	}
}

func TestSubscribeTaskEvents_WorksWithoutABroadcaster(t *testing.T) {
	e := newEngine(t, &noopWriter{}, newMemSessionRepo(), nil, &scriptLLM{name: "p", alive: true, replies: []string{"x"}})
	e.orch.SetBroadcaster(nil)
	events, cancel := e.orch.SubscribeTaskEvents()
	defer cancel()
	id := e.queue(domain.Task{Instruction: "nobcast"})
	e.runOne(id, domain.StatusCompleted)
	if ev := next(t, events); ev.TaskID != id {
		t.Errorf("event for %q", ev.TaskID)
	}
}

func TestSubscribeTaskEvents_CancelClosesTheChannelAndStopsDelivery(t *testing.T) {
	e := newEngine(t, &noopWriter{}, newMemSessionRepo(), nil, &scriptLLM{name: "p", alive: true, replies: []string{"x"}})
	events, cancel := e.orch.SubscribeTaskEvents()
	cancel()
	cancel() // idempotent
	if _, ok := <-events; ok {
		t.Fatal("channel must be closed after cancel")
	}
	id := e.queue(domain.Task{Instruction: "after"})
	e.runOne(id, domain.StatusCompleted) // must not panic on the closed subscriber
}

func TestSubscribeTaskEvents_ASlowSubscriberDropsInsteadOfBlocking(t *testing.T) {
	e := newEngine(t, &noopWriter{}, newMemSessionRepo(), nil, &scriptLLM{name: "p", alive: true, replies: []string{"x"}})
	_, cancel := e.orch.SubscribeTaskEvents() // never read
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 400; i++ {
			e.orch.PublishForTest(ports.TaskEvent{Type: ports.EventTaskChunk, TaskID: "x", Text: "y"})
		}
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("publishing blocked on a slow subscriber")
	}
}
