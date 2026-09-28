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

	chain, err := o.resolveProviderWithFallback(task)
	if err != nil || len(chain) == 0 {
		logMsg := "no provider available"
		if err != nil {
			logMsg = err.Error()
		}
		log.Printf("orchestrator: no provider for task %s (role=%q, model=%q): %s", task.ID, task.Role, task.ModelID, logMsg)
		if err2 := o.repo.UpdateLogs(task.ID, logMsg); err2 != nil {
			log.Printf("orchestrator: update logs for task %s: %v", task.ID, err2)
		}
		if err2 := o.repo.UpdateStatus(task.ID, domain.StatusNoProvider); err2 != nil {
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

	return o.executeWithFallbackChain(task, chain, prompt, sessionHistory)
}

// isFrontierRole returns true for complex reasoning, architectural, and planning roles.
func isFrontierRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "architect", "techlead", "planner", "frontier", "deep_debug", "complex", "backend":
		return true
	default:
		return false
	}
}

// isFastLocalRole returns true for routine, formatting, and boilerplate tasks.
func isFastLocalRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "linter", "formatter", "test", "tester", "boilerplate", "fast", "local":
		return true
	default:
		return false
	}
}

// isFrontierProvider tests if a provider matches known frontier cloud LLMs.
func isFrontierProvider(p ports.LLMClient) bool {
	name := strings.ToLower(p.ProviderName())
	return strings.Contains(name, "anthropic") ||
		strings.Contains(name, "gemini") ||
		strings.Contains(name, "openai") ||
		strings.Contains(name, "claude")
}

// isLocalProvider tests if a provider matches known local/on-premise LLMs.
func isLocalProvider(p ports.LLMClient) bool {
	name := strings.ToLower(p.ProviderName())
	return strings.Contains(name, "ollama") ||
		strings.Contains(name, "lmstudio") ||
		strings.Contains(name, "localai") ||
		strings.Contains(name, "vllm") ||
		strings.Contains(name, "local")
}

// resolveProviderWithFallback returns an ordered chain of available LLM providers
// based on task requirements, role-driven routing hints, and health status.
func (o *OrchestratorService) resolveProviderWithFallback(task domain.Task) ([]ports.LLMClient, error) {
	if o.discovery == nil {
		return nil, errors.New("discovery service not configured")
	}

	// Case 1: Explicit ProviderName specified
	if task.ProviderName != "" {
		client, ok := o.discovery.GetClientByName(task.ProviderName)
		if !ok {
			return nil, fmt.Errorf("provider %q not found or not active", task.ProviderName)
		}
		chain := []ports.LLMClient{client}
		for _, c := range o.discovery.GetAllClients() {
			if !strings.EqualFold(c.ProviderName(), client.ProviderName()) && c.Ping() {
				chain = append(chain, c)
			}
		}
		return chain, nil
	}

	all := o.discovery.GetAllClients()
	if len(all) == 0 {
		return nil, errors.New("discovery: no active provider available")
	}

	// Check if a specific model was requested
	var preferred ports.LLMClient
	if task.ModelID != "" {
		c, err := o.discovery.FindForModel(task.ModelID, task.ProviderHint)
		if err != nil {
			return nil, err
		}
		preferred = c
	}

	// Filter down to alive providers that can serve the task
	var alive []ports.LLMClient
	for _, c := range all {
		if c.Ping() {
			if task.ModelID != "" {
				if strings.EqualFold(c.ActiveModel(), task.ModelID) {
					alive = append(alive, c)
				} else if models, err := c.GetAvailableModels(); err == nil {
					for _, m := range models {
						if strings.EqualFold(m, task.ModelID) {
							alive = append(alive, c)
							break
						}
					}
				}
			} else {
				alive = append(alive, c)
			}
		}
	}
	if len(alive) == 0 {
		if task.ModelID != "" {
			return nil, fmt.Errorf("discovery: model %q not available on any registered provider", task.ModelID)
		}
		return nil, errors.New("discovery: no active provider available")
	}

	// Classify alive providers
	var frontierClients, localClients, otherClients []ports.LLMClient
	for _, c := range alive {
		switch {
		case isFrontierProvider(c):
			frontierClients = append(frontierClients, c)
		case isLocalProvider(c):
			localClients = append(localClients, c)
		default:
			otherClients = append(otherClients, c)
		}
	}

	var ordered []ports.LLMClient
	switch {
	case isFastLocalRole(task.Role):
		// Fast local chain: Local -> Other -> Frontier
		ordered = append(ordered, localClients...)
		ordered = append(ordered, otherClients...)
		ordered = append(ordered, frontierClients...)
	case isFrontierRole(task.Role):
		// Frontier chain: Frontier -> Other -> Local
		ordered = append(ordered, frontierClients...)
		ordered = append(ordered, otherClients...)
		ordered = append(ordered, localClients...)
	default:
		// Default / Balanced: preserve alive order, or Frontier first if alive
		if len(frontierClients) > 0 {
			ordered = append(ordered, frontierClients...)
			ordered = append(ordered, otherClients...)
			ordered = append(ordered, localClients...)
		} else {
			ordered = alive
		}
	}

	// Deduplicate, ensuring preferred client is at index 0 if present
	var result []ports.LLMClient
	seen := make(map[string]bool)
	if preferred != nil {
		result = append(result, preferred)
		seen[strings.ToLower(preferred.ProviderName())] = true
	}
	for _, c := range ordered {
		name := strings.ToLower(c.ProviderName())
		if !seen[name] {
			result = append(result, c)
			seen[name] = true
		}
	}

	if len(result) == 0 {
		return nil, errors.New("discovery: no active provider available")
	}
	return result, nil
}

