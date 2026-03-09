package services

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"nexus-ai/internal/core/domain"
	"nexus-ai/internal/core/ports"
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
	stopCh      chan struct{}
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
		stopCh:      make(chan struct{}),
	}
	go svc.runWorker()
	return svc
}

// SubmitTask enqueues a new Task and returns its generated ID.
func (o *OrchestratorService) SubmitTask(task domain.Task) (string, error) {
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
	return errors.New("orchestrator: task not found in queue")
}

// Stop signals the worker goroutine to exit.
func (o *OrchestratorService) Stop() {
	close(o.stopCh)
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
	eventType := "task." + strings.ToLower(string(status))
	b.Broadcast(ports.TaskEvent{
		Type:   eventType,
		TaskID: taskID,
		Status: status,
	})
}

// runWorker is the background loop that processes QUEUED tasks sequentially.
func (o *OrchestratorService) runWorker() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-o.stopCh:
			return
		case <-ticker.C:
			o.processNext()
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

	llm := o.discovery.DetectActive()
	if llm == nil {
		log.Printf("orchestrator: no LLM available, re-queuing task %s", task.ID)
		o.mu.Lock()
		o.queue = append([]domain.Task{task}, o.queue...)
		o.mu.Unlock()
		return
	}

	_ = o.repo.UpdateStatus(task.ID, domain.StatusProcessing)
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

	var code string
	if o.sessionRepo != nil {
		// Session-isolated generation: load history, append user turn, call Chat.
		userMsg := domain.Message{Role: "user", Content: prompt, CreatedAt: time.Now()}
		if err := o.sessionRepo.AppendMessage(task.ProjectPath, userMsg); err != nil {
			log.Printf("orchestrator: append user message for task %s: %v", task.ID, err)
		}
		sess, err := o.sessionRepo.GetByProjectPath(task.ProjectPath)
		if err != nil {
			log.Printf("orchestrator: load session for task %s: %v", task.ID, err)
			sess.Messages = []domain.Message{userMsg}
		}
		code, err = llm.Chat(sess.Messages)
		if err != nil {
			logEntry := fmt.Sprintf("failed via %s: %v", llm.ProviderName(), err)
			log.Printf("orchestrator: chat for task %s: %v", task.ID, err)
			_ = o.repo.UpdateLogs(task.ID, logEntry)
			_ = o.repo.UpdateStatus(task.ID, domain.StatusFailed)
			o.emit(task.ID, domain.StatusFailed)
			return
		}
		// Persist assistant reply to session.
		assistantMsg := domain.Message{Role: "assistant", Content: code, CreatedAt: time.Now()}
		if err := o.sessionRepo.AppendMessage(task.ProjectPath, assistantMsg); err != nil {
			log.Printf("orchestrator: append assistant message for task %s: %v", task.ID, err)
		}
	} else {
		var err error
		code, err = llm.GenerateCode(prompt)
		if err != nil {
			logEntry := fmt.Sprintf("failed via %s: %v", llm.ProviderName(), err)
			log.Printf("orchestrator: generate code for task %s: %v", task.ID, err)
			_ = o.repo.UpdateLogs(task.ID, logEntry)
			_ = o.repo.UpdateStatus(task.ID, domain.StatusFailed)
			o.emit(task.ID, domain.StatusFailed)
			return
		}
	}

	if o.fileWriter != nil && task.TargetFile != "" {
		if err := o.fileWriter.WriteCodeToFile(task.ProjectPath, task.TargetFile, code); err != nil {
			logEntry := fmt.Sprintf("failed writing output via %s: %v", llm.ProviderName(), err)
			log.Printf("orchestrator: write file for task %s: %v", task.ID, err)
			_ = o.repo.UpdateLogs(task.ID, logEntry)
			_ = o.repo.UpdateStatus(task.ID, domain.StatusFailed)
			o.emit(task.ID, domain.StatusFailed)
			return
		}
	}

	logEntry := fmt.Sprintf("completed via %s at %s", llm.ProviderName(), time.Now().UTC().Format(time.RFC3339))
	_ = o.repo.UpdateLogs(task.ID, logEntry)
	_ = o.repo.UpdateStatus(task.ID, domain.StatusCompleted)
	o.emit(task.ID, domain.StatusCompleted)
	log.Printf("orchestrator: task %s completed via %s", task.ID, llm.ProviderName())
}
