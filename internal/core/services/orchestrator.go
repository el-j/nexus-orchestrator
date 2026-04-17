// Package services implements the core business logic of nexusOrchestrator.
// The OrchestratorService manages a task queue, routes code-generation tasks to
// available LLM providers, and maintains per-project conversation history.
package services

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"encoding/base64"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// discoveredAgentStore is the minimal subset of the discovered-agent storage
// interface used by OrchestratorService.
type discoveredAgentStore interface {
	UpsertDiscoveredAgent(ctx context.Context, a domain.DiscoveredAgent) error
	ListDiscoveredAgents(ctx context.Context) ([]domain.DiscoveredAgent, error)
}

// ErrQueueFull is returned by SubmitTask when the number of QUEUED tasks reaches the queue cap.
var ErrQueueFull = errors.New("queue is full")

// Option is a functional option for configuring OrchestratorService.
type Option func(*OrchestratorService)

// WithMaxRetries sets the maximum LLM call attempts before a task is permanently failed. Default: 3.
func WithMaxRetries(n int) Option {
	return func(s *OrchestratorService) { s.maxRetries = n }
}

// WithMaxResponseTokens sets the token budget reserved for the assistant reply in pre-flight checks. Default: 512.
func WithMaxResponseTokens(n int) Option {
	return func(s *OrchestratorService) { s.maxResponseTokens = n }
}

// WithCleanupInterval sets how often the session cleanup goroutine runs. Default: 2 minutes.
func WithCleanupInterval(d time.Duration) Option {
	return func(s *OrchestratorService) { s.cleanupInterval = d }
}

// WithStaleThreshold sets the session inactivity duration before it is marked disconnected. Default: 5 minutes.
func WithStaleThreshold(d time.Duration) Option {
	return func(s *OrchestratorService) { s.staleThreshold = d }
}

// WithDaemonAddr sets the base URL of the running daemon used in delegation instructions.
// Defaults to http://127.0.0.1:63987.
func WithDaemonAddr(addr string) Option {
	return func(s *OrchestratorService) { s.daemonAddr = addr }
}

// WithDisableBackgroundWorkers prevents the background worker/cleanup goroutines
// from starting. Intended for deterministic unit tests.
func WithDisableBackgroundWorkers() Option {
	return func(s *OrchestratorService) { s.disableWorkers = true }
}

// WithPlanFileRepo sets the repository used to persist discovered plan files.
func WithPlanFileRepo(r ports.DiscoveredPlanFileRepo) Option {
	return func(o *OrchestratorService) { o.planFileRepo = r }
}

// WithWatchdogInterval sets how often the task watchdog checks for stale tasks. Default: 1 minute.
func WithWatchdogInterval(d time.Duration) Option {
	return func(s *OrchestratorService) { s.watchdogInterval = d }
}

// OrchestratorService implements ports.Orchestrator and drives the worker loop.
type OrchestratorService struct {
	mu          sync.Mutex
	discovery   *DiscoveryService
	fileWriter  ports.FileWriter
	repo        ports.TaskRepository
	sessionRepo ports.SessionRepository
	runtimeCfg  ports.RuntimeConfigRepository
	broadcaster ports.EventBroadcaster // optional; nil = no event publishing
	workCh      chan struct{}          // notified when a task is enqueued; capacity 1
	stopCh      chan struct{}
	stopped     bool
	stopOnce    sync.Once
	workerWg    sync.WaitGroup // tracks the background worker goroutine
	queueCap    int            // max number of QUEUED tasks; 50 when zero
	// providerFactory builds a concrete LLMClient from a ProviderConfig.
	// Injected by entry points to keep service layer free of adapter imports.
	providerFactory    func(domain.ProviderConfig) (ports.LLMClient, error)
	providerConfigRepo ports.ProviderConfigRepository
	scanner            ports.SystemScanner
	lastScan           []domain.DiscoveredProvider
	scanMu             sync.RWMutex // guards lastScan; separate from task-queue mu
	aiSessionRepo      ports.AISessionRepository
	agentScanner       ports.AgentScanner
	agentRepo          discoveredAgentStore
	planFileRepo       ports.DiscoveredPlanFileRepo
	lastAgentScan      time.Time
	lastAgentScanMu    sync.Mutex
	maxRetries         int
	maxResponseTokens  int
	cleanupInterval    time.Duration
	watchdogInterval   time.Duration
	daemonAddr         string // base URL of the running daemon; used in delegation instructions
	staleThreshold     time.Duration
	disableWorkers     bool
}