// selectProviderForTask resolves the single primary LLM client for the task.
// Retained for backward compatibility and direct single-provider invocations.
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

// prepareChatPrompt loads context files and session history once for the task.
func (o *OrchestratorService) prepareChatPrompt(task domain.Task) (string, []domain.Message, error) {
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

	return prompt, sessionHistory, nil
}

// buildChatContext constructs the prompt with optional context file content prepended,
// loads session history, and guards against context-window overflow.
// On overflow it sets StatusTooLarge, logs the reason, and emits the event.
func (o *OrchestratorService) buildChatContext(task domain.Task, llm ports.LLMClient) (string, []domain.Message, error) {
	prompt, sessionHistory, err := o.prepareChatPrompt(task)
	if err != nil {
		return "", nil, err
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
			return "", nil, errContextTooLarge
		}
	}

	return prompt, sessionHistory, nil
}

// tryGenerate dispatches to Chat (when sessionRepo is set) or GenerateCode for a single attempt.
func (o *OrchestratorService) tryGenerate(task domain.Task, llm ports.LLMClient, prompt string, sessionHistory []domain.Message) (string, error) {
	if o.sessionRepo != nil {
		userMsg := domain.Message{Role: domain.RoleUser, Content: prompt, CreatedAt: time.Now()}
		history := append(append([]domain.Message(nil), sessionHistory...), userMsg)
		return llm.Chat(history)
	}
	return llm.GenerateCode(prompt)
}

// appendTaskLog appends a new log line to existing task logs without overwriting history.
func (o *OrchestratorService) appendTaskLog(taskID string, newEntry string) {
	entry := newEntry
	if existing, err := o.repo.GetByID(taskID); err == nil && existing.Logs != "" {
		entry = existing.Logs + "\n" + newEntry
	}
	if err := o.repo.UpdateLogs(taskID, entry); err != nil {
		log.Printf("orchestrator: update logs for task %s: %v", taskID, err)
	}
}

