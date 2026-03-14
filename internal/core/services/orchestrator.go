// Package services implements the core business logic of nexusOrchestrator.
// The OrchestratorService manages a task queue, routes code-generation tasks to
// available LLM providers, and maintains per-project conversation history.
package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"

	"github.com/google/uuid"
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

// OrchestratorService implements ports.Orchestrator and drives the worker loop.
type OrchestratorService struct {
	mu          sync.Mutex
	discovery   *DiscoveryService
	fileWriter  ports.FileWriter
	repo        ports.TaskRepository
	sessionRepo ports.SessionRepository
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
	lastAgentScan      time.Time
	lastAgentScanMu    sync.Mutex
	maxRetries         int
	maxResponseTokens  int
	cleanupInterval    time.Duration
	staleThreshold     time.Duration
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
	}
	for _, opt := range opts {
		opt(svc)
	}
	svc.recoverStuckTasks()
	svc.workerWg.Add(1)
	go svc.runWorker()
	svc.workerWg.Add(1)
	go svc.runSessionCleanup()
	return svc
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
		queueCap = 50
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

// SubmitTask enqueues a new Task and returns its generated ID.
func (o *OrchestratorService) SubmitTask(task domain.Task) (string, error) {
	// Command validation
	if !task.Command.IsValid() {
		return "", fmt.Errorf("orchestrator: submit task: invalid command type %q", task.Command)
	}
	if task.Command == "" {
		task.Command = domain.CommandAuto
	}

	// Normalize ProjectPath to an absolute, cleaned path before any repo queries.
	if task.ProjectPath != "" {
		if abs, absErr := filepath.Abs(task.ProjectPath); absErr == nil {
			task.ProjectPath = filepath.Clean(abs)
		}
	}
	if err := o.validateQueueAdmission(task); err != nil {
		return "", fmt.Errorf("orchestrator: submit task: %w", err)
	}

	task.ID = uuid.NewString()
	task.Status = domain.StatusQueued
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()

	if err := o.repo.Save(task); err != nil {
		return "", fmt.Errorf("orchestrator: save task: %w", err)
	}
	o.emit(task.ID, domain.StatusQueued)
	o.signalWorker()

	return task.ID, nil
}

// GetTask retrieves a single task by ID from the repository.
// Returns domain.ErrNotFound when no task matches.
func (o *OrchestratorService) GetTask(id string) (domain.Task, error) {
	t, err := o.repo.GetByID(id)
	if err != nil {
		return domain.Task{}, fmt.Errorf("orchestrator: get task: %w", err)
	}
	return t, nil
}

// GetQueue returns all pending (QUEUED or PROCESSING) tasks.
func (o *OrchestratorService) GetQueue() ([]domain.Task, error) {
	return o.repo.GetPending()
}

// GetAllTasks returns every task regardless of status.
func (o *OrchestratorService) GetAllTasks() ([]domain.Task, error) {
	return o.repo.GetAll()
}

// GetProviders returns the liveness status of every registered LLM backend.
func (o *OrchestratorService) GetProviders() ([]ports.ProviderInfo, error) {
	return o.discovery.ListProviders(), nil
}

