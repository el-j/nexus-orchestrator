package services

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"

	"github.com/google/uuid"
)

// OrchestratorService implements ports.Orchestrator and drives the worker loop.
type OrchestratorService struct {
	mu          sync.Mutex
	queue       []domain.Task
	discovery   *DiscoveryService
	fileWriter  ports.FileWriter
	repo        ports.TaskRepository
	sessionRepo ports.SessionRepository
	broadcaster ports.EventBroadcaster // optional; nil = no event publishing
	workCh      chan struct{} // notified when a task is enqueued; capacity 1
	stopCh      chan struct{}
	stopped     bool
	stopOnce    sync.Once
	// providerFactory builds a concrete LLMClient from a ProviderConfig.
	// Injected by entry points to keep service layer free of adapter imports.
	providerFactory func(domain.ProviderConfig) (ports.LLMClient, error)
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
) *OrchestratorService {
	svc := &OrchestratorService{
		discovery:   discovery,
		repo:        repo,
		fileWriter:  writer,
		sessionRepo: sessionRepo,
		workCh:      make(chan struct{}, 1),
		stopCh:      make(chan struct{}),
	}
	go svc.runWorker()
	return svc
}

// SubmitTask enqueues a new Task and returns its generated ID.
func (o *OrchestratorService) SubmitTask(task domain.Task) (string, error) {
	o.mu.Lock()
	if o.stopped {
		o.mu.Unlock()
		return "", fmt.Errorf("orchestrator: submit task: service is stopped")
	}
	o.mu.Unlock()

	task.ID = uuid.NewString()
	task.Status = domain.StatusQueued
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()

	if err := o.repo.Save(task); err != nil {
		return "", fmt.Errorf("orchestrator: save task: %w", err)
	}
	o.emit(task.ID, domain.StatusQueued)

	o.mu.Lock()
	o.queue = append(o.queue, task)
	o.mu.Unlock()

	// Wake the worker without blocking if it is already awake.
	select {
	case o.workCh <- struct{}{}:
	default:
	}

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

// GetProviders returns the liveness status of every registered LLM backend.
func (o *OrchestratorService) GetProviders() ([]ports.ProviderInfo, error) {
	return o.discovery.ListProviders(), nil
}

// CancelTask removes a QUEUED task before it is processed.
func (o *OrchestratorService) CancelTask(id string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	for i, t := range o.queue {
		if t.ID == id {
			o.queue = append(o.queue[:i], o.queue[i+1:]...)
			return o.repo.UpdateStatus(id, domain.StatusCancelled)
		}
	}
	return fmt.Errorf("orchestrator: cancel task: %w", domain.ErrNotFound)
}

// Stop signals the worker goroutine to exit.
func (o *OrchestratorService) Stop() {
	o.stopOnce.Do(func() {
		o.mu.Lock()
		o.stopped = true
		o.mu.Unlock()
		close(o.stopCh)
	})
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
}

func statusEventType(s domain.TaskStatus) ports.EventType {
	return statusEventMap[s]
}

// runWorker is the background loop that processes QUEUED tasks sequentially.
// It blocks on workCh until a task is submitted, then drains the entire queue
// before waiting again — guaranteeing only one LLM call is ever in flight.
func (o *OrchestratorService) runWorker() {
	for {
		select {
		case <-o.stopCh:
			return
		case <-o.workCh:
			for {
				o.mu.Lock()
				empty := len(o.queue) == 0
				o.mu.Unlock()
				if empty {
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

func (o *OrchestratorService) processNext() {
	o.mu.Lock()
	if len(o.queue) == 0 {
		o.mu.Unlock()
		return
	}
	task := o.queue[0]
	o.queue = o.queue[1:]
	o.mu.Unlock()

	llm, err := o.discovery.FindForModel(task.ModelID, task.ProviderHint)
	if err != nil {
		log.Printf("orchestrator: no provider for task %s (model=%q): %v", task.ID, task.ModelID, err)
		if err := o.repo.UpdateLogs(task.ID, err.Error()); err != nil {
			log.Printf("orchestrator: update logs for task %s: %v", task.ID, err)
		}
		if err := o.repo.UpdateStatus(task.ID, domain.StatusNoProvider); err != nil {
			log.Printf("orchestrator: update status for task %s: %v", task.ID, err)
		}
		o.emit(task.ID, domain.StatusNoProvider)
		return
	}

	if err := o.repo.UpdateStatus(task.ID, domain.StatusProcessing); err != nil {
		log.Printf("orchestrator: update status for task %s: %v", task.ID, err)
	}
	o.emit(task.ID, domain.StatusProcessing)

	// Build the prompt with optional context files
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
	// maxResponseTokens reserves budget for the assistant reply.
	const maxResponseTokens = 512
	if limit := llm.ContextLimit(); limit > 0 {
		estHistory := make([]domain.Message, len(sessionHistory)+1)
		copy(estHistory, sessionHistory)
		estHistory[len(sessionHistory)] = domain.Message{Role: domain.RoleUser, Content: prompt}
		if estimated := estimateTokens(estHistory); estimated > limit-maxResponseTokens {
			logEntry := fmt.Sprintf(
				"context too large: ~%d tokens estimated, model limit is %d (headroom %d) — shorten the instruction or reduce context files",
				estimated, limit, maxResponseTokens,
			)
			log.Printf("orchestrator: task %s: %s", task.ID, logEntry)
			if err := o.repo.UpdateLogs(task.ID, logEntry); err != nil {
				log.Printf("orchestrator: update logs for task %s: %v", task.ID, err)
			}
			if err := o.repo.UpdateStatus(task.ID, domain.StatusTooLarge); err != nil {
				log.Printf("orchestrator: update status for task %s: %v", task.ID, err)
			}
			o.emit(task.ID, domain.StatusTooLarge)
			return
		}
	}

	var code string
	if o.sessionRepo != nil {
		// Build the chat history using the already-loaded session (no second DB call).
		userMsg := domain.Message{Role: domain.RoleUser, Content: prompt, CreatedAt: time.Now()}
		history := append(append([]domain.Message(nil), sessionHistory...), userMsg)
		var err error
		code, err = llm.Chat(history)
		if err != nil {
			logEntry := fmt.Sprintf("failed via %s: %v", llm.ProviderName(), err)
			log.Printf("orchestrator: chat for task %s: %v", task.ID, err)
			if err := o.repo.UpdateLogs(task.ID, logEntry); err != nil {
				log.Printf("orchestrator: update logs for task %s: %v", task.ID, err)
			}
			if err := o.repo.UpdateStatus(task.ID, domain.StatusFailed); err != nil {
				log.Printf("orchestrator: update status for task %s: %v", task.ID, err)
			}
			o.emit(task.ID, domain.StatusFailed)
			return
		}
		// Only persist messages after a successful response.
		assistantMsg := domain.Message{Role: domain.RoleAssistant, Content: code, CreatedAt: time.Now()}
		if err := o.sessionRepo.AppendMessage(task.ProjectPath, userMsg); err != nil {
			log.Printf("orchestrator: append user message for task %s: %v", task.ID, err)
		}
		if err := o.sessionRepo.AppendMessage(task.ProjectPath, assistantMsg); err != nil {
			log.Printf("orchestrator: append assistant message for task %s: %v", task.ID, err)
		}
	} else {
		var err error
		code, err = llm.GenerateCode(prompt)
		if err != nil {
			logEntry := fmt.Sprintf("failed via %s: %v", llm.ProviderName(), err)
			log.Printf("orchestrator: generate code for task %s: %v", task.ID, err)
			if err := o.repo.UpdateLogs(task.ID, logEntry); err != nil {
				log.Printf("orchestrator: update logs for task %s: %v", task.ID, err)
			}
			if err := o.repo.UpdateStatus(task.ID, domain.StatusFailed); err != nil {
				log.Printf("orchestrator: update status for task %s: %v", task.ID, err)
			}
			o.emit(task.ID, domain.StatusFailed)
			return
		}
	}

	if o.fileWriter != nil && task.TargetFile != "" {
		if err := o.fileWriter.WriteCodeToFile(task.ProjectPath, task.TargetFile, extractCode(code)); err != nil {
			logEntry := fmt.Sprintf("failed writing output via %s: %v", llm.ProviderName(), err)
			log.Printf("orchestrator: write file for task %s: %v", task.ID, err)
			if err := o.repo.UpdateLogs(task.ID, logEntry); err != nil {
				log.Printf("orchestrator: update logs for task %s: %v", task.ID, err)
			}
			if err := o.repo.UpdateStatus(task.ID, domain.StatusFailed); err != nil {
				log.Printf("orchestrator: update status for task %s: %v", task.ID, err)
			}
			o.emit(task.ID, domain.StatusFailed)
			return
		}
	}

	logEntry := fmt.Sprintf("completed via %s at %s", llm.ProviderName(), time.Now().UTC().Format(time.RFC3339))
	if err := o.repo.UpdateLogs(task.ID, logEntry); err != nil {
		log.Printf("orchestrator: update logs for task %s: %v", task.ID, err)
	}
	if err := o.repo.UpdateStatus(task.ID, domain.StatusCompleted); err != nil {
		log.Printf("orchestrator: update status for task %s: %v", task.ID, err)
	}
	o.emit(task.ID, domain.StatusCompleted)
	log.Printf("orchestrator: task %s completed via %s", task.ID, llm.ProviderName())
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