// executeWithFallbackChain attempts generation across an ordered chain of providers.
// On transient error or rate limit, it falls back to the next provider in the chain.
func (o *OrchestratorService) executeWithFallbackChain(task domain.Task, chain []ports.LLMClient, prompt string, sessionHistory []domain.Message) bool {
	var lastErr error
	var tried []string
	allTooLarge := true

	for i, llm := range chain {
		if limit := llm.ContextLimit(); limit > 0 {
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

		code, err := o.tryGenerate(task, llm, prompt, sessionHistory)
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
			if err := o.sessionRepo.AppendMessage(task.ProjectPath, userMsg); err != nil {
				log.Printf("orchestrator: append user message for task %s: %v", task.ID, err)
			}
			if err := o.sessionRepo.AppendMessage(task.ProjectPath, assistantMsg); err != nil {
				log.Printf("orchestrator: append assistant message for task %s: %v", task.ID, err)
			}
		}

		o.writeAndVerifyTaskOutput(task, code, llm, sessionHistory)
		return true
	}

	// All providers failed
	if allTooLarge && errors.Is(lastErr, errContextTooLarge) {
		logEntry := "context too large: all candidate providers exceeded context limits"
		_ = o.repo.UpdateLogs(task.ID, logEntry)
		_ = o.repo.UpdateStatus(task.ID, domain.StatusTooLarge)
		o.emit(task.ID, domain.StatusTooLarge)
		return true
	}

	if o.requeueForRetry(task) {
		return true
	}

	failLog := fmt.Sprintf("all candidate providers failed: %s (last error: %v)", strings.Join(tried, ", "), lastErr)
	_ = o.repo.UpdateLogs(task.ID, failLog)
	if err := o.repo.UpdateStatus(task.ID, domain.StatusFailed); err != nil {
		log.Printf("orchestrator: update status for task %s: %v", task.ID, err)
	}
	o.emit(task.ID, domain.StatusFailed)
	return true
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
// as COMPLETED. It delegates to writeAndVerifyTaskOutput without active LLM correction.
func (o *OrchestratorService) writeTaskOutput(task domain.Task, code string, providerName string) {
	o.writeAndVerifyTaskOutput(task, code, nil, nil)
}

// writeAndVerifyTaskOutput writes the generated code to disk, runs verification commands if configured,
// and initiates a self-healing correction loop if verification fails.
func (o *OrchestratorService) writeAndVerifyTaskOutput(task domain.Task, code string, llm ports.LLMClient, sessionHistory []domain.Message) {
	providerName := "unknown"
	if llm != nil {
		providerName = llm.ProviderName()
	} else if task.ProviderName != "" {
		providerName = task.ProviderName
	}

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

	// If no verification command is configured or command runner is nil, complete normally.
	if strings.TrimSpace(task.VerificationCommand) == "" || o.commandRunner == nil {
		logEntry := fmt.Sprintf("completed via %s at %s", providerName, time.Now().UTC().Format(time.RFC3339))
		o.appendTaskLog(task.ID, logEntry)
		if err := o.repo.UpdateStatus(task.ID, domain.StatusCompleted); err != nil {
			log.Printf("orchestrator: update status for task %s: %v", task.ID, err)
		}
		o.emit(task.ID, domain.StatusCompleted)
		log.Printf("orchestrator: task %s completed via %s", task.ID, providerName)
		return
	}

	// Run verification command
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	output, runErr := o.commandRunner.Run(ctx, task.ProjectPath, task.VerificationCommand)
	task.VerificationOutput = output

	if runErr == nil {
		// Verification passed on first attempt
		logEntry := fmt.Sprintf("verification passed (%s) — completed via %s at %s", task.VerificationCommand, providerName, time.Now().UTC().Format(time.RFC3339))
		_ = o.repo.Update(task)
		o.appendTaskLog(task.ID, logEntry)
		if err := o.repo.UpdateStatus(task.ID, domain.StatusCompleted); err != nil {
			log.Printf("orchestrator: update status for task %s: %v", task.ID, err)
		}
		o.emit(task.ID, domain.StatusCompleted)
		log.Printf("orchestrator: task %s verification passed, completed via %s", task.ID, providerName)
		return
	}

	// Verification failed — attempt self-healing correction loop if LLM is available
	maxTurns := task.MaxCorrectionTurns
	if maxTurns <= 0 {
		maxTurns = 2
	}

	if llm != nil {
		for turn := 1; turn <= maxTurns; turn++ {
			log.Printf("orchestrator: task %s: verification failed on turn %d/%d (%v), attempting self-healing correction", task.ID, turn, maxTurns, runErr)
			feedbackPrompt := fmt.Sprintf(
				"The code written to %s failed verification using command %q:\n\n%s\n\nPlease fix the errors and output the entire corrected file content.",
				task.TargetFile, task.VerificationCommand, output,
			)

			correctedCode, genErr := o.executeGeneration(task, llm, feedbackPrompt, sessionHistory)
			if genErr != nil {
				log.Printf("orchestrator: task %s: self-healing generation turn %d failed: %v", task.ID, turn, genErr)
				return
			}

			if o.fileWriter != nil && task.TargetFile != "" {
				if err := o.fileWriter.WriteCodeToFile(task.ProjectPath, task.TargetFile, extractCode(correctedCode)); err != nil {
					logEntry := fmt.Sprintf("failed writing corrected output via %s: %v", providerName, err)
					o.appendTaskLog(task.ID, logEntry)
					_ = o.repo.UpdateStatus(task.ID, domain.StatusFailed)
					o.emit(task.ID, domain.StatusFailed)
					return
				}
			}

			output, runErr = o.commandRunner.Run(ctx, task.ProjectPath, task.VerificationCommand)
			task.VerificationOutput = output
			if runErr == nil {
				logEntry := fmt.Sprintf("verification passed on correction turn %d/%d (%s) — completed via %s at %s", turn, maxTurns, task.VerificationCommand, providerName, time.Now().UTC().Format(time.RFC3339))
				_ = o.repo.Update(task)
				o.appendTaskLog(task.ID, logEntry)
				_ = o.repo.UpdateStatus(task.ID, domain.StatusCompleted)
				o.emit(task.ID, domain.StatusCompleted)
				log.Printf("orchestrator: task %s verification passed on correction turn %d via %s", task.ID, turn, providerName)
				return
			}
		}
	}

	// Verification failed and correction turns exhausted
	logEntry := fmt.Sprintf("verification failed after %d correction turns (%s): %v\nOutput:\n%s", maxTurns, task.VerificationCommand, runErr, output)
	log.Printf("orchestrator: task %s: %s", task.ID, logEntry)
	_ = o.repo.Update(task)
	o.appendTaskLog(task.ID, logEntry)
	_ = o.repo.UpdateStatus(task.ID, domain.StatusFailed)
	o.emit(task.ID, domain.StatusFailed)
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