// CancelTask removes a QUEUED task before it is processed.
func (o *OrchestratorService) CancelTask(id string) error {
	ok, err := o.repo.UpdateStatusIfCurrent(id, domain.StatusQueued, domain.StatusCancelled)
	if err != nil {
		return fmt.Errorf("orchestrator: cancel task: %w", err)
	}
	if ok {
		o.emit(id, domain.StatusCancelled)
		return nil
	}
	task, err := o.repo.GetByID(id)
	if err != nil {
		return fmt.Errorf("orchestrator: cancel task: %w", err)
	}
	return fmt.Errorf("orchestrator: cancel task: cannot cancel task with status %s", task.Status)
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

// WithProviderFactory sets the factory used by RegisterCloudProvider to construct
// new LLM adapters from a ProviderConfig. Must be called before the first
// RegisterCloudProvider call.
func (o *OrchestratorService) WithProviderFactory(fn func(domain.ProviderConfig) (ports.LLMClient, error)) *OrchestratorService {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.providerFactory = fn
	return o
}

// RegisterCloudProvider builds a new LLM adapter from cfg using the registered
// factory and adds it to the DiscoveryService.
func (o *OrchestratorService) RegisterCloudProvider(cfg domain.ProviderConfig) error {
	o.mu.Lock()
	factory := o.providerFactory
	o.mu.Unlock()
	if factory == nil {
		return fmt.Errorf("orchestrator: no provider factory configured")
	}
	client, err := factory(cfg)
	if err != nil {
		return fmt.Errorf("orchestrator: register cloud provider: %w", err)
	}
	o.discovery.RegisterProvider(client)
	return nil
}

// RemoveProvider deregisters the named provider from DiscoveryService.
// Returns domain.ErrNotFound when no provider with that name is registered.
func (o *OrchestratorService) RemoveProvider(providerName string) error {
	if ok := o.discovery.RemoveProvider(providerName); !ok {
		return fmt.Errorf("orchestrator: remove provider: %w", domain.ErrNotFound)
	}
	return nil
}

// AddProviderConfig persists a new provider config and, when Enabled, instantiates
// and registers an adapter via the configured providerFactory.
func (o *OrchestratorService) AddProviderConfig(ctx context.Context, cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	o.mu.Lock()
	repo := o.providerConfigRepo
	factory := o.providerFactory
	o.mu.Unlock()

	if cfg.ID == "" {
		cfg.ID = uuid.NewString()
	}
	now := time.Now()
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = now
	}
	cfg.UpdatedAt = now

	if repo != nil {
		if err := repo.SaveProviderConfig(ctx, cfg); err != nil {
			return domain.ProviderConfig{}, fmt.Errorf("orchestrator: add provider config: %w", err)
		}
	}

	if cfg.Enabled && factory != nil {
		client, err := factory(cfg)
		if err != nil {
			return domain.ProviderConfig{}, fmt.Errorf("orchestrator: add provider config: build adapter: %w", err)
		}
		o.discovery.RegisterProvider(client)
	}

	return cfg, nil
}

// UpdateProviderConfig overwrites an existing provider config and refreshes its
// in-process adapter registration.
func (o *OrchestratorService) UpdateProviderConfig(ctx context.Context, cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	o.mu.Lock()
	repo := o.providerConfigRepo
	factory := o.providerFactory
	o.mu.Unlock()

	if repo == nil {
		return domain.ProviderConfig{}, fmt.Errorf("orchestrator: update provider config: no config repo configured")
	}

	old, err := repo.GetProviderConfig(ctx, cfg.ID)
	if err != nil {
		return domain.ProviderConfig{}, fmt.Errorf("orchestrator: update provider config: %w", err)
	}

	cfg.CreatedAt = old.CreatedAt
	cfg.UpdatedAt = time.Now()

	if err := repo.SaveProviderConfig(ctx, cfg); err != nil {
		return domain.ProviderConfig{}, fmt.Errorf("orchestrator: update provider config: %w", err)
	}

	// Remove old adapter registration (name may have changed).
	o.discovery.RemoveProvider(old.Name)

	if cfg.Enabled && factory != nil {
		client, err := factory(cfg)
		if err != nil {
			return domain.ProviderConfig{}, fmt.Errorf("orchestrator: update provider config: build adapter: %w", err)
		}
		o.discovery.RegisterProvider(client)
	}

	return cfg, nil
}

// RemoveProviderConfig deletes the persisted provider config identified by id
// and deregisters its adapter.
func (o *OrchestratorService) RemoveProviderConfig(ctx context.Context, id string) error {
	o.mu.Lock()
	repo := o.providerConfigRepo
	o.mu.Unlock()

	if repo == nil {
		return fmt.Errorf("orchestrator: remove provider config: no config repo configured")
	}

	cfg, err := repo.GetProviderConfig(ctx, id)
	if err != nil {
		return fmt.Errorf("orchestrator: remove provider config: %w", err)
	}

	if err := repo.DeleteProviderConfig(ctx, id); err != nil {
		return fmt.Errorf("orchestrator: remove provider config: %w", err)
	}

	o.discovery.RemoveProvider(cfg.Name)
	return nil
}

// ListProviderConfigs returns all persisted provider configuration records.
// Returns an empty slice when no repository is configured.
func (o *OrchestratorService) ListProviderConfigs(ctx context.Context) ([]domain.ProviderConfig, error) {
	o.mu.Lock()
	repo := o.providerConfigRepo
	o.mu.Unlock()

	if repo == nil {
		return []domain.ProviderConfig{}, nil
	}

	cfgs, err := repo.ListProviderConfigs(ctx)
	if err != nil {
		return nil, fmt.Errorf("orchestrator: list provider configs: %w", err)
	}
	if cfgs == nil {
		return []domain.ProviderConfig{}, nil
	}
	return cfgs, nil
}

