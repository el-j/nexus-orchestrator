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
		proc, err := os.FindProcess(sess.PID)
		if err == nil {
			if force {
				if err := proc.Kill(); err != nil {
					log.Printf("session_service: kill process %d: %v", proc.Pid, err)
				}
			} else {
				// On Unix systems, you'd send SIGTERM.
				// Since os.Interrupt works cross-platform mostly:
				if err := proc.Signal(os.Interrupt); err != nil {
					log.Printf("session_service: signal process %d: %v", proc.Pid, err)
				}
			}
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
	n, err := repo.PurgeDisconnected(ctx, DefaultPurgeDisconnectAge)
	if err != nil {
		return 0, fmt.Errorf("orchestrator: purge disconnected sessions: %w", err)
	}
	return n, nil
}

// GetDiscoveredAgents returns known AI agent processes, triggering a fresh scan
// if the last scan was more than 30 seconds ago.
func (o *OrchestratorService) GetDiscoveredAgents(ctx context.Context) ([]domain.DiscoveredAgent, error) {
	o.lastAgentScanMu.Lock()
	shouldScan := o.agentScanner != nil && (o.lastAgentScan.IsZero() || time.Since(o.lastAgentScan) > 30*time.Second)
	o.lastAgentScanMu.Unlock()

	var scanned []domain.DiscoveredAgent
	if shouldScan {
		agents, err := o.agentScanner.ScanAgents(ctx)
		if err != nil {
			log.Printf("orchestrator: scan agents: %v", err)
		} else {
			scanned = agents
			if o.agentRepo != nil {
				for _, a := range agents {
					if err := o.agentRepo.UpsertDiscoveredAgent(ctx, a); err != nil {
						log.Printf("orchestrator: upsert discovered agent %s: %v", a.ID, err)
					}
				}
			}
		}
		o.lastAgentScanMu.Lock()
		o.lastAgentScan = time.Now()
		o.lastAgentScanMu.Unlock()
	}

	if o.agentRepo != nil {
		return o.agentRepo.ListDiscoveredAgents(ctx)
	}
	return scanned, nil
}

// GetDiscoveredPlanFiles scans for plan/task/orchestration files near projectPath,
// persists results to SQLite, and returns the stored list for the project.
// If projectPath is empty, returns all stored plan files without scanning.
func (o *OrchestratorService) GetDiscoveredPlanFiles(ctx context.Context, projectPath string) ([]domain.DiscoveredPlanFile, error) {
	if o.agentScanner == nil && o.planFileRepo == nil {
		return nil, domain.ErrSubsystemNotConfigured
	}
	// When no project specified, return all persisted plan files.
	if projectPath == "" {
		if o.planFileRepo != nil {
			return o.planFileRepo.ListPlanFiles(ctx, "")
		}
		return nil, nil
	}
	if o.agentScanner != nil {
		files, err := o.agentScanner.ScanPlanFiles(ctx, []string{projectPath})
		if err != nil {
			return nil, err
		}
		if o.planFileRepo != nil {
			for _, f := range files {
				if err := o.planFileRepo.UpsertPlanFile(ctx, f); err != nil {
					log.Printf("session_service: upsert plan file %s: %v", f.Path, err)
				}
			}
		}
	}
	if o.planFileRepo != nil {
		return o.planFileRepo.ListPlanFiles(ctx, projectPath)
	}
	return nil, nil
}

// DelegateToNexus marks the AI session as delegated to the nexus orchestrator
// and returns a formatted instruction string for the agent.
func (o *OrchestratorService) DelegateToNexus(ctx context.Context, sessionID string) (string, error) {
	if o.aiSessionRepo == nil {
		return "", fmt.Errorf("orchestrator: delegate to nexus: session repo not configured")
	}
	session, err := o.aiSessionRepo.GetAISessionByID(ctx, sessionID)
	if err != nil {
		return "", fmt.Errorf("orchestrator: delegate to nexus: %w", err)
	}
	now := time.Now()
	session.DelegatedToNexus = true
	session.DelegationTimestamp = &now
	if err := o.aiSessionRepo.SaveAISession(ctx, session); err != nil {
		return "", fmt.Errorf("orchestrator: delegate to nexus: save: %w", err)
	}
	return o.delegationInstruction(session, now), nil
}
