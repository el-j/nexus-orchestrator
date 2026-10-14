package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

var errContextTooLarge = errors.New("context too large")

// errEmptyResponse is returned when a provider answers successfully but with no
// usable text. Treating it as a failed attempt lets the fallback chain and retry
// logic engage instead of "completing" a task with an empty file.
var errEmptyResponse = errors.New("provider returned an empty response")

// nonEmpty turns a blank successful reply into errEmptyResponse.
func nonEmpty(code string, err error) (string, error) {
	if err == nil && strings.TrimSpace(code) == "" {
		return "", errEmptyResponse
	}
	return code, err
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

// emit publishes a TaskEvent if a broadcaster is configured.
// It acquires the mutex only to read the broadcaster pointer, then releases it
// before calling Broadcast so the hub's own lock is never nested under o.mu.
func (o *OrchestratorService) emit(taskID string, status domain.TaskStatus) {
	o.recordOutcome(taskID, status)
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

func (o *OrchestratorService) processNext() bool {
	task, err := o.repo.ClaimNextQueued(o.dbCtx())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return false
		}
		log.Printf("orchestrator: claim next queued: %v", err)
		return false
	}

	run := o.beginRun(task.ID)
	defer o.endRun(task.ID)
	// A cancel can land between the claim and the registration above, when there is
	// no run to abort yet; honour it here instead of overwriting CANCELLED later.
	if cur, err := o.repo.GetByID(o.dbCtx(), task.ID); err == nil && cur.Status == domain.StatusCancelled {
		return true
	}

	chain, err := o.resolveProviderWithFallback(run.ctx, task)
	if err != nil || len(chain) == 0 {
		logMsg := "no provider available"
		if err != nil {
			logMsg = err.Error()
		}
		log.Printf("orchestrator: no provider for task %s (role=%q, model=%q): %s", task.ID, task.Role, task.ModelID, logMsg)
		if err2 := o.repo.UpdateLogs(o.dbCtx(), task.ID, logMsg); err2 != nil {
			log.Printf("orchestrator: update logs for task %s: %v", task.ID, err2)
		}
		if err2 := o.repo.UpdateStatus(o.dbCtx(), task.ID, domain.StatusNoProvider); err2 != nil {
			log.Printf("orchestrator: update status for task %s: %v", task.ID, err2)
		}
		o.emit(task.ID, domain.StatusNoProvider)
		return true
	}
	o.emit(task.ID, domain.StatusProcessing)

	prompt, sessionHistory, err := o.prepareChatPrompt(task)
	if err != nil {
		// prepareChatPrompt sets the task status internally before returning an error.
		return true
	}

	task, policy, prompt, err := o.applyProjectRules(run.ctx, task, chain, prompt)
	if err != nil {
		o.appendTaskLog(task.ID, "not started: "+err.Error())
		if err2 := o.repo.UpdateStatus(o.dbCtx(), task.ID, domain.StatusFailed); err2 != nil {
			log.Printf("orchestrator: update status for task %s: %v", task.ID, err2)
		}
		o.emit(task.ID, domain.StatusFailed)
		return true
	}

	if task.Agentic {
		return o.runAgentLoop(run, task, chain, prompt, policy)
	}
	return o.executeWithFallbackChain(run, task, chain, prompt, sessionHistory)
}

// beginRun registers a task as running so CancelTask can abort it. Its context is
// a child of the service's base context, so Stop aborts it as well.
func (o *OrchestratorService) beginRun(taskID string) *runHandle {
	ctx, cancel := context.WithCancel(o.baseCtx)
	h := &runHandle{ctx: ctx, cancel: cancel}
	o.runMu.Lock()
	o.running[taskID] = h
	o.runMu.Unlock()
	return h
}

// endRun releases the resources of a finished run.
func (o *OrchestratorService) endRun(taskID string) {
	o.runMu.Lock()
	h := o.running[taskID]
	delete(o.running, taskID)
	o.runMu.Unlock()
	if h != nil {
		h.cancel()
	}
}

// abortRun asks the engine to stop generating for taskID on behalf of a user.
// It reports whether this engine was running the task.
func (o *OrchestratorService) abortRun(taskID string) bool {
	o.runMu.Lock()
	h := o.running[taskID]
	o.runMu.Unlock()
	if h == nil {
		return false
	}
	h.userCancelled.Store(true)
	h.cancel()
	return true
}

