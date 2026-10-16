// Package services implements the core business logic of nexusOrchestrator.
// The OrchestratorService manages a task queue, routes code-generation tasks to
// available LLM providers, and maintains per-project conversation history.
package services

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

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
// It aliases domain.ErrQueueFull so adapters can match it without importing this package.
var ErrQueueFull = domain.ErrQueueFull

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

// WithUsageRepo sets the usage ledger: after every successful model call that reported its token
// usage, one record is stored (see GetUsage).
func WithUsageRepo(r ports.UsageRepository) Option {
	return func(o *OrchestratorService) { o.usageRepo = r }
}

// WithPlanFileRepo sets the repository used to persist discovered plan files.
func WithPlanFileRepo(r ports.DiscoveredPlanFileRepo) Option {
	return func(o *OrchestratorService) { o.planFileRepo = r }
}

// WithWatchdogInterval sets how often the task watchdog checks for stale tasks. Default: 1 minute.
func WithWatchdogInterval(d time.Duration) Option {
	return func(s *OrchestratorService) { s.watchdogInterval = d }
}

// WithCommandRunner sets the command runner used for post-execution verification gates.
func WithCommandRunner(runner ports.CommandRunner) Option {
	return func(s *OrchestratorService) { s.commandRunner = runner }
}

// runHandle lets CancelTask and Stop abort the generation of a task in flight.
type runHandle struct {
	ctx           context.Context
	cancel        context.CancelFunc
	userCancelled atomic.Bool // true when CancelTask (not shutdown) caused the abort
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
	tapMu       sync.Mutex             // guards taps
	taps        map[chan ports.TaskEvent]struct{}
	workCh      chan struct{} // notified when a task is enqueued; capacity 1
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
	commandRunner      ports.CommandRunner
	usageRepo          ports.UsageRepository // optional: the usage ledger
	brain              ports.BrainService    // optional: project briefing for prompts and task memory
	lastAgentScan      time.Time
	lastAgentScanMu    sync.Mutex
	signalFn           func(pid int, force bool) error // test seam; nil = real os signals
	baseCtx            context.Context                 // cancelled by Stop: aborts every in-flight generation
	baseCancel         context.CancelFunc
	runMu              sync.Mutex
	running            map[string]*runHandle // tasks being executed by this engine, by task ID
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
//
// NewOrchestrator panics if discovery, repo or writer is nil: these are wiring
// errors in the composition root, not recoverable runtime conditions.
func NewOrchestrator(
	discovery *DiscoveryService,
	repo ports.TaskRepository,
	writer ports.FileWriter,
	sessionRepo ports.SessionRepository,
	opts ...Option,
) *OrchestratorService {
	if discovery == nil {
		panic("orchestrator: NewOrchestrator: discovery is required")
	}
	if repo == nil {
		panic("orchestrator: NewOrchestrator: repo is required")
	}
	if writer == nil {
		panic("orchestrator: NewOrchestrator: writer is required")
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
		running:           make(map[string]*runHandle),
	}
	svc.baseCtx, svc.baseCancel = context.WithCancel(context.Background())
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
	return svc
}

// Stop signals the worker goroutine to exit and waits for it to finish.
// It is safe to close the backing repository only after Stop returns.
func (o *OrchestratorService) Stop() {
	o.stopOnce.Do(func() {
		o.mu.Lock()
		o.stopped = true
		o.mu.Unlock()
		// Abort in-flight generations first: without this Stop would wait out a
		// provider request that can legitimately run for minutes.
		o.baseCancel()
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

// dbCtx is the context for repository calls made outside a caller-supplied
// one. It keeps the service's values but not its cancellation: a status
// transition that has begun (for example resetting an interrupted task to
// QUEUED while Stop is cancelling baseCtx) must still reach the database.
func (o *OrchestratorService) dbCtx() context.Context {
	return context.WithoutCancel(o.baseCtx)
}
