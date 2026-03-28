package services

import (
	"context"
	"log"
	"sync"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

const (
	activityPollInterval    = 5 * time.Second
	activityRetentionWindow = 24 * time.Hour
	retentionPurgeInterval  = 1 * time.Hour
	idleTimeout             = 5 * time.Minute
	disconnectTimeout       = 15 * time.Minute
)

// ActivityService polls activity readers, persists observations, bridges discovered
// agents to sessions, and broadcasts SSE events for the observatory dashboard.
type ActivityService struct {
	activityRepo ports.AIActivityRepository
	sessionRepo  ports.AISessionRepository
	readers      []ports.ActivityReader
	broadcaster  ports.ActivityBroadcaster // optional; may be nil

	stopCh chan struct{}
	mu     sync.Mutex
	// lastSeen tracks when we last received activity per key "agentName\x00projectPath"
	lastSeen map[string]time.Time
}

// NewActivityService creates an ActivityService. Call Start() to begin polling.
func NewActivityService(
	activityRepo ports.AIActivityRepository,
	sessionRepo ports.AISessionRepository,
	readers ...ports.ActivityReader,
) *ActivityService {
	return &ActivityService{
		activityRepo: activityRepo,
		sessionRepo:  sessionRepo,
		readers:      readers,
		stopCh:       make(chan struct{}),
		lastSeen:     make(map[string]time.Time),
	}
}

// SetBroadcaster wires an ActivityBroadcaster for SSE event delivery.
func (s *ActivityService) SetBroadcaster(b ports.ActivityBroadcaster) {
	s.broadcaster = b
}

// Start launches the background polling and purge goroutines.
// It is safe to call multiple times but only one goroutine is started.
func (s *ActivityService) Start() {
	go s.pollLoop()
	go s.purgeLoop()
}

// Stop signals the background goroutines to exit gracefully.
func (s *ActivityService) Stop() {
	close(s.stopCh)
}

// GetRecentActivities queries the repository with the given filter.
func (s *ActivityService) GetRecentActivities(ctx context.Context, f domain.ActivityFilter) ([]domain.AIActivity, error) {
	return s.activityRepo.ListActivities(ctx, f)
}

// GetTimeline returns a chronological merged feed across all agents.
func (s *ActivityService) GetTimeline(ctx context.Context, since time.Time, limit int) ([]domain.AIActivity, error) {
	return s.activityRepo.ListActivities(ctx, domain.ActivityFilter{Since: since, Limit: limit})
}

func (s *ActivityService) pollLoop() {
	ticker := time.NewTicker(activityPollInterval)
	defer ticker.Stop()

	// Do an immediate first poll.
	s.poll()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.poll()
		}
	}
}

func (s *ActivityService) purgeLoop() {
	ticker := time.NewTicker(retentionPurgeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			cutoff := time.Now().Add(-activityRetentionWindow)
			ctx := context.Background()
			n, err := s.activityRepo.PurgeOlderThan(ctx, cutoff)
			if err != nil {
				log.Printf("activity_service: purge: %v", err)
			} else if n > 0 {
				log.Printf("activity_service: purged %d old activities", n)
			}
		}
	}
}

func (s *ActivityService) poll() {
	ctx := context.Background()
	since := time.Now().Add(-activityRetentionWindow)

	for _, reader := range s.readers {
		activities, err := reader.ReadActivities(ctx, since)
		if err != nil {
			log.Printf("activity_service: reader %s: %v", reader.SourceName(), err)
			continue
		}
		for _, a := range activities {
			if err := s.activityRepo.SaveActivity(ctx, a); err != nil {
				log.Printf("activity_service: save activity: %v", err)
				continue
			}
			s.bridgeSession(ctx, a)
			if s.broadcaster != nil {
				s.broadcaster.BroadcastActivityEvent(a)
			}
		}
	}
	s.checkIdleDisconnect(ctx)
}

// bridgeSession (TASK-338): auto-create or update an AISession from observed activity.
func (s *ActivityService) bridgeSession(ctx context.Context, a domain.AIActivity) {
	if a.AgentName == "" {
		return
	}
	key := a.AgentName + "\x00" + a.ProjectPath

	// Update last-seen tracking (protected by single poll goroutine, but use mu for
	// safety if callers ever call concurrently).
	s.mu.Lock()
	s.lastSeen[key] = time.Now()
	s.mu.Unlock()

	// Look for an existing session by agentName + projectPath.
	sessions, err := s.sessionRepo.ListAISessions(ctx)
	if err != nil {
		log.Printf("activity_service: list sessions: %v", err)
		return
	}
	var existing *domain.AISession
	for i := range sessions {
		if sessions[i].AgentName == a.AgentName && sessions[i].ProjectPath == a.ProjectPath {
			existing = &sessions[i]
			break
		}
	}

	now := time.Now()
	if existing == nil {
		// Auto-create session.
		sess := domain.AISession{
			ID:              "act-" + a.AgentName + "-" + a.ID,
			Source:          "discovered",
			AgentName:       a.AgentName,
			ProjectPath:     a.ProjectPath,
			Status:          domain.SessionStatusActive,
			LastActivity:    now,
			CurrentActivity: a.Summary,
			MessageCount:    1,
			TokensUsed:      int64(a.TokensIn + a.TokensOut),
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if err := s.sessionRepo.SaveAISession(ctx, sess); err != nil {
			log.Printf("activity_service: create session: %v", err)
		}
		return
	}

	// Update existing session.
	existing.Status = domain.SessionStatusActive
	existing.LastActivity = now
	existing.CurrentActivity = a.Summary
	existing.MessageCount++
	existing.TokensUsed += int64(a.TokensIn + a.TokensOut)
	if a.Summary != "" {
		existing.LastMessage = a.Summary
	}
	existing.UpdatedAt = now
	if err := s.sessionRepo.SaveAISession(ctx, *existing); err != nil {
		log.Printf("activity_service: update session: %v", err)
	}
}

// checkIdleDisconnect marks sessions idle/disconnected based on last-seen times.
func (s *ActivityService) checkIdleDisconnect(ctx context.Context) {
	s.mu.Lock()
	lastSeen := make(map[string]time.Time, len(s.lastSeen))
	for k, v := range s.lastSeen {
		lastSeen[k] = v
	}
	s.mu.Unlock()

	sessions, err := s.sessionRepo.ListAISessions(ctx)
	if err != nil {
		return
	}
	now := time.Now()
	for _, sess := range sessions {
		if sess.Source != "discovered" {
			continue // only manage auto-created sessions
		}
		key := sess.AgentName + "\x00" + sess.ProjectPath
		last, seen := lastSeen[key]
		if !seen {
			last = sess.LastActivity
		}
		age := now.Sub(last)
		var newStatus domain.AISessionStatus
		switch {
		case age >= disconnectTimeout:
			newStatus = domain.SessionStatusDisconnected
		case age >= idleTimeout:
			newStatus = domain.SessionStatusIdle
		default:
			continue
		}
		if sess.Status == newStatus {
			continue
		}
		sess.Status = newStatus
		sess.UpdatedAt = now
		if err := s.sessionRepo.SaveAISession(ctx, sess); err != nil {
			log.Printf("activity_service: status update: %v", err)
		}
	}
}