// finishInterrupted settles a task whose run was aborted. A task the user
// cancelled becomes CANCELLED; one interrupted by shutdown goes back to the queue
// so the next start picks it up (it was not the task's fault).
func (o *OrchestratorService) finishInterrupted(task domain.Task, run *runHandle) {
	if run.userCancelled.Load() {
		log.Printf("orchestrator: task %s cancelled while running", task.ID)
		o.appendTaskLog(task.ID, "cancelled by user while running")
		if err := o.repo.UpdateStatus(o.dbCtx(), task.ID, domain.StatusCancelled); err != nil {
			log.Printf("orchestrator: update status for task %s: %v", task.ID, err)
		}
		o.emit(task.ID, domain.StatusCancelled)
		return
	}
	log.Printf("orchestrator: task %s interrupted by shutdown; re-queued", task.ID)
	if ok, err := o.repo.UpdateStatusIfCurrent(o.dbCtx(), task.ID, domain.StatusProcessing, domain.StatusQueued); err != nil {
		log.Printf("orchestrator: re-queue task %s: %v", task.ID, err)
	} else if ok {
		o.emit(task.ID, domain.StatusQueued)
	}
}

// tryGenerate dispatches to Chat (when sessionRepo is set) or GenerateCode for a single attempt.
func (o *OrchestratorService) tryGenerate(ctx context.Context, llm ports.LLMClient, prompt string, sessionHistory []domain.Message) (string, error) {
	if o.sessionRepo != nil {
		userMsg := domain.Message{Role: domain.RoleUser, Content: prompt, CreatedAt: time.Now()}
		history := append(append([]domain.Message(nil), sessionHistory...), userMsg)
		return nonEmpty(llm.Chat(ctx, history))
	}
	return nonEmpty(llm.GenerateCode(ctx, prompt))
}

// appendTaskLog appends a new log line to existing task logs without overwriting history.
func (o *OrchestratorService) appendTaskLog(taskID string, newEntry string) {
	entry := newEntry
	if existing, err := o.repo.GetByID(o.dbCtx(), taskID); err == nil && existing.Logs != "" {
		entry = existing.Logs + "\n" + newEntry
	}
	if err := o.repo.UpdateLogs(o.dbCtx(), taskID, entry); err != nil {
		log.Printf("orchestrator: update logs for task %s: %v", taskID, err)
	}
}

// executeWithFallbackChain attempts generation across an ordered chain of providers.
// On transient error or rate limit, it falls back to the next provider in the chain.
func (o *OrchestratorService) executeWithFallbackChain(run *runHandle, task domain.Task, chain []ports.LLMClient, prompt string, sessionHistory []domain.Message) bool {
	var lastErr error
	var tried []string
	allTooLarge := true

	for i, llm := range chain {
		if run.ctx.Err() != nil {
			o.finishInterrupted(task, run)
			return true
		}
		if limit := llm.ContextLimit(run.ctx); limit > 0 {
			estHistory := make([]domain.Message, len(sessionHistory)+1)
			copy(estHistory, sessionHistory)
			estHistory[len(sessionHistory)] = domain.Message{Role: domain.RoleUser, Content: prompt}
			if estimated := estimateTokens(estHistory); estimated > limit-o.maxResponseTokens {
				logEntry := fmt.Sprintf(
					"context too large for %s: ~%d tokens estimated, model limit is %d (headroom %d)",
					llm.ProviderName(), estimated, limit, o.maxResponseTokens,
				)
				log.Printf("orchestrator: task %s: %s", task.ID, logEntry)
				lastErr = errContextTooLarge
				continue
			}
		}
		allTooLarge = false

		code, err := o.tryGenerate(run.ctx, llm, prompt, sessionHistory)
		if run.ctx.Err() != nil {
			// Cancelled mid-generation: whatever came back (error or not) is discarded.
			o.finishInterrupted(task, run)
			return true
		}
		if err != nil {
			lastErr = err
			tried = append(tried, llm.ProviderName())
			logEntry := fmt.Sprintf("provider %s failed: %v", llm.ProviderName(), err)
			log.Printf("orchestrator: task %s: %s", task.ID, logEntry)
			o.appendTaskLog(task.ID, logEntry)
			continue
		}

		// Success!
		if i > 0 {
			failoverNote := fmt.Sprintf("[failover: primary provider failed, executed via %s]", llm.ProviderName())
			log.Printf("orchestrator: task %s: %s", task.ID, failoverNote)
			o.appendTaskLog(task.ID, failoverNote)
		}

		if o.sessionRepo != nil {
			userMsg := domain.Message{Role: domain.RoleUser, Content: prompt, CreatedAt: time.Now()}
			assistantMsg := domain.Message{Role: domain.RoleAssistant, Content: code, CreatedAt: time.Now()}
			if err := o.sessionRepo.AppendMessage(o.dbCtx(), task.ProjectPath, userMsg); err != nil {
				log.Printf("orchestrator: append user message for task %s: %v", task.ID, err)
			}
			if err := o.sessionRepo.AppendMessage(o.dbCtx(), task.ProjectPath, assistantMsg); err != nil {
				log.Printf("orchestrator: append assistant message for task %s: %v", task.ID, err)
			}
		}

		o.writeAndVerifyTaskOutput(run, task, code, llm, sessionHistory)
		return true
	}

	// All providers failed
	if allTooLarge && errors.Is(lastErr, errContextTooLarge) {
		logEntry := "context too large: all candidate providers exceeded context limits"
		_ = o.repo.UpdateLogs(o.dbCtx(), task.ID, logEntry)
		_ = o.repo.UpdateStatus(o.dbCtx(), task.ID, domain.StatusTooLarge)
		o.emit(task.ID, domain.StatusTooLarge)
		return true
	}

	if o.requeueForRetry(task) {
		return true
	}

	failLog := fmt.Sprintf("all candidate providers failed: %s (last error: %v)", strings.Join(tried, ", "), lastErr)
	_ = o.repo.UpdateLogs(o.dbCtx(), task.ID, failLog)
	if err := o.repo.UpdateStatus(o.dbCtx(), task.ID, domain.StatusFailed); err != nil {
		log.Printf("orchestrator: update status for task %s: %v", task.ID, err)
	}
	o.emit(task.ID, domain.StatusFailed)
	return true
}