// NewOrchestrator constructs an OrchestratorService and starts the background
// worker that pulls QUEUED tasks and sends them to the active LLM.
// sessionRepo may be nil; when nil, sessions are not persisted and GenerateCode
// is used as a fallback instead of Chat.
func NewOrchestrator(
	discovery *DiscoveryService,
	repo ports.TaskRepository,
	writer ports.FileWriter,
	sessionRepo ports.SessionRepository,
	opts ...Option,
) (*OrchestratorService, error) {
	if discovery == nil {
		return nil, fmt.Errorf("orchestrator: new: missing %s dependency", "discovery")
	}
	if repo == nil {
		return nil, fmt.Errorf("orchestrator: new: missing %s dependency", "repo")
	}
	if writer == nil {
		return nil, fmt.Errorf("orchestrator: new: missing %s dependency", "writer")
	}
	svc := &OrchestratorService{
		discovery:         discovery,
		repo:              repo,
		fileWriter:        writer,
		sessionRepo:       sessionRepo,
		workCh:            make(chan struct{}, 1),
		stopCh:            make(chan struct{}),
		maxRetries:        3,
		maxResponseTokens: 512,
		cleanupInterval:   2 * time.Minute,
		staleThreshold:    5 * time.Minute,
		daemonAddr:        "http://127.0.0.1:63987",
	}
	for _, opt := range opts {
		opt(svc)
	}
	svc.recoverStuckTasks()
	if !svc.disableWorkers {
		svc.workerWg.Add(1)
		go svc.runWorker()
		svc.workerWg.Add(1)
		go svc.runSessionCleanup()
		svc.workerWg.Add(1)
		go svc.runTaskWatchdog()
	}
	return svc, nil
}

// Stop signals the worker goroutine to exit and waits for it to finish.
// It is safe to close the backing repository only after Stop returns.
func (o *OrchestratorService) Stop() {
	o.stopOnce.Do(func() {
		o.mu.Lock()
		o.stopped = true
		o.mu.Unlock()
		close(o.stopCh)
	})
	o.workerWg.Wait()
}

// WithProviderConfigRepo sets the repository used to persist ProviderConfig records.
// Must be called before any AddProviderConfig / UpdateProviderConfig / RemoveProviderConfig
// / ListProviderConfigs call.
func (o *OrchestratorService) WithProviderConfigRepo(r ports.ProviderConfigRepository) *OrchestratorService {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.providerConfigRepo = r
	return o
}

// WithRuntimeConfigRepo sets the repository used to persist runtime-managed configuration.
func (o *OrchestratorService) WithRuntimeConfigRepo(r ports.RuntimeConfigRepository) *OrchestratorService {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.runtimeCfg = r
	return o
}

// WithProviderFactory sets the factory used by RegisterCloudProvider to construct
// new LLM adapters from a ProviderConfig. Must be called before the first
// RegisterCloudProvider call.
func (o *OrchestratorService) WithProviderFactory(fn func(domain.ProviderConfig) (ports.LLMClient, error)) *OrchestratorService {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.providerFactory = fn
	return o
}

// WithSystemScanner sets the SystemScanner used for provider discovery.
func (o *OrchestratorService) WithSystemScanner(s ports.SystemScanner) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.scanner = s
}

