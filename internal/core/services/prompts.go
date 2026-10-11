package services

import (
	"errors"
	"fmt"
	"log"
	"nexus-orchestrator/internal/core/domain"
	"strings"
	"time"
)

// prepareChatPrompt loads context files and session history once for the task.
func (o *OrchestratorService) prepareChatPrompt(task domain.Task) (string, []domain.Message, error) {
	prompt := task.Instruction
	if len(task.ContextFiles) > 0 && o.fileWriter != nil {
		ctx, err := o.fileWriter.ReadContextFiles(task.ProjectPath, task.ContextFiles)
		if err != nil {
			logEntry := fmt.Sprintf("failed reading context files: %v", err)
			log.Printf("orchestrator: task %s: %s", task.ID, logEntry)
			if err2 := o.repo.UpdateLogs(o.dbCtx(), task.ID, logEntry); err2 != nil {
				log.Printf("orchestrator: update logs for task %s: %v", task.ID, err2)
			}
			if err2 := o.repo.UpdateStatus(o.dbCtx(), task.ID, domain.StatusFailed); err2 != nil {
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
		sess, err := o.sessionRepo.GetByProjectPath(o.dbCtx(), task.ProjectPath)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			logEntry := fmt.Sprintf("failed loading session history: %v", err)
			log.Printf("orchestrator: task %s: %s", task.ID, logEntry)
			if err2 := o.repo.UpdateLogs(o.dbCtx(), task.ID, logEntry); err2 != nil {
				log.Printf("orchestrator: update logs for task %s: %v", task.ID, err2)
			}
			if err2 := o.repo.UpdateStatus(o.dbCtx(), task.ID, domain.StatusFailed); err2 != nil {
				log.Printf("orchestrator: update status for task %s: %v", task.ID, err2)
			}
			o.emit(task.ID, domain.StatusFailed)
			return "", nil, fmt.Errorf("orchestrator: load session: %w", err)
		}
		sessionHistory = sess.Messages
	}

	return prompt, sessionHistory, nil
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