// executeGeneration dispatches to Chat (when sessionRepo is set) or GenerateCode,
// with retry on transient failures. On fatal failure it persists StatusFailed.
// On success with a sessionRepo it appends the user and assistant messages to the session.
func (o *OrchestratorService) executeGeneration(run *runHandle, task domain.Task, llm ports.LLMClient, prompt string, sessionHistory []domain.Message) (string, error) {
	if o.sessionRepo != nil {
		// Build the chat history using the already-loaded session (no second DB call).
		userMsg := domain.Message{Role: domain.RoleUser, Content: prompt, CreatedAt: time.Now()}
		history := append(append([]domain.Message(nil), sessionHistory...), userMsg)
		code, err := nonEmpty(llm.Chat(run.ctx, history))
		if run.ctx.Err() != nil {
			o.finishInterrupted(task, run)
			return "", run.ctx.Err()
		}
		if err != nil {
			logEntry := fmt.Sprintf("failed via %s: %v", llm.ProviderName(), err)
			log.Printf("orchestrator: chat for task %s: %v", task.ID, err)
			if o.requeueForRetry(task) {
				return "", err
			}
			if err2 := o.repo.UpdateLogs(o.dbCtx(), task.ID, logEntry); err2 != nil {
				log.Printf("orchestrator: update logs for task %s: %v", task.ID, err2)
			}
			if err2 := o.repo.UpdateStatus(o.dbCtx(), task.ID, domain.StatusFailed); err2 != nil {
				log.Printf("orchestrator: update status for task %s: %v", task.ID, err2)
			}
			o.emit(task.ID, domain.StatusFailed)
			return "", err
		}
		// Only persist messages after a successful response.
		assistantMsg := domain.Message{Role: domain.RoleAssistant, Content: code, CreatedAt: time.Now()}
		if err := o.sessionRepo.AppendMessage(o.dbCtx(), task.ProjectPath, userMsg); err != nil {
			log.Printf("orchestrator: append user message for task %s: %v", task.ID, err)
		}
		if err := o.sessionRepo.AppendMessage(o.dbCtx(), task.ProjectPath, assistantMsg); err != nil {
			log.Printf("orchestrator: append assistant message for task %s: %v", task.ID, err)
		}
		return code, nil
	}

	code, err := nonEmpty(llm.GenerateCode(run.ctx, prompt))
	if run.ctx.Err() != nil {
		o.finishInterrupted(task, run)
		return "", run.ctx.Err()
	}
	if err != nil {
		logEntry := fmt.Sprintf("failed via %s: %v", llm.ProviderName(), err)
		log.Printf("orchestrator: generate code for task %s: %v", task.ID, err)
		if o.requeueForRetry(task) {
			return "", err
		}
		if err2 := o.repo.UpdateLogs(o.dbCtx(), task.ID, logEntry); err2 != nil {
			log.Printf("orchestrator: update logs for task %s: %v", task.ID, err2)
		}
		if err2 := o.repo.UpdateStatus(o.dbCtx(), task.ID, domain.StatusFailed); err2 != nil {
			log.Printf("orchestrator: update status for task %s: %v", task.ID, err2)
		}
		o.emit(task.ID, domain.StatusFailed)
		return "", err
	}
	return code, nil
}