// WithQueueCap sets the maximum number of QUEUED tasks allowed at one time.
// When the cap is reached, SubmitTask returns ErrQueueFull.
// Default (and zero) means 50.
func (o *OrchestratorService) WithQueueCap(n int) *OrchestratorService {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.queueCap = n
	return o
}

func (o *OrchestratorService) GetRuntimeConfig(ctx context.Context) (domain.RuntimeConfig, error) {
	o.mu.Lock()
	queueCap := o.queueCap
	repo := o.runtimeCfg
	o.mu.Unlock()

	effectiveCap := queueCap
	if effectiveCap <= 0 {
		effectiveCap = DefaultQueueCap
	}

	cfg := domain.RuntimeConfig{
		QueueCap: effectiveCap,
		APIToken: os.Getenv("NEXUS_API_TOKEN"),
		MCPToken: os.Getenv("NEXUS_MCP_TOKEN"),
	}

	if repo == nil {
		return cfg, nil
	}
	stored, err := repo.GetRuntimeConfig(ctx)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return domain.RuntimeConfig{}, fmt.Errorf("orchestrator: get runtime config: %w", err)
	}
	if err == nil {
		// Prefer persisted values when env vars are not set.
		if cfg.APIToken == "" {
			cfg.APIToken = stored.APIToken
		}
		if cfg.MCPToken == "" {
			cfg.MCPToken = stored.MCPToken
		}
		if stored.QueueCap > 0 {
			cfg.QueueCap = stored.QueueCap
		}
		cfg.UpdatedAt = stored.UpdatedAt
	}

	return cfg, nil
}

func (o *OrchestratorService) UpdateRuntimeConfig(ctx context.Context, update domain.RuntimeConfigUpdate) (domain.RuntimeConfig, error) {
	o.mu.Lock()
	repo := o.runtimeCfg
	o.mu.Unlock()

	current, err := o.GetRuntimeConfig(ctx)
	if err != nil {
		return domain.RuntimeConfig{}, err
	}

	if update.QueueCap != nil {
		if *update.QueueCap <= 0 {
			return domain.RuntimeConfig{}, fmt.Errorf("orchestrator: update runtime config: queueCap must be > 0")
		}
		current.QueueCap = *update.QueueCap
		o.WithQueueCap(current.QueueCap)
	}

	if update.RotateAPIToken {
		tok, err := generateToken(32)
		if err != nil {
			return domain.RuntimeConfig{}, fmt.Errorf("orchestrator: rotate api token: %w", err)
		}
		current.APIToken = tok
	}
	if update.RotateMCPToken {
		tok, err := generateToken(32)
		if err != nil {
			return domain.RuntimeConfig{}, fmt.Errorf("orchestrator: rotate mcp token: %w", err)
		}
		current.MCPToken = tok
	}
	if update.APIToken != nil {
		current.APIToken = *update.APIToken
	}
	if update.MCPToken != nil {
		current.MCPToken = *update.MCPToken
	}

	current.UpdatedAt = time.Now()
	if repo != nil {
		if err := repo.SaveRuntimeConfig(ctx, current); err != nil {
			return domain.RuntimeConfig{}, fmt.Errorf("orchestrator: save runtime config: %w", err)
		}
	}

	return current, nil
}

func generateToken(n int) (string, error) {
	if n <= 0 {
		return "", fmt.Errorf("token bytes must be > 0")
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// SetAISessionRepo wires the repository used to persist AI agent sessions.
func (o *OrchestratorService) SetAISessionRepo(r ports.AISessionRepository) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.aiSessionRepo = r
}

// SetAgentScanner wires the scanner used to detect running AI agent tools.
func (o *OrchestratorService) SetAgentScanner(s ports.AgentScanner) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.agentScanner = s
}

// SetDiscoveredAgentRepo wires the repository used to persist discovered agents.
func (o *OrchestratorService) SetDiscoveredAgentRepo(r discoveredAgentStore) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.agentRepo = r
}

