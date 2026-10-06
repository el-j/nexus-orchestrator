package services

import (
	"context"
	"time"
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
