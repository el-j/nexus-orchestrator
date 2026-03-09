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
	mu         sync.Mutex
	queue      []domain.Task
	discovery  *DiscoveryService
	fileWriter ports.FileWriter
	repo       ports.TaskRepository
	stopCh     chan struct{}
}

// NewOrchestrator constructs an OrchestratorService and starts the background
// worker that pulls QUEUED tasks and sends them to the active LLM.
func NewOrchestrator(
	discovery *DiscoveryService,
	repo ports.TaskRepository,
	writer ports.FileWriter,
) *OrchestratorService {
	svc := &OrchestratorService{
		discovery:  discovery,
		repo:       repo,
		fileWriter: writer,
		stopCh:     make(chan struct{}),
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

	o.mu.Lock()
	o.queue = append(o.queue, task)
	o.mu.Unlock()

	return task.ID, nil
}

// GetQueue returns all pending (QUEUED or PROCESSING) tasks.
func (o *OrchestratorService) GetQueue() ([]domain.Task, error) {
	return o.repo.GetPending()
}

// CancelTask removes a QUEUED task before it is processed.
func (o *OrchestratorService) CancelTask(id string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	for i, t := range o.queue {
		if t.ID == id {
			o.queue = append(o.queue[:i], o.queue[i+1:]...)
			return o.repo.UpdateStatus(id, domain.StatusFailed)
		}
	}
	return errors.New("orchestrator: task not found in queue")
}

// Stop signals the worker goroutine to exit.
func (o *OrchestratorService) Stop() {
	close(o.stopCh)
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

	code, err := llm.GenerateCode(prompt)
	if err != nil {
		log.Printf("orchestrator: generate code for task %s: %v", task.ID, err)
		_ = o.repo.UpdateStatus(task.ID, domain.StatusFailed)
		return
	}

	if o.fileWriter != nil && task.TargetFile != "" {
		if err := o.fileWriter.WriteCodeToFile(task.ProjectPath, task.TargetFile, code); err != nil {
			log.Printf("orchestrator: write file for task %s: %v", task.ID, err)
			_ = o.repo.UpdateStatus(task.ID, domain.StatusFailed)
			return
		}
	}

	_ = o.repo.UpdateStatus(task.ID, domain.StatusCompleted)
	log.Printf("orchestrator: task %s completed via %s", task.ID, llm.ProviderName())
}