// GetProviderModels returns the model catalogue of the named provider.
// Returns domain.ErrNotFound when no provider with that name is registered.
func (o *OrchestratorService) GetProviderModels(providerName string) ([]string, error) {
	client, ok := o.discovery.GetClientByName(providerName)
	if !ok {
		return nil, fmt.Errorf("orchestrator: get provider models: %w", domain.ErrNotFound)
	}
	models, err := client.GetAvailableModels()
	if err != nil {
		return nil, fmt.Errorf("orchestrator: get provider models: %w", err)
	}
	return models, nil
}

// GetDiscoveredProviders returns auto-detected AI tools from the local system
// that have NOT yet been promoted to active/configured providers.
func (o *OrchestratorService) GetDiscoveredProviders() ([]domain.DiscoveredProvider, error) {
	o.scanMu.RLock()
	defer o.scanMu.RUnlock()
	if o.lastScan == nil {
		return []domain.DiscoveredProvider{}, nil
	}
	result := make([]domain.DiscoveredProvider, len(o.lastScan))
	copy(result, o.lastScan)
	return result, nil
}

// TriggerScan requests an immediate re-scan of the local system for AI providers.
func (o *OrchestratorService) TriggerScan(ctx context.Context) ([]domain.DiscoveredProvider, error) {
	o.mu.Lock()
	scanner := o.scanner
	o.mu.Unlock()

	if scanner == nil {
		return []domain.DiscoveredProvider{}, fmt.Errorf("orchestrator: trigger scan: scanner not configured")
	}

	results, err := scanner.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("orchestrator: trigger scan: %w", err)
	}

	o.scanMu.Lock()
	o.lastScan = results
	o.scanMu.Unlock()

	o.discovery.InvalidateHealthCache()

	return results, nil
}

// PromoteProvider converts a discovered provider into an active registered backend.
func (o *OrchestratorService) PromoteProvider(ctx context.Context, discoveredID string) error {
	o.scanMu.RLock()
	var found *domain.DiscoveredProvider
	for i := range o.lastScan {
		if o.lastScan[i].ID == discoveredID {
			discovered := o.lastScan[i]
			found = &discovered
			break
		}
	}
	o.scanMu.RUnlock()

	if found == nil {
		return fmt.Errorf("orchestrator: promote provider: %w", domain.ErrNotFound)
	}
	if found.Status != domain.DiscoveryStatusReachable {
		return fmt.Errorf("orchestrator: promote provider: provider %q is not reachable (status: %s)", found.Name, found.Status)
	}
	if found.BaseURL == "" {
		return fmt.Errorf("orchestrator: promote provider: discovered provider has no base URL")
	}

	cfg := domain.ProviderConfig{
		ID:      found.ID,
		Name:    found.Name,
		Kind:    found.Kind,
		BaseURL: found.BaseURL,
		Enabled: true,
	}

	if o.providerConfigRepo != nil {
		if _, err := o.AddProviderConfig(ctx, cfg); err == nil {
			return nil
		} else {
			log.Printf("orchestrator: promote provider: persist: %v", err)
		}
	}

	if err := o.RegisterCloudProvider(cfg); err != nil {
		return fmt.Errorf("orchestrator: promote provider: %w", err)
	}

	return nil
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
	o.agentScanner = s
}

// SetDiscoveredAgentRepo wires the repository used to persist discovered agents.
func (o *OrchestratorService) SetDiscoveredAgentRepo(r discoveredAgentStore) {
	o.agentRepo = r
}

// CreateDraft creates a task with StatusDraft without entering the execution queue.
func (o *OrchestratorService) CreateDraft(task domain.Task) (string, error) {
	if task.Instruction == "" {
		return "", fmt.Errorf("orchestrator: create draft: instruction is required")
	}
	if task.ProjectPath == "" {
		return "", fmt.Errorf("orchestrator: create draft: project path is required")
	}
	// Normalize ProjectPath to an absolute, cleaned path.
	if abs, absErr := filepath.Abs(task.ProjectPath); absErr == nil {
		task.ProjectPath = filepath.Clean(abs)
	}
	task.ID = uuid.NewString()
	task.Status = domain.StatusDraft
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	if task.Priority == 0 {
		task.Priority = 2
	}
	if err := o.repo.Save(task); err != nil {
		return "", fmt.Errorf("orchestrator: create draft: %w", err)
	}
	o.emit(task.ID, domain.StatusDraft)
	return task.ID, nil
}

