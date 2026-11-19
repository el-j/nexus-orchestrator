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
	providerName := llm.ProviderName() // the engine always passes the provider that answered

	if o.fileWriter != nil && task.TargetFile != "" {
		if err := o.fileWriter.WriteCodeToFile(task.ProjectPath, task.TargetFile, extractCode(code)); err != nil {
			o.failOutputWrite(task, providerName, err)
			return code
		}
	}

	// If no verification command is configured or command runner is nil, complete normally.
	if strings.TrimSpace(task.VerificationCommand) == "" || o.commandRunner == nil {
		o.markCompleted(task, fmt.Sprintf("completed via %s at %s", providerName, time.Now().UTC().Format(time.RFC3339)),
			fmt.Sprintf("task %s completed via %s", task.ID, providerName))
		return code
	}

	ctx, cancel := context.WithTimeout(run.ctx, 2*time.Minute)
	defer cancel()

	output, runErr := o.commandRunner.Run(ctx, task.ProjectPath, task.VerificationCommand)
	task.VerificationOutput = output
	if run.ctx.Err() != nil {
		o.finishInterrupted(task, run)
		return ""
	}

	if runErr == nil { // verification passed on the first attempt
		_ = o.repo.Update(o.dbCtx(), task)
		o.markCompleted(task, fmt.Sprintf("verification passed (%s) — completed via %s at %s", task.VerificationCommand, providerName, time.Now().UTC().Format(time.RFC3339)),
			fmt.Sprintf("task %s verification passed, completed via %s", task.ID, providerName))
		return code
	}

	maxTurns := task.MaxCorrectionTurns
	if maxTurns <= 0 {
		maxTurns = 2
	}
	h := &healer{o: o, rh: run, ctx: ctx, task: task, history: sessionHistory, current: llm, latest: code, output: output, runErr: runErr, maxTurns: maxTurns}
	return h.heal()
}

// failOutputWrite records that the generated code could not be written and fails the task.
func (o *OrchestratorService) failOutputWrite(task domain.Task, providerName string, err error) {
	log.Printf("orchestrator: write file for task %s: %v", task.ID, err)
	if err2 := o.repo.UpdateLogs(o.dbCtx(), task.ID, fmt.Sprintf("failed writing output via %s: %v", providerName, err)); err2 != nil {
		log.Printf("orchestrator: update logs for task %s: %v", task.ID, err2)
	}
	if err2 := o.repo.UpdateStatus(o.dbCtx(), task.ID, domain.StatusFailed); err2 != nil {
		log.Printf("orchestrator: update status for task %s: %v", task.ID, err2)
	}
	o.emit(task.ID, domain.StatusFailed)
}

// markCompleted appends the entry to the task log, sets the status to completed and announces it.
func (o *OrchestratorService) markCompleted(task domain.Task, logEntry, logLine string) {
	o.appendTaskLog(task.ID, logEntry)
	if err := o.repo.UpdateStatus(o.dbCtx(), task.ID, domain.StatusCompleted); err != nil {
		log.Printf("orchestrator: update status for task %s: %v", task.ID, err)
	}
	o.emit(task.ID, domain.StatusCompleted)
	log.Printf("orchestrator: %s", logLine)
}

// healer is the self-healing loop of one task whose verification failed: it asks a model for a corrected
// file, writes it and runs the verification again, for at most maxTurns turns.
type healer struct {
	o        *OrchestratorService
	rh       *runHandle
	ctx      context.Context
	task     domain.Task
	history  []domain.Message
	current  ports.LLMClient // the provider asked for the next correction; may move up the ladder
	latest   string          // the most recent output the model produced; advances every turn
	output   string          // the output of the last verification run
	runErr   error           // the error of the last verification run
	maxTurns int
}

