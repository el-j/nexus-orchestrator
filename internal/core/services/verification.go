package services

import (
	"context"
	"fmt"
	"log"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"strings"
	"time"
)

// writeAndVerifyTaskOutput writes the generated code to disk, runs verification commands if configured,
// and initiates a self-healing correction loop if verification fails. It returns the final code produced.
func (o *OrchestratorService) writeAndVerifyTaskOutput(run *runHandle, task domain.Task, code string, llm ports.LLMClient, sessionHistory []domain.Message) string {
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
			if err2 := o.repo.UpdateLogs(o.dbCtx(), task.ID, logEntry); err2 != nil {
				log.Printf("orchestrator: update logs for task %s: %v", task.ID, err2)
			}
			if err2 := o.repo.UpdateStatus(o.dbCtx(), task.ID, domain.StatusFailed); err2 != nil {
				log.Printf("orchestrator: update status for task %s: %v", task.ID, err2)
			}
			o.emit(task.ID, domain.StatusFailed)
			return code
		}
	}

	// If no verification command is configured or command runner is nil, complete normally.
	if strings.TrimSpace(task.VerificationCommand) == "" || o.commandRunner == nil {
		logEntry := fmt.Sprintf("completed via %s at %s", providerName, time.Now().UTC().Format(time.RFC3339))
		o.appendTaskLog(task.ID, logEntry)
		if err := o.repo.UpdateStatus(o.dbCtx(), task.ID, domain.StatusCompleted); err != nil {
			log.Printf("orchestrator: update status for task %s: %v", task.ID, err)
		}
		o.emit(task.ID, domain.StatusCompleted)
		log.Printf("orchestrator: task %s completed via %s", task.ID, providerName)
		return code
	}

	// Run verification command
	ctx, cancel := context.WithTimeout(run.ctx, 2*time.Minute)
	defer cancel()

	output, runErr := o.commandRunner.Run(ctx, task.ProjectPath, task.VerificationCommand)
	task.VerificationOutput = output
	if run.ctx.Err() != nil {
		o.finishInterrupted(task, run)
		return ""
	}

	if runErr == nil {
		// Verification passed on first attempt
		logEntry := fmt.Sprintf("verification passed (%s) — completed via %s at %s", task.VerificationCommand, providerName, time.Now().UTC().Format(time.RFC3339))
		_ = o.repo.Update(o.dbCtx(), task)
		o.appendTaskLog(task.ID, logEntry)
		if err := o.repo.UpdateStatus(o.dbCtx(), task.ID, domain.StatusCompleted); err != nil {
			log.Printf("orchestrator: update status for task %s: %v", task.ID, err)
		}
		o.emit(task.ID, domain.StatusCompleted)
		log.Printf("orchestrator: task %s verification passed, completed via %s", task.ID, providerName)
		return code
	}

	// Verification failed — attempt self-healing correction loop if LLM is available
	maxTurns := task.MaxCorrectionTurns
	if maxTurns <= 0 {
		maxTurns = 2
	}

	latest := code // the most recent output the model produced; advances every turn
	if llm != nil {
		current := llm
		for turn := 1; turn <= maxTurns; turn++ {
			if policy, ok, _ := LoadProjectPolicy(task.ProjectPath); ok {
				o.enforceBudgets(run.chain, policy)
			}
			if next := o.stepUp(run, current); next != current {
				o.appendTaskLog(task.ID, fmt.Sprintf("verification failed on %s/%s: asking %s/%s (next rung up)",
					current.ProviderName(), current.ActiveModel(), next.ProviderName(), next.ActiveModel()))
				current, providerName = next, next.ProviderName()
			}
			log.Printf("orchestrator: task %s: verification failed on turn %d/%d (%v), attempting self-healing correction", task.ID, turn, maxTurns, runErr)
			feedbackPrompt := correctionPrompt(task, latest, output)

			correctedCode, genErr := o.executeGeneration(run, task, current, feedbackPrompt, sessionHistory)
			if genErr != nil {
				log.Printf("orchestrator: task %s: self-healing generation turn %d failed: %v", task.ID, turn, genErr)
				return ""
			}
			latest = correctedCode

			if o.fileWriter != nil && task.TargetFile != "" {
				if err := o.fileWriter.WriteCodeToFile(task.ProjectPath, task.TargetFile, extractCode(correctedCode)); err != nil {
					logEntry := fmt.Sprintf("failed writing corrected output via %s: %v", providerName, err)
					o.appendTaskLog(task.ID, logEntry)
					_ = o.repo.UpdateStatus(o.dbCtx(), task.ID, domain.StatusFailed)
					o.emit(task.ID, domain.StatusFailed)
					return latest
				}
			}

			output, runErr = o.commandRunner.Run(ctx, task.ProjectPath, task.VerificationCommand)
			task.VerificationOutput = output
			if run.ctx.Err() != nil {
				o.finishInterrupted(task, run)
				return ""
			}
			if runErr == nil {
				logEntry := fmt.Sprintf("verification passed on correction turn %d/%d (%s) — completed via %s at %s", turn, maxTurns, task.VerificationCommand, providerName, time.Now().UTC().Format(time.RFC3339))
				_ = o.repo.Update(o.dbCtx(), task)
				o.appendTaskLog(task.ID, logEntry)
				_ = o.repo.UpdateStatus(o.dbCtx(), task.ID, domain.StatusCompleted)
				o.emit(task.ID, domain.StatusCompleted)
				log.Printf("orchestrator: task %s verification passed on correction turn %d via %s", task.ID, turn, providerName)
				return latest
			}
		}
	}

	// Verification failed and correction turns exhausted
	logEntry := fmt.Sprintf("verification failed after %d correction turns (%s): %v\nOutput:\n%s", maxTurns, task.VerificationCommand, runErr, output)
	log.Printf("orchestrator: task %s: %s", task.ID, logEntry)
	_ = o.repo.Update(o.dbCtx(), task)
	o.appendTaskLog(task.ID, logEntry)
	_ = o.repo.UpdateStatus(o.dbCtx(), task.ID, domain.StatusFailed)
	o.emit(task.ID, domain.StatusFailed)
	return latest
}

// correctionPrompt builds the self-healing prompt for a failed verification. It
// restates the original instruction and the model's latest output so that a
// stateless GenerateCode call (no session history) has everything it needs to
// produce a corrected file.
func correctionPrompt(task domain.Task, latestOutput, verificationOutput string) string {
	return fmt.Sprintf(
		"Original task:\n%s\n\nYour latest output for %s:\n%s\n\n"+
			"It failed verification using command %q:\n\n%s\n\n"+
			"Please fix the errors and output the entire corrected file content.",
		task.Instruction, task.TargetFile, extractCode(latestOutput), task.VerificationCommand, truncateObservation(verificationOutput),
	)
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