// GetBacklog returns DRAFT and BACKLOG tasks for the given project.
// If projectPath is empty, returns all DRAFT and BACKLOG tasks across all projects.
func (o *OrchestratorService) GetBacklog(projectPath string) ([]domain.Task, error) {
	if projectPath == "" {
		all, err := o.repo.GetAll()
		if err != nil {
			return nil, fmt.Errorf("orchestrator: get backlog: %w", err)
		}
		backlog := []domain.Task{}
		for _, t := range all {
			if t.Status == domain.StatusDraft || t.Status == domain.StatusBacklog {
				backlog = append(backlog, t)
			}
		}
		return backlog, nil
	}
	tasks, err := o.repo.GetByProjectPathAndStatus(projectPath, domain.StatusDraft, domain.StatusBacklog)
	if err != nil {
		return nil, fmt.Errorf("orchestrator: get backlog: %w", err)
	}
	return tasks, nil
}

// PromoteTask transitions a DRAFT or BACKLOG task to QUEUED and enqueues it.
func (o *OrchestratorService) PromoteTask(id string) error {
	task, err := o.repo.GetByID(id)
	if err != nil {
		return fmt.Errorf("orchestrator: promote task: %w", err)
	}
	if task.Status != domain.StatusDraft && task.Status != domain.StatusBacklog {
		return fmt.Errorf("orchestrator: promote task: cannot promote task with status %s", task.Status)
	}
	if err := o.validateQueueAdmission(task); err != nil {
		return fmt.Errorf("orchestrator: promote task: %w", err)
	}
	ok, err := o.repo.UpdateStatusIfCurrent(id, task.Status, domain.StatusQueued)
	if err != nil {
		return fmt.Errorf("orchestrator: promote task: %w", err)
	}
	if !ok {
		return fmt.Errorf("orchestrator: promote task: task state changed during promotion")
	}
	o.signalWorker()
	o.emit(task.ID, domain.StatusQueued)
	return nil
}

// UpdateTask merges non-zero fields from updates into the stored task and persists.
// Status transitions are only allowed for non-executing states (DRAFT, BACKLOG, QUEUED).
func (o *OrchestratorService) UpdateTask(id string, updates domain.Task) (domain.Task, error) {
	task, err := o.repo.GetByID(id)
	if err != nil {
		return domain.Task{}, fmt.Errorf("orchestrator: update task: %w", err)
	}
	if updates.Instruction != "" {
		task.Instruction = updates.Instruction
	}
	if updates.TargetFile != "" {
		task.TargetFile = updates.TargetFile
	}
	if updates.ProviderName != "" {
		task.ProviderName = updates.ProviderName
	}
	if updates.ModelID != "" {
		task.ModelID = updates.ModelID
	}
	if updates.ProviderHint != "" {
		task.ProviderHint = updates.ProviderHint
	}
	if updates.Priority != 0 {
		task.Priority = updates.Priority
	}
	if updates.Tags != nil {
		task.Tags = updates.Tags
	}
	if updates.Status == domain.StatusQueued {
		return domain.Task{}, fmt.Errorf("orchestrator: update task: use promote task to transition into %s", domain.StatusQueued)
	}
	if updates.Status != "" &&
		(updates.Status == domain.StatusDraft ||
			updates.Status == domain.StatusBacklog) {
		task.Status = updates.Status
	}
	task.UpdatedAt = time.Now()
	if err := o.repo.Update(task); err != nil {
		return domain.Task{}, fmt.Errorf("orchestrator: update task: %w", err)
	}
	o.emit(task.ID, task.Status)
	return task, nil
}

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

