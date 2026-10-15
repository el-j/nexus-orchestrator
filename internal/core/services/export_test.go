package services

import (
	"context"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// ProcessNext runs exactly one iteration of the worker loop (claim the next
// QUEUED task and execute it) and reports whether a task was claimed. It exists
// so tests can drive the execution engine deterministically with
// WithDisableBackgroundWorkers instead of polling a background goroutine.
func (o *OrchestratorService) ProcessNext() bool { return o.processNext() }

// SetIntervals shortens the poll and purge intervals; call before Start.
func (s *ActivityService) SetIntervals(poll, purge time.Duration) {
	s.pollEvery, s.purgeEvery = poll, purge
}

// CheckIdleDisconnect exposes the idle/disconnect sweep to tests.
func (s *ActivityService) CheckIdleDisconnect(ctx context.Context) { s.checkIdleDisconnect(ctx) }

// MarkSeen records activity for an agent/project at the given time, as bridgeSession would.
func (s *ActivityService) MarkSeen(agent, project string, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastSeen[agent+"\x00"+project] = at
}

// SetSignalFunc replaces the real process signalling so tests can observe, or
// assert the absence of, signals without touching real processes.
func (o *OrchestratorService) SetSignalFunc(fn func(pid int, force bool) error) { o.signalFn = fn }

// ResolveChain exposes provider ordering: the names of the providers a task
// would be tried against, in order.
func (o *OrchestratorService) ResolveChain(task domain.Task) ([]string, error) {
	chain, err := o.resolveProviderWithFallback(context.Background(), task)
	var names []string
	for _, c := range chain {
		names = append(names, c.ProviderName())
	}
	return names, err
}

// PublishForTest sends a stream event the way the engine does, so tests can flood subscribers.
func (o *OrchestratorService) PublishForTest(ev ports.TaskEvent) { o.publish(ev) }

// GetClientForTest returns the registered client with the given name.
func (o *OrchestratorService) GetClientForTest(name string) (ports.LLMClient, bool) {
	return o.discovery.GetClientByName(name)
}

// PollNow runs one poll of the activity readers, so tests need not wait for the ticker.
func (s *ActivityService) PollNow() { s.poll() }