// SetBroadcaster wires an optional EventBroadcaster for task lifecycle events.
// Call before starting the worker (before NewOrchestrator returns, or immediately after).
func (o *OrchestratorService) SetBroadcaster(b ports.EventBroadcaster) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.broadcaster = b
}

func (o *OrchestratorService) signalWorker() {
	select {
	case o.workCh <- struct{}{}:
	default:
	}
}

func (o *OrchestratorService) validateQueueAdmission(task domain.Task) error {
	o.mu.Lock()
	stopped := o.stopped
	queueCap := o.queueCap
	o.mu.Unlock()
	if stopped {
		return fmt.Errorf("orchestrator: queue task: service is stopped")
	}
	if queueCap <= 0 {
		queueCap = DefaultQueueCap
	}

	pending, err := o.repo.GetPending()
	if err != nil {
		return fmt.Errorf("orchestrator: queue task: check queue cap: %w", err)
	}
	if len(pending) >= queueCap {
		return fmt.Errorf("orchestrator: queue task: %w", ErrQueueFull)
	}

	if task.Command == domain.CommandExecute {
		existing, err := o.repo.GetByProjectPath(task.ProjectPath)
		if err != nil {
			return fmt.Errorf("orchestrator: queue task: %w", err)
		}
		hasPlan := false
		for _, existingTask := range existing {
			if existingTask.Command == domain.CommandPlan && existingTask.Status == domain.StatusCompleted {
				hasPlan = true
				break
			}
		}
		if !hasPlan {
			return fmt.Errorf("orchestrator: queue task: %w", domain.ErrNoPlan)
		}
	}

	return nil
}

// recoverStuckTasks re-queues any tasks that were in PROCESSING state when the
// previous service instance crashed. Called from NewOrchestrator before the
// worker goroutine starts, so no locking is needed on o.queue.
func (o *OrchestratorService) recoverStuckTasks() {
	pending, err := o.repo.GetPending()
	if err != nil {
		log.Printf("orchestrator: startup recovery: get pending: %v", err)
		return
	}
	requeued := 0
	hasQueued := false
	for _, t := range pending {
		if t.Status == domain.StatusQueued {
			hasQueued = true
		}
		if t.Status == domain.StatusProcessing {
			ok, err := o.repo.UpdateStatusIfCurrent(t.ID, domain.StatusProcessing, domain.StatusQueued)
			if err != nil {
				log.Printf("orchestrator: startup recovery: re-queue task %s: %v", t.ID, err)
				continue
			}
			if ok {
				requeued++
				hasQueued = true
			}
		}
	}
	if requeued > 0 {
		log.Printf("orchestrator: startup recovery: re-queued %d stuck tasks", requeued)
	}
	if hasQueued {
		o.signalWorker()
	}
}

// requeueForRetry increments the task's RetryCount, persists it with StatusQueued,
// re-adds it to the in-memory queue, and signals the worker.
// Returns true when the task was successfully re-queued; false when maxRetries is
// exhausted or the repo update fails (caller should then mark the task FAILED).
func (o *OrchestratorService) requeueForRetry(task domain.Task) bool {
	if task.RetryCount >= o.maxRetries {
		return false
	}
	task.RetryCount++
	task.Status = domain.StatusQueued
	task.UpdatedAt = time.Now()
	if err := o.repo.Update(task); err != nil {
		log.Printf("orchestrator: requeue task %s: update: %v", task.ID, err)
		return false
	}
	log.Printf("orchestrator: task %s: retry %d/%d", task.ID, task.RetryCount, o.maxRetries)
	o.signalWorker()
	o.emit(task.ID, domain.StatusQueued)
	return true
}