// ClaimTask assigns a QUEUED task to the given AI session, transitioning it to PROCESSING.
// Returns domain.ErrNotFound if the task or session does not exist.
func (o *OrchestratorService) ClaimTask(ctx context.Context, taskID string, sessionID string) (domain.Task, error) {
	o.mu.Lock()
	repo := o.repo
	sessionRepo := o.aiSessionRepo
	b := o.broadcaster
	o.mu.Unlock()

	if sessionRepo == nil {
		return domain.Task{}, fmt.Errorf("orchestrator: claim task: no session repo configured")
	}

	// Verify session exists and is active.
	sess, err := sessionRepo.GetAISessionByID(ctx, sessionID)
	if err != nil {
		return domain.Task{}, fmt.Errorf("orchestrator: claim task: session lookup: %w", err)
	}
	if sess.Status.IsTerminal() {
		return domain.Task{}, fmt.Errorf("orchestrator: claim task: session %s is %s", sessionID, sess.Status)
	}

	// Atomically transition QUEUED → PROCESSING.
	ok, err := repo.UpdateStatusIfCurrent(taskID, domain.StatusQueued, domain.StatusProcessing)
	if err != nil {
		return domain.Task{}, fmt.Errorf("orchestrator: claim task: %w", err)
	}
	if !ok {
		return domain.Task{}, fmt.Errorf("orchestrator: claim task: task %s is not QUEUED", taskID)
	}

	// Bind the session to the task.
	task, err := repo.GetByID(taskID)
	if err != nil {
		return domain.Task{}, fmt.Errorf("orchestrator: claim task: get task: %w", err)
	}
	task.AISessionID = sessionID
	if err := repo.Update(task); err != nil {
		return domain.Task{}, fmt.Errorf("orchestrator: claim task: update session binding: %w", err)
	}

	// Record the task on the session's routed list.
	if err := sessionRepo.AppendRoutedTaskID(ctx, sessionID, taskID); err != nil {
		log.Printf("orchestrator: claim task: append routed task id: %v", err)
	}

	if b != nil {
		b.Broadcast(ports.TaskEvent{
			Type:   ports.EventTaskProcessing,
			TaskID: taskID,
			Status: domain.StatusProcessing,
		})
	}
	return task, nil
}

// UpdateTaskStatus allows an external AI session to report task completion or failure.
// Only the session that claimed the task (matching AISessionID) may update its status.
func (o *OrchestratorService) UpdateTaskStatus(ctx context.Context, taskID string, sessionID string, status domain.TaskStatus, logs string) (domain.Task, error) {
	o.mu.Lock()
	repo := o.repo
	b := o.broadcaster
	o.mu.Unlock()

	// Only allow COMPLETED or FAILED from external agents.
	if status != domain.StatusCompleted && status != domain.StatusFailed {
		return domain.Task{}, fmt.Errorf("orchestrator: update task status: invalid target status %s (must be COMPLETED or FAILED)", status)
	}

	task, err := repo.GetByID(taskID)
	if err != nil {
		return domain.Task{}, fmt.Errorf("orchestrator: update task status: %w", err)
	}

	// Ownership check.
	if task.AISessionID != sessionID {
		return domain.Task{}, fmt.Errorf("orchestrator: update task status: session %s does not own task %s", sessionID, taskID)
	}

	// Only PROCESSING tasks can be completed/failed.
	if task.Status != domain.StatusProcessing {
		return domain.Task{}, fmt.Errorf("orchestrator: update task status: task %s is %s, not PROCESSING", taskID, task.Status)
	}

	if err := repo.UpdateStatus(taskID, status); err != nil {
		return domain.Task{}, fmt.Errorf("orchestrator: update task status: %w", err)
	}
	if logs != "" {
		if err := repo.UpdateLogs(taskID, logs); err != nil {
			log.Printf("orchestrator: update task status: update logs: %v", err)
		}
	}

	task.Status = status
	task.Logs = logs

	if b != nil {
		var evtType ports.EventType
		if status == domain.StatusCompleted {
			evtType = ports.EventTaskCompleted
		} else {
			evtType = ports.EventTaskFailed
		}
		b.Broadcast(ports.TaskEvent{
			Type:   evtType,
			TaskID: taskID,
			Status: status,
		})
	}
	return task, nil
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

// SetBroadcaster wires an optional EventBroadcaster for task lifecycle events.
// Call before starting the worker (before NewOrchestrator returns, or immediately after).
func (o *OrchestratorService) SetBroadcaster(b ports.EventBroadcaster) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.broadcaster = b
}

