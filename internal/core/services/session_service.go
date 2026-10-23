package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"nexus-orchestrator/internal/core/domain"

	"github.com/google/uuid"
)

// RegisterAISession registers an AI agent session, persists it, and broadcasts an event.
// If the session carries a non-empty ExternalID and a session with that ExternalID already
// exists, the existing session's last-activity is refreshed and it is returned unchanged
// (idempotent). This prevents multiple heartbeat calls from creating duplicate rows.
func (o *OrchestratorService) RegisterAISession(ctx context.Context, s domain.AISession) (domain.AISession, error) {
	o.mu.Lock()
	repo := o.aiSessionRepo
	b := o.broadcaster
	o.mu.Unlock()

	if repo == nil {
		return domain.AISession{}, fmt.Errorf("orchestrator: register ai session: no session repo configured")
	}

	now := time.Now()

	// Idempotency: if an externalId is provided, re-use the existing session.
	if s.ExternalID != "" {
		existing, err := repo.GetAISessionByExternalID(ctx, s.ExternalID)
		if err == nil {
			// Session already exists — just refresh its last-activity timestamp.
			existing.LastActivity = now
			existing.UpdatedAt = now
			existing.Status = domain.SessionStatusActive
			if saveErr := repo.SaveAISession(ctx, existing); saveErr != nil {
				return domain.AISession{}, fmt.Errorf("orchestrator: register ai session: refresh existing: %w", saveErr)
			}
			if b != nil {
				b.BroadcastAISessionEvent(domain.AISessionEvent{
					Type:        "ai_session_changed",
					AISessionID: existing.ID,
					Status:      existing.Status,
					Timestamp:   time.Now(),
				})
			}
			return existing, nil
		}
		// ErrNotFound is expected for the first registration — fall through.
	}

	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	s.Status = domain.SessionStatusActive
	s.CreatedAt = now
	s.UpdatedAt = now
	s.LastActivity = now

	if err := repo.SaveAISession(ctx, s); err != nil {
		return domain.AISession{}, fmt.Errorf("orchestrator: register ai session: %w", err)
	}
	if b != nil {
		b.BroadcastAISessionEvent(domain.AISessionEvent{
			Type:        "ai_session_changed",
			AISessionID: s.ID,
			Status:      s.Status,
			Timestamp:   time.Now(),
		})
	}
	return s, nil
}

// ListAISessions returns all persisted AI agent sessions.
func (o *OrchestratorService) ListAISessions(ctx context.Context) ([]domain.AISession, error) {
	o.mu.Lock()
	repo := o.aiSessionRepo
	o.mu.Unlock()

	if repo == nil {
		return []domain.AISession{}, nil
	}
	sessions, err := repo.ListAISessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("orchestrator: list ai sessions: %w", err)
	}
	return sessions, nil
}

// DeregisterAISession marks the session as disconnected and broadcasts an event.
func (o *OrchestratorService) DeregisterAISession(ctx context.Context, id string) error {
	o.mu.Lock()
	repo := o.aiSessionRepo
	b := o.broadcaster
	o.mu.Unlock()

	if repo == nil {
		return fmt.Errorf("orchestrator: deregister ai session: no session repo configured")
	}
	if err := repo.UpdateAISessionStatus(ctx, id, domain.SessionStatusDisconnected, time.Now()); err != nil {
		return fmt.Errorf("orchestrator: deregister ai session: %w", err)
	}
	if b != nil {
		b.BroadcastAISessionEvent(domain.AISessionEvent{
			Type:        "ai_session_changed",
			AISessionID: id,
			Status:      domain.SessionStatusDisconnected,
			Timestamp:   time.Now(),
		})
	}
	return nil
}

