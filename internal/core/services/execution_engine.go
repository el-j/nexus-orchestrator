package services

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

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
		// buildChatContext sets the task status internally before returning an error.
		// No second UpdateStatus call here — that would overwrite TooLarge/Failed correctly set within.
		return true
	}

	code, err := o.executeGeneration(task, llm, prompt, sessionHistory)
	if err != nil {
		// executeGeneration sets the task status internally before returning an error.
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
			logEntry := fmt.Sprintf("failed reading context files: %v", err)
			log.Printf("orchestrator: task %s: %s", task.ID, logEntry)
			if err2 := o.repo.UpdateLogs(task.ID, logEntry); err2 != nil {
				log.Printf("orchestrator: update logs for task %s: %v", task.ID, err2)
			}
			if err2 := o.repo.UpdateStatus(task.ID, domain.StatusFailed); err2 != nil {
				log.Printf("orchestrator: update status for task %s: %v", task.ID, err2)
			}
			o.emit(task.ID, domain.StatusFailed)
			return "", nil, fmt.Errorf("orchestrator: read context files: %w", err)
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
			logEntry := fmt.Sprintf("failed loading session history: %v", err)
			log.Printf("orchestrator: task %s: %s", task.ID, logEntry)
			if err2 := o.repo.UpdateLogs(task.ID, logEntry); err2 != nil {
				log.Printf("orchestrator: update logs for task %s: %v", task.ID, err2)
			}
			if err2 := o.repo.UpdateStatus(task.ID, domain.StatusFailed); err2 != nil {
				log.Printf("orchestrator: update status for task %s: %v", task.ID, err2)
			}
			o.emit(task.ID, domain.StatusFailed)
			return "", nil, fmt.Errorf("orchestrator: load session: %w", err)
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

// delegationInstruction returns a formatted prompt instructing an AI agent to
// operate under nexusOrchestrator coordination.
func (o *OrchestratorService) delegationInstruction(session domain.AISession, now time.Time) string {
	addr := o.daemonAddr
	return fmt.Sprintf(`You are now operating under nexusOrchestrator coordination.

YOUR NEW WORKFLOW:
1. Check the nexus queue:  GET %s/api/tasks
2. Claim the next QUEUED task:
   POST %s/api/tasks/{id}/claim
   Body: {"sessionId": "%s"}
3. Execute the task's instruction for %s.
4. Report completion:
   PUT %s/api/tasks/{id}/status
   Body: {"sessionId": "%s", "status": "COMPLETED", "logs": "<summary>"}
5. Repeat from step 1.  If queue is empty, wait 30 s and check again.

Do NOT take autonomous actions outside the nexus task queue while this delegation is active.

Nexus session ID: %s
Project:          %s
Timestamp:        %s`,
		addr, addr, session.ID, session.ProjectPath,
		addr, session.ID,
		session.ID, session.ProjectPath,
		now.UTC().Format(time.RFC3339))
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