// emit publishes a TaskEvent if a broadcaster is configured.
// It acquires the mutex only to read the broadcaster pointer, then releases it
// before calling Broadcast so the hub's own lock is never nested under o.mu.
func (o *OrchestratorService) emit(taskID string, status domain.TaskStatus) {
	o.mu.Lock()
	b := o.broadcaster
	o.mu.Unlock()
	if b == nil {
		return
	}
	b.Broadcast(ports.TaskEvent{
		Type:   statusEventType(status),
		TaskID: taskID,
		Status: status,
	})
}

// statusEventType maps a TaskStatus to its corresponding EventType.
var statusEventMap = map[domain.TaskStatus]ports.EventType{
	domain.StatusQueued:     ports.EventTaskQueued,
	domain.StatusProcessing: ports.EventTaskProcessing,
	domain.StatusCompleted:  ports.EventTaskCompleted,
	domain.StatusFailed:     ports.EventTaskFailed,
	domain.StatusCancelled:  ports.EventTaskCancelled,
	domain.StatusTooLarge:   ports.EventTaskTooLarge,
	domain.StatusNoProvider: ports.EventTaskNoProvider,
	domain.StatusDraft:      ports.EventTaskDraft,
	domain.StatusBacklog:    ports.EventTaskBacklog,
}

func statusEventType(s domain.TaskStatus) ports.EventType {
	return statusEventMap[s]
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
				if !o.processNext() {
					break
				}
				select {
				case <-o.stopCh:
					return
				default:
				}
				o.processNext()
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
			// Purge disconnected sessions older than 2 hours to prevent unbounded growth.
			if n, purgeErr := repo.PurgeDisconnected(ctx, 2*time.Hour); purgeErr != nil {
				log.Printf("orchestrator: session cleanup: purge: %v", purgeErr)
			} else if n > 0 {
				log.Printf("orchestrator: session cleanup: purged %d stale disconnected sessions", n)
			}
		}
	}
}

func (o *OrchestratorService) processNext() bool {
	task, err := o.repo.ClaimNextQueued()
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return false
		}
		log.Printf("orchestrator: claim next queued: %v", err)
		return false
	}

	llm, err := o.selectProviderForTask(task)
	if err != nil {
		return true
	}
	o.emit(task.ID, domain.StatusProcessing)

	prompt, sessionHistory, err := o.buildChatContext(task, llm)
	if err != nil {
		return true
	}

	code, err := o.executeGeneration(task, llm, prompt, sessionHistory)
	if err != nil {
		return true
	}

	o.writeTaskOutput(task, code, llm.ProviderName())
	return true
}

// selectProviderForTask resolves the LLM client for the task by provider name or
// by model/hint lookup. On failure it sets StatusNoProvider, logs the reason, and emits the event.
func (o *OrchestratorService) selectProviderForTask(task domain.Task) (ports.LLMClient, error) {
	if task.ProviderName != "" {
		client, ok := o.discovery.GetClientByName(task.ProviderName)
		if !ok {
			logMsg := fmt.Sprintf("provider '%s' not found or not active", task.ProviderName)
			log.Printf("orchestrator: no provider for task %s: %s", task.ID, logMsg)
			if err := o.repo.UpdateLogs(task.ID, logMsg); err != nil {
				log.Printf("orchestrator: update logs for task %s: %v", task.ID, err)
			}
			if err := o.repo.UpdateStatus(task.ID, domain.StatusNoProvider); err != nil {
				log.Printf("orchestrator: update status for task %s: %v", task.ID, err)
			}
			o.emit(task.ID, domain.StatusNoProvider)
			return nil, fmt.Errorf("provider %q not found or not active", task.ProviderName)
		}
		return client, nil
	}
	llm, err := o.discovery.FindForModel(task.ModelID, task.ProviderHint)
	if err != nil {
		log.Printf("orchestrator: no provider for task %s (model=%q): %v", task.ID, task.ModelID, err)
		if err2 := o.repo.UpdateLogs(task.ID, err.Error()); err2 != nil {
			log.Printf("orchestrator: update logs for task %s: %v", task.ID, err2)
		}
		if err2 := o.repo.UpdateStatus(task.ID, domain.StatusNoProvider); err2 != nil {
			log.Printf("orchestrator: update status for task %s: %v", task.ID, err2)
		}
		o.emit(task.ID, domain.StatusNoProvider)
		return nil, err
	}
	return llm, nil
}