// runWorker is the background loop that processes QUEUED tasks sequentially.
// It blocks on workCh until a task is submitted, then drains the entire queue
// before waiting again — guaranteeing only one LLM call is ever in flight.
func (o *OrchestratorService) runWorker() {
	defer o.workerWg.Done()
	for {
		select {
		case <-o.stopCh:
			return
		case <-o.workCh:
			for {
				select {
				case <-o.stopCh:
					return
				default:
				}
				if !o.processNext() {
					break
				}
			}
		}
	}
}

// runSessionCleanup periodically marks AI sessions as disconnected when they
// have not sent a heartbeat within the stale threshold (5 × heartbeat interval = 5 min).
// It runs until stopCh is closed and shares the workerWg lifecycle.
func (o *OrchestratorService) runSessionCleanup() {
	defer o.workerWg.Done()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-o.stopCh
		cancel()
	}()
	defer cancel()
	ticker := time.NewTicker(o.cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-o.stopCh:
			return
		case <-ticker.C:
			o.mu.Lock()
			repo := o.aiSessionRepo
			b := o.broadcaster
			o.mu.Unlock()
			if repo == nil {
				continue
			}
			sessions, err := repo.ListAISessions(ctx)
			if err != nil {
				log.Printf("orchestrator: session cleanup: list: %v", err)
				continue
			}
			cutoff := time.Now().Add(-o.staleThreshold)
			for _, s := range sessions {
				if s.Status != domain.SessionStatusDisconnected && s.LastActivity.Before(cutoff) {
					if markErr := repo.UpdateAISessionStatus(ctx, s.ID, domain.SessionStatusDisconnected, s.LastActivity); markErr != nil {
						log.Printf("orchestrator: session cleanup: mark disconnected %s: %v", s.ID, markErr)
						continue
					}
					if b != nil {
						b.BroadcastAISessionEvent(domain.AISessionEvent{
							Type:        "ai_session_changed",
							AISessionID: s.ID,
							Status:      domain.SessionStatusDisconnected,
							Timestamp:   time.Now(),
						})
					}
				}
			}
			// Purge disconnected sessions older than DefaultPurgeDisconnectAge to prevent unbounded growth.
			if n, purgeErr := repo.PurgeDisconnected(ctx, DefaultPurgeDisconnectAge); purgeErr != nil {
				log.Printf("orchestrator: session cleanup: purge: %v", purgeErr)
			} else if n > 0 {
				log.Printf("orchestrator: session cleanup: purged %d stale disconnected sessions", n)
			}
		}
	}
}

func (o *OrchestratorService) runTaskWatchdog() {
	defer o.workerWg.Done()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-o.stopCh
		cancel()
	}()
	defer cancel()
	watchdogInterval := o.watchdogInterval
	if watchdogInterval <= 0 {
		watchdogInterval = time.Minute
	}
	ticker := time.NewTicker(watchdogInterval)
	defer ticker.Stop()
	for {
		select {
		case <-o.stopCh:
			return
		case <-ticker.C:
			// DefaultWatchdogStale timeout for processing tasks
			tasks, err := o.repo.GetStaleProcessing(ctx, DefaultWatchdogStale)
			if err != nil {
				log.Printf("orchestrator: task watchdog: get stale tasks: %v", err)
				continue
			}
			for _, t := range tasks {
				log.Printf("orchestrator: task watchdog: task %s stuck in PROCESSING, failing", t.ID)
				t.Status = domain.StatusFailed
				t.Logs += "\n\n[System] Task timed out while PROCESSING. Marked as FAILED by watchdog."
				if err := o.repo.Update(t); err != nil {
					log.Printf("orchestrator: task watchdog: update task %s fail: %v", t.ID, err)
					continue
				}
				o.emit(t.ID, domain.StatusFailed)
				if o.broadcaster != nil {
					o.broadcaster.Broadcast(ports.TaskEvent{
						Type: ports.EventTaskFailed, TaskID: t.ID, Status: domain.StatusFailed,
					})
				}
			}
		}
	}
}
