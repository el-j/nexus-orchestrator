package services

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"

	"github.com/google/uuid"
)

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

// CancelTask removes a QUEUED or NO_PROVIDER task before it is processed.
func (o *OrchestratorService) CancelTask(id string) error {
	// First try QUEUED → CANCELLED
	ok, err := o.repo.UpdateStatusIfCurrent(id, domain.StatusQueued, domain.StatusCancelled)
	if err != nil {
		return fmt.Errorf("orchestrator: cancel task: %w", err)
	}
	if ok {
		o.emit(id, domain.StatusCancelled)
		return nil
	}
	// Also allow NO_PROVIDER → CANCELLED (task was stuck waiting for provider)
	ok, err = o.repo.UpdateStatusIfCurrent(id, domain.StatusNoProvider, domain.StatusCancelled)
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
func (o *OrchestratorService) PromoteTask(id string) (ports.PromoteResult, error) {
	task, err := o.repo.GetByID(id)
	if err != nil {
		return ports.PromoteResult{}, fmt.Errorf("orchestrator: promote task: %w", err)
	}
	if task.Status != domain.StatusDraft && task.Status != domain.StatusBacklog {
		return ports.PromoteResult{}, fmt.Errorf("orchestrator: promote task: cannot promote task with status %s", task.Status)
	}
	if err := o.validateQueueAdmission(task); err != nil {
		return ports.PromoteResult{}, fmt.Errorf("orchestrator: promote task: %w", err)
	}
	// Soft pre-flight: check provider availability (no DB side effects)
	var warning string
	if _, provErr := o.selectProviderForTaskDryRun(task); provErr != nil {
		warning = fmt.Sprintf("no active provider available (%s); task will execute when a provider comes online", provErr.Error())
	}
	ok, err := o.repo.UpdateStatusIfCurrent(id, task.Status, domain.StatusQueued)
	if err != nil {
		return ports.PromoteResult{}, fmt.Errorf("orchestrator: promote task: %w", err)
	}
	if !ok {
		return ports.PromoteResult{}, fmt.Errorf("orchestrator: promote task: task state changed during promotion")
	}
	o.signalWorker()
	o.emit(task.ID, domain.StatusQueued)
	return ports.PromoteResult{Promoted: true, Warning: warning}, nil
}

// selectProviderForTaskDryRun checks provider availability without mutating task state.
func (o *OrchestratorService) selectProviderForTaskDryRun(task domain.Task) (ports.LLMClient, error) {
	if task.ProviderName != "" {
		if client, ok := o.discovery.GetClientByName(task.ProviderName); ok {
			return client, nil
		}
		return nil, fmt.Errorf("provider %q not found or not active", task.ProviderName)
	}
	return o.discovery.FindForModel(task.ModelID, task.ProviderHint)
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

// HeartbeatTask keeps a PROCESSING task alive, preventing the watchdog from failing it.
func (o *OrchestratorService) HeartbeatTask(ctx context.Context, taskID string, sessionID string) error {
	task, err := o.repo.GetByID(taskID)
	if err != nil {
		return fmt.Errorf("heartbeat task: %w", err)
	}
	if task.Status != domain.StatusProcessing {
		return fmt.Errorf("heartbeat task: task not processing")
	}
	if task.AISessionID != sessionID {
		return fmt.Errorf("heartbeat task: not claimed by session %s", sessionID)
	}
	// UpdateStatusIfCurrent touches UpdatedAt
	ok, err := o.repo.UpdateStatusIfCurrent(task.ID, domain.StatusProcessing, domain.StatusProcessing)
	if err != nil {
		return fmt.Errorf("heartbeat task: update status: %w", err)
	}
	if !ok {
		return fmt.Errorf("heartbeat task: status changed")
	}
	return nil
}