// buildChatContext constructs the prompt with optional context file content prepended,
// loads session history, and guards against context-window overflow.
// On overflow it sets StatusTooLarge, logs the reason, and emits the event.
func (o *OrchestratorService) buildChatContext(task domain.Task, llm ports.LLMClient) (string, []domain.Message, error) {
	// Build the prompt with optional context files.
	prompt := task.Instruction
	if len(task.ContextFiles) > 0 && o.fileWriter != nil {
		ctx, err := o.fileWriter.ReadContextFiles(task.ProjectPath, task.ContextFiles)
		if err != nil {
			log.Printf("orchestrator: read context for task %s: %v", task.ID, err)
		} else if strings.TrimSpace(ctx) != "" {
			prompt = ctx + "\n\n" + prompt
		}
	}

	// Load session history once — reused for both the pre-flight token check and
	// the Chat call to avoid double GetByProjectPath.
	var sessionHistory []domain.Message
	if o.sessionRepo != nil {
		sess, err := o.sessionRepo.GetByProjectPath(task.ProjectPath)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			log.Printf("orchestrator: load session for task %s: %v", task.ID, err)
		}
		sessionHistory = sess.Messages
	}

	// Pre-flight: guard against context-window overflow before spending LLM time.
	if limit := llm.ContextLimit(); limit > 0 {
		estHistory := make([]domain.Message, len(sessionHistory)+1)
		copy(estHistory, sessionHistory)
		estHistory[len(sessionHistory)] = domain.Message{Role: domain.RoleUser, Content: prompt}
		if estimated := estimateTokens(estHistory); estimated > limit-o.maxResponseTokens {
			logEntry := fmt.Sprintf(
				"context too large: ~%d tokens estimated, model limit is %d (headroom %d) — shorten the instruction or reduce context files",
				estimated, limit, o.maxResponseTokens,
			)
			log.Printf("orchestrator: task %s: %s", task.ID, logEntry)
			if err := o.repo.UpdateLogs(task.ID, logEntry); err != nil {
				log.Printf("orchestrator: update logs for task %s: %v", task.ID, err)
			}
			if err := o.repo.UpdateStatus(task.ID, domain.StatusTooLarge); err != nil {
				log.Printf("orchestrator: update status for task %s: %v", task.ID, err)
			}
			o.emit(task.ID, domain.StatusTooLarge)
			return "", nil, fmt.Errorf("context too large")
		}
	}

	return prompt, sessionHistory, nil
}

// executeGeneration dispatches to Chat (when sessionRepo is set) or GenerateCode,
// with retry on transient failures. On fatal failure it persists StatusFailed.
// On success with a sessionRepo it appends the user and assistant messages to the session.
func (o *OrchestratorService) executeGeneration(task domain.Task, llm ports.LLMClient, prompt string, sessionHistory []domain.Message) (string, error) {
	if o.sessionRepo != nil {
		// Build the chat history using the already-loaded session (no second DB call).
		userMsg := domain.Message{Role: domain.RoleUser, Content: prompt, CreatedAt: time.Now()}
		history := append(append([]domain.Message(nil), sessionHistory...), userMsg)
		code, err := llm.Chat(history)
		if err != nil {
			logEntry := fmt.Sprintf("failed via %s: %v", llm.ProviderName(), err)
			log.Printf("orchestrator: chat for task %s: %v", task.ID, err)
			if o.requeueForRetry(task) {
				return "", err
			}
			if err2 := o.repo.UpdateLogs(task.ID, logEntry); err2 != nil {
				log.Printf("orchestrator: update logs for task %s: %v", task.ID, err2)
			}
			if err2 := o.repo.UpdateStatus(task.ID, domain.StatusFailed); err2 != nil {
				log.Printf("orchestrator: update status for task %s: %v", task.ID, err2)
			}
			o.emit(task.ID, domain.StatusFailed)
			return "", err
		}
		// Only persist messages after a successful response.
		assistantMsg := domain.Message{Role: domain.RoleAssistant, Content: code, CreatedAt: time.Now()}
		if err := o.sessionRepo.AppendMessage(task.ProjectPath, userMsg); err != nil {
			log.Printf("orchestrator: append user message for task %s: %v", task.ID, err)
		}
		if err := o.sessionRepo.AppendMessage(task.ProjectPath, assistantMsg); err != nil {
			log.Printf("orchestrator: append assistant message for task %s: %v", task.ID, err)
		}
		return code, nil
	}

	code, err := llm.GenerateCode(prompt)
	if err != nil {
		logEntry := fmt.Sprintf("failed via %s: %v", llm.ProviderName(), err)
		log.Printf("orchestrator: generate code for task %s: %v", task.ID, err)
		if o.requeueForRetry(task) {
			return "", err
		}
		if err2 := o.repo.UpdateLogs(task.ID, logEntry); err2 != nil {
			log.Printf("orchestrator: update logs for task %s: %v", task.ID, err2)
		}
		if err2 := o.repo.UpdateStatus(task.ID, domain.StatusFailed); err2 != nil {
			log.Printf("orchestrator: update status for task %s: %v", task.ID, err2)
		}
		o.emit(task.ID, domain.StatusFailed)
		return "", err
	}
	return code, nil
}