// TerminateAISession attempts to gracefully shut down or force kill the external agent process.
func (o *OrchestratorService) TerminateAISession(ctx context.Context, id string, force bool) error {
	o.mu.Lock()
	repo := o.aiSessionRepo
	b := o.broadcaster
	o.mu.Unlock()

	if repo == nil {
		return fmt.Errorf("orchestrator: terminate ai session: no session repo configured")
	}

	sess, err := repo.GetAISessionByID(ctx, id)
	if err != nil {
		return fmt.Errorf("orchestrator: terminate ai session: %w", err)
	}

	if sess.PID > 0 {
		// The PID was supplied by whoever registered the session, so it is only a claim.
		// Signalling it blindly would let any API caller kill any process the daemon's
		// user may kill; it is honoured only for processes Nexus itself found running.
		if reason := o.unsafeToSignal(ctx, sess.PID); reason != "" {
			log.Printf("orchestrator: terminate ai session %s: not signalling pid %d: %s", id, sess.PID, reason)
		} else if err := o.signalProcess(sess.PID, force); err != nil {
			log.Printf("orchestrator: terminate ai session %s: signal pid %d: %v", id, sess.PID, err)
		}
	}

	// Mark it disconnected logically.
	if err := repo.UpdateAISessionStatus(ctx, id, domain.SessionStatusDisconnected, time.Now()); err != nil {
		return fmt.Errorf("orchestrator: terminate ai session: log disconnect: %w", err)
	}

	if b != nil {
		b.BroadcastAISessionEvent(domain.AISessionEvent{
			Type:        "ai_session_terminated",
			AISessionID: sess.ID,
			Status:      domain.SessionStatusDisconnected,
			Timestamp:   time.Now(),
		})
	}

	return nil
}

// agentFreshness is how recently the scanner must have seen an agent process for
// its PID to be trusted (a recycled PID could otherwise point at an unrelated process).
const agentFreshness = 2 * time.Minute

// unsafeToSignal returns why pid must not be signalled, or "" when it is safe: it
// must not be the daemon, its parent or init, and must belong to an agent process
// the scanner reported as running recently.
func (o *OrchestratorService) unsafeToSignal(ctx context.Context, pid int) string {
	switch {
	case pid <= 1:
		return "reserved pid"
	case pid == os.Getpid():
		return "that is the daemon itself"
	case pid == os.Getppid():
		return "that is the daemon's parent process"
	}
	agents, err := o.GetDiscoveredAgents(ctx)
	if err != nil {
		return "could not list discovered agents: " + err.Error()
	}
	for _, a := range agents {
		if a.PID == pid && a.IsRunning && time.Since(a.LastSeen) <= agentFreshness {
			return ""
		}
	}
	return "it is not a running agent process discovered by Nexus"
}