// heal runs the correction turns and returns the final output; "" means the task was interrupted, put
// on hold or failed during generation, and its status is already set.
func (h *healer) heal() string {
	o, task := h.o, h.task
	for turn := 1; turn <= h.maxTurns; turn++ {
		if h.chooseProvider() {
			return ""
		}
		log.Printf("orchestrator: task %s: verification failed on turn %d/%d (%v), attempting self-healing correction", task.ID, turn, h.maxTurns, h.runErr)
		corrected, genErr := o.executeGeneration(h.rh, task, h.current, correctionPrompt(task, h.latest, h.output), h.history)
		if genErr != nil {
			log.Printf("orchestrator: task %s: self-healing generation turn %d failed: %v", task.ID, turn, genErr)
			return ""
		}
		h.latest = corrected

		if o.fileWriter != nil && task.TargetFile != "" {
			if err := o.fileWriter.WriteCodeToFile(task.ProjectPath, task.TargetFile, extractCode(corrected)); err != nil {
				o.appendTaskLog(task.ID, fmt.Sprintf("failed writing corrected output via %s: %v", h.current.ProviderName(), err))
				_ = o.repo.UpdateStatus(o.dbCtx(), task.ID, domain.StatusFailed)
				o.emit(task.ID, domain.StatusFailed)
				return h.latest
			}
		}

		h.output, h.runErr = o.commandRunner.Run(h.ctx, task.ProjectPath, task.VerificationCommand)
		h.task.VerificationOutput = h.output
		task = h.task
		if h.rh.ctx.Err() != nil {
			o.finishInterrupted(task, h.rh)
			return ""
		}
		if h.runErr == nil {
			_ = o.repo.Update(o.dbCtx(), task)
			o.markCompleted(task, fmt.Sprintf("verification passed on correction turn %d/%d (%s) — completed via %s at %s", turn, h.maxTurns, task.VerificationCommand, h.current.ProviderName(), time.Now().UTC().Format(time.RFC3339)),
				fmt.Sprintf("task %s verification passed on correction turn %d via %s", task.ID, turn, h.current.ProviderName()))
			return h.latest
		}
	}

	// Verification failed and correction turns exhausted
	logEntry := fmt.Sprintf("verification failed after %d correction turns (%s): %v\nOutput:\n%s", h.maxTurns, task.VerificationCommand, h.runErr, h.output)
	log.Printf("orchestrator: task %s: %s", task.ID, logEntry)
	_ = o.repo.Update(o.dbCtx(), task)
	o.appendTaskLog(task.ID, logEntry)
	_ = o.repo.UpdateStatus(o.dbCtx(), task.ID, domain.StatusFailed)
	o.emit(task.ID, domain.StatusFailed)
	return h.latest
}

// chooseProvider picks the provider for the next turn: another one when the current one is cooling down
// (or the wait for a reset when none is left), and the next rung up the model ladder. It reports true
// when the task was put on hold and the loop must stop.
func (h *healer) chooseProvider() (hold bool) {
	o, task, run := h.o, h.task, h.rh
	if policy, ok, _ := LoadProjectPolicy(task.ProjectPath); ok {
		o.enforceBudgets(run.chain, policy)
	}
	if cd, cooling := o.discovery.Cooldown(h.current, time.Now()); cooling {
		// The provider hit a limit or used up its budget during this task: ask another one,
		// or wait for the reset when none is left, instead of spending past the limit.
		next := o.firstAvailable(run.chain)
		if next == nil {
			if o.holdUntilReset(task, run.chain) {
				return true
			}
		} else {
			o.appendTaskLog(task.ID, fmt.Sprintf("%s is cooling down (%s): the correction goes to %s/%s",
				h.current.ProviderName(), cd.reason, next.ProviderName(), next.ActiveModel()))
			h.current = next
		}
	}
	if next := o.stepUp(run, h.current); next != h.current {
		o.appendTaskLog(task.ID, fmt.Sprintf("verification failed on %s/%s: asking %s/%s (next rung up)",
			h.current.ProviderName(), h.current.ActiveModel(), next.ProviderName(), next.ActiveModel()))
		h.current = next
	}
	return false
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