// writeTaskOutput optionally writes the generated code to disk and marks the task
// as COMPLETED. On write failure it persists StatusFailed and emits the event.
func (o *OrchestratorService) writeTaskOutput(task domain.Task, code string, providerName string) {
	if o.fileWriter != nil && task.TargetFile != "" {
		if err := o.fileWriter.WriteCodeToFile(task.ProjectPath, task.TargetFile, extractCode(code)); err != nil {
			logEntry := fmt.Sprintf("failed writing output via %s: %v", providerName, err)
			log.Printf("orchestrator: write file for task %s: %v", task.ID, err)
			if err2 := o.repo.UpdateLogs(task.ID, logEntry); err2 != nil {
				log.Printf("orchestrator: update logs for task %s: %v", task.ID, err2)
			}
			if err2 := o.repo.UpdateStatus(task.ID, domain.StatusFailed); err2 != nil {
				log.Printf("orchestrator: update status for task %s: %v", task.ID, err2)
			}
			o.emit(task.ID, domain.StatusFailed)
			return
		}
	}

	logEntry := fmt.Sprintf("completed via %s at %s", providerName, time.Now().UTC().Format(time.RFC3339))
	if err := o.repo.UpdateLogs(task.ID, logEntry); err != nil {
		log.Printf("orchestrator: update logs for task %s: %v", task.ID, err)
	}
	if err := o.repo.UpdateStatus(task.ID, domain.StatusCompleted); err != nil {
		log.Printf("orchestrator: update status for task %s: %v", task.ID, err)
	}
	o.emit(task.ID, domain.StatusCompleted)
	log.Printf("orchestrator: task %s completed via %s", task.ID, providerName)
}

// extractCode strips the first markdown code fence from s, returning the raw
// source within. If no fence is found, s is returned unchanged.
func extractCode(s string) string {
	lines := strings.Split(s, "\n")
	start := -1
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			start = i
			break
		}
	}
	if start == -1 {
		return s
	}
	end := -1
	for i := start + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "```" {
			end = i
			break
		}
	}
	if end == -1 {
		return strings.Join(lines[start+1:], "\n")
	}
	return strings.Join(lines[start+1:end], "\n")
}

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
	return delegationInstruction(session, now), nil
}

func delegationInstruction(session domain.AISession, now time.Time) string {
	return fmt.Sprintf(`You are now operating under nexusOrchestrator coordination.

YOUR NEW WORKFLOW:
1. Check the nexus queue:  GET http://127.0.0.1:63987/api/tasks
2. Claim the next QUEUED task:
   POST http://127.0.0.1:63987/api/tasks/{id}/claim
   Body: {"sessionId": "%s"}
3. Execute the task's instruction for %s.
4. Report completion:
   PUT http://127.0.0.1:63987/api/tasks/{id}/status
   Body: {"sessionId": "%s", "status": "COMPLETED", "logs": "<summary>"}
5. Repeat from step 1.  If queue is empty, wait 30 s and check again.

Do NOT take autonomous actions outside the nexus task queue while this delegation is active.

Nexus session ID: %s
Project:          %s
Timestamp:        %s`,
		session.ID, session.ProjectPath,
		session.ID,
		session.ID, session.ProjectPath,
		now.UTC().Format(time.RFC3339))
}

// estimateTokens approximates the total token count for a message slice using
// the widely-accepted heuristic of 4 characters per token, plus 4 overhead
// tokens per message (role + chat-formatting separators).
// It deliberately over-estimates slightly to stay safely within the model's
// context window.
func estimateTokens(messages []domain.Message) int {
	total := 0
	for _, m := range messages {
		total += (len(m.Content)+3)/4 + 4
	}
	return total
}