// signalProcess interrupts (or, with force, kills) a single process.
func (o *OrchestratorService) signalProcess(pid int, force bool) error {
	if o.signalFn != nil {
		return o.signalFn(pid, force)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if force {
		return proc.Kill()
	}
	return proc.Signal(os.Interrupt)
}

// HeartbeatAISession refreshes the last-activity timestamp on an active session.
// It is intended to be called periodically by connected agents to signal liveness.
func (o *OrchestratorService) HeartbeatAISession(ctx context.Context, id string) error {
	o.mu.Lock()
	repo := o.aiSessionRepo
	o.mu.Unlock()

	if repo == nil {
		return fmt.Errorf("orchestrator: heartbeat ai session: no session repo configured")
	}
	if err := repo.UpdateAISessionStatus(ctx, id, domain.SessionStatusActive, time.Now()); err != nil {
		return fmt.Errorf("orchestrator: heartbeat ai session: %w", err)
	}
	return nil
}

// PurgeDisconnectedSessions immediately deletes all AI sessions with status
// "disconnected" that have been inactive for more than 2 hours. Returns the
// number of sessions deleted.
func (o *OrchestratorService) PurgeDisconnectedSessions(ctx context.Context) (int, error) {
	o.mu.Lock()
	repo := o.aiSessionRepo
	o.mu.Unlock()
	if repo == nil {
		return 0, fmt.Errorf("orchestrator: purge disconnected sessions: no session repo configured")
	}
	n, err := repo.PurgeDisconnected(ctx, 2*time.Hour)
	if err != nil {
		return 0, fmt.Errorf("orchestrator: purge disconnected sessions: %w", err)
	}
	return n, nil
}

// GetDiscoveredAgents returns known AI agent processes, triggering a fresh scan
// if the last scan was more than 30 seconds ago.
func (o *OrchestratorService) GetDiscoveredAgents(ctx context.Context) ([]domain.DiscoveredAgent, error) {
	o.mu.Lock()
	scanner := o.agentScanner
	repo := o.agentRepo
	o.mu.Unlock()

	o.lastAgentScanMu.Lock()
	shouldScan := scanner != nil && (o.lastAgentScan.IsZero() || time.Since(o.lastAgentScan) > 30*time.Second)
	cached := append([]domain.DiscoveredAgent(nil), o.lastScannedAgents...)
	o.lastAgentScanMu.Unlock()

	if shouldScan {
		agents, err := scanner.ScanAgents(ctx)
		if err != nil {
			log.Printf("orchestrator: scan agents: %v", err)
		} else {
			cached = agents
			if repo != nil {
				for _, a := range agents {
					if err := repo.UpsertDiscoveredAgent(ctx, a); err != nil {
						log.Printf("orchestrator: upsert discovered agent %s: %v", a.ID, err)
					}
				}
			}
		}
		o.lastAgentScanMu.Lock()
		o.lastAgentScan = time.Now()
		if err == nil {
			o.lastScannedAgents = append([]domain.DiscoveredAgent(nil), agents...)
		}
		o.lastAgentScanMu.Unlock()
	}

	if repo != nil {
		return repo.ListDiscoveredAgents(ctx)
	}
	return cached, nil
}

// GetDiscoveredPlanFiles scans for plan/task/orchestration files near projectPath,
// persists results to SQLite, and returns the stored list for the project.
// If projectPath is empty, returns all stored plan files without scanning.
func (o *OrchestratorService) GetDiscoveredPlanFiles(ctx context.Context, projectPath string) ([]domain.DiscoveredPlanFile, error) {
	o.mu.Lock()
	scanner := o.agentScanner
	planRepo := o.planFileRepo
	o.mu.Unlock()

	if scanner == nil && planRepo == nil {
		return nil, domain.ErrSubsystemNotConfigured
	}
	// When no project specified, return all persisted plan files.
	if projectPath == "" {
		if planRepo != nil {
			return planRepo.ListPlanFiles(ctx, "")
		}
		return nil, domain.ErrSubsystemNotConfigured
	}
	var files []domain.DiscoveredPlanFile
	if scanner != nil {
		var err error
		files, err = scanner.ScanPlanFiles(ctx, []string{projectPath})
		if err != nil {
			return nil, err
		}
		if planRepo != nil {
			for _, f := range files {
				_ = planRepo.UpsertPlanFile(ctx, f)
			}
		}
	}
	if planRepo != nil {
		return planRepo.ListPlanFiles(ctx, projectPath)
	}
	if files != nil {
		return files, nil
	}
	return nil, domain.ErrSubsystemNotConfigured
}

// DelegateToNexus marks the AI session as delegated to the nexus orchestrator
// and returns a formatted instruction string for the agent.
func (o *OrchestratorService) DelegateToNexus(ctx context.Context, sessionID string) (string, error) {
	o.mu.Lock()
	aiRepo := o.aiSessionRepo
	o.mu.Unlock()

	if aiRepo == nil {
		return "", fmt.Errorf("orchestrator: delegate to nexus: session repo not configured")
	}
	session, err := aiRepo.GetAISessionByID(ctx, sessionID)
	if err != nil {
		return "", fmt.Errorf("orchestrator: delegate to nexus: %w", err)
	}
	now := time.Now()
	session.DelegatedToNexus = true
	session.DelegationTimestamp = &now
	if err := aiRepo.SaveAISession(ctx, session); err != nil {
		return "", fmt.Errorf("orchestrator: delegate to nexus: save: %w", err)
	}
	return o.delegationInstruction(session, now), nil
}
