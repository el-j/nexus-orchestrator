package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// The agentic loop turns a task into a bounded conversation in which the model
// may read and write files and run the task's verification command.
//
// Protocol (provider-agnostic, plain text so every LLMClient can speak it): the
// model answers each turn with one or more fenced blocks
//
//	```nexus-tool
//	{"tool":"write_file","path":"internal/x.go","content":"..."}
//	```
//
// and the engine replies with the results of those calls. Tools:
//
//	read_file   {path}            read a file inside the project
//	write_file  {path, content}   create or replace a file inside the project
//	edit_file   {path, old, new}  replace the one occurrence of old with new
//	verify      {}                run the task's verification command
//	finish      {summary}         declare the task done (re-verified if configured)
//
// Sandbox: file access goes through ports.FileWriter, whose path rules reject
// absolute paths, ".." escapes and symlinks leading out of the project. There is
// deliberately no tool that runs arbitrary commands; only the verification
// command the task was submitted with can be executed.
//
// Limits: at most MaxAgentTurns model calls (default defaultAgentTurns, never more
// than hardMaxAgentTurns), at most maxCallsPerTurn tool calls per reply, each
// observation is truncated to maxObservationBytes, and the transcript must stay
// inside the provider's context window. Cancelling the task (or stopping the
// daemon) aborts the loop between and during model calls. A task re-queued after
// a daemon restart starts over from the first turn; files already written stay.
const (
	defaultAgentTurns   = 8
	hardMaxAgentTurns   = 25
	maxCallsPerTurn     = 8
	maxObservationBytes = 16 << 10
)

var toolBlock = regexp.MustCompile("(?s)```nexus-tool\\s*\n(.*?)```")

// agentCall is one tool invocation decoded from a model reply.
type agentCall struct {
	Tool    string `json:"tool"`
	Path    string `json:"path"`
	Content string `json:"content"`
	Old     string `json:"old"`
	New     string `json:"new"`
	Summary string `json:"summary"`
}

func agentSystemPrompt(task domain.Task, canVerify bool) string {
	var sb strings.Builder
	sb.WriteString("You are a software engineering agent working inside the project directory. ")
	sb.WriteString("Act by replying with fenced blocks, each holding one JSON tool call:\n\n")
	sb.WriteString("```nexus-tool\n{\"tool\":\"read_file\",\"path\":\"relative/path\"}\n```\n\n")
	sb.WriteString("Tools: read_file {path}; write_file {path, content} (whole file); ")
	sb.WriteString("edit_file {path, old, new} (replaces the single exact occurrence of old; prefer it for small changes); ")
	if canVerify {
		sb.WriteString("verify {} runs the project's verification command; ")
	}
	sb.WriteString("finish {summary} when the work is complete. ")
	sb.WriteString("Paths are relative to the project root. You receive the results of your calls in the next message. ")
	sb.WriteString("Read before you edit, write complete files, and call finish only when done.")
	if task.TargetFile != "" {
		sb.WriteString("\nPrimary target file: " + task.TargetFile)
	}
	return sb.String()
}

// parseToolCalls extracts the tool calls from a reply. Malformed blocks are
// reported back to the model as errors instead of being silently skipped.
func parseToolCalls(reply string) (calls []agentCall, problems []string) {
	for _, m := range toolBlock.FindAllStringSubmatch(reply, -1) {
		var c agentCall
		if err := json.Unmarshal([]byte(strings.TrimSpace(m[1])), &c); err != nil {
			problems = append(problems, fmt.Sprintf("invalid tool call JSON: %v", err))
			continue
		}
		calls = append(calls, c)
	}
	return calls, problems
}

func truncateObservation(s string) string {
	if len(s) <= maxObservationBytes {
		return s
	}
	return s[:maxObservationBytes] + fmt.Sprintf("\n[... truncated, %d more bytes]", len(s)-maxObservationBytes)
}

func agentTurnLimit(task domain.Task) int {
	switch {
	case task.MaxAgentTurns <= 0:
		return defaultAgentTurns
	case task.MaxAgentTurns > hardMaxAgentTurns:
		return hardMaxAgentTurns
	}
	return task.MaxAgentTurns
}

// agentRun is the mutable state of one loop.
type agentRun struct {
	o         *OrchestratorService
	task      domain.Task
	policy    domain.ProjectPolicy
	written   []string
	wroteTest bool // a test file has been written in this run (tests-first rule)
}

// noteWrite records a successful write for the final report and the tests-first rule.
func (a *agentRun) noteWrite(path string) {
	a.written = append(a.written, path)
	if domain.ClassifyFile(path) == domain.FileTest {
		a.wroteTest = true
	}
}

func (a *agentRun) canVerify() bool {
	return strings.TrimSpace(a.task.VerificationCommand) != "" && a.o.commandRunner != nil
}

// verify runs the verification command and reports whether it passed.
func (a *agentRun) verify(run *runHandle) (string, bool) {
	ctx, cancel := context.WithTimeout(run.ctx, 2*time.Minute)
	defer cancel()
	out, err := a.o.commandRunner.Run(ctx, a.task.ProjectPath, a.task.VerificationCommand)
	a.task.VerificationOutput = out
	if err != nil {
		return fmt.Sprintf("verification FAILED (%v):\n%s", err, truncateObservation(out)), false
	}
	return "verification passed:\n" + truncateObservation(out), true
}

// persist saves the task's verification output without clobbering the log lines
// appended since the task was claimed.
func (a *agentRun) persist() {
	if cur, err := a.o.repo.GetByID(a.o.dbCtx(), a.task.ID); err == nil {
		a.task.Logs = cur.Logs
	}
	_ = a.o.repo.Update(a.o.dbCtx(), a.task)
}

// execute performs one call and returns its observation and whether the model
// finished the task successfully.
func (a *agentRun) execute(run *runHandle, c agentCall) (obs string, done bool) {
	switch c.Tool {
	case "read_file":
		if c.Path == "" {
			return "read_file: path is required", false
		}
		text, err := a.o.fileWriter.ReadContextFiles(a.task.ProjectPath, []string{c.Path})
		if err != nil {
			return "read_file failed: " + err.Error(), false
		}
		return truncateObservation(text), false
	case "write_file":
		if c.Path == "" {
			return "write_file: path is required", false
		}
		if v := policyWriteViolation(a.policy, c.Path, c.Content, a.wroteTest); v != "" {
			return v, false
		}
		if err := a.o.fileWriter.WriteCodeToFile(a.task.ProjectPath, c.Path, c.Content); err != nil {
			return "write_file failed: " + err.Error(), false
		}
		a.noteWrite(c.Path)
		return fmt.Sprintf("wrote %s (%d bytes)", c.Path, len(c.Content)), false
	case "edit_file":
		return a.edit(c), false
	case "verify":
		if !a.canVerify() {
			return "verify: this task has no verification command", false
		}
		out, _ := a.verify(run)
		return out, false
	case "finish":
		if a.canVerify() {
			out, ok := a.verify(run)
			if !ok {
				return "cannot finish yet: " + out, false
			}
		}
		return "finished: " + c.Summary, true
	default:
		return fmt.Sprintf("unknown tool %q (use read_file, write_file, edit_file, verify or finish)", c.Tool), false
	}
}

// runAgentLoop drives the loop to a terminal task status. It returns true like
// the other execution paths (the task was consumed).
func (o *OrchestratorService) runAgentLoop(run *runHandle, task domain.Task, chain []ports.LLMClient, prompt string, policy domain.ProjectPolicy) bool {
	a := &agentRun{o: o, task: task, policy: policy}
	msgs := []domain.Message{
		{Role: domain.RoleSystem, Content: agentSystemPrompt(task, a.canVerify())},
		{Role: domain.RoleUser, Content: prompt},
	}
	limit := agentTurnLimit(task)
	provider := 0

	fail := func(reason string) bool {
		a.persist()
		o.appendTaskLog(task.ID, reason)
		if err := o.repo.UpdateStatus(o.dbCtx(), task.ID, domain.StatusFailed); err != nil {
			log.Printf("orchestrator: update status for task %s: %v", task.ID, err)
		}
		o.emit(task.ID, domain.StatusFailed)
		return true
	}

	for turn := 1; turn <= limit; turn++ {
		if run.ctx.Err() != nil {
			o.finishInterrupted(a.task, run)
			return true
		}

		var reply string
		for {
			llm := chain[provider]
			if _, cooling := o.discovery.Cooldown(llm, time.Now()); cooling {
				if provider+1 < len(chain) {
					o.appendTaskLog(task.ID, fmt.Sprintf("agent turn %d: %s is cooling down; skipped", turn, llm.ProviderName()))
					provider++
					continue
				}
				if o.holdUntilReset(a.task, chain) {
					return true
				}
			}
			if l := llm.ContextLimit(run.ctx); l > 0 && estimateTokens(msgs) > l-o.maxResponseTokens {
				if provider+1 < len(chain) {
					provider++
					continue
				}
				o.appendTaskLog(task.ID, fmt.Sprintf("agent turn %d: transcript exceeds the context window of %s", turn, llm.ProviderName()))
				_ = o.repo.UpdateStatus(o.dbCtx(), task.ID, domain.StatusTooLarge)
				o.emit(task.ID, domain.StatusTooLarge)
				return true
			}
			out, err := nonEmpty(o.liveChat(run.ctx, task.ID, llm, msgs))
			if run.ctx.Err() != nil {
				o.finishInterrupted(a.task, run)
				return true
			}
			if err != nil {
				o.appendTaskLog(task.ID, fmt.Sprintf("agent turn %d: provider %s failed: %v", turn, llm.ProviderName(), err))
				o.noteProviderError(llm, err)
				if provider+1 < len(chain) {
					provider++
					continue
				}
				if o.holdUntilReset(a.task, chain) {
					return true
				}
				return fail(fmt.Sprintf("agent turn %d: all providers failed (last: %v)", turn, err))
			}
			reply = out
			break
		}
		msgs = append(msgs, domain.Message{Role: domain.RoleAssistant, Content: reply})

		calls, problems := parseToolCalls(reply)
		var results []string
		results = append(results, problems...)
		if len(calls) == 0 && len(problems) == 0 {
			results = append(results, "No tool call found. Reply with ```nexus-tool blocks, and call finish when done.")
		}
		if len(calls) > maxCallsPerTurn {
			results = append(results, fmt.Sprintf("only the first %d tool calls of a reply are run", maxCallsPerTurn))
			calls = calls[:maxCallsPerTurn]
		}
		for _, c := range calls {
			obs, done := a.execute(run, c)
			if run.ctx.Err() != nil {
				o.finishInterrupted(a.task, run)
				return true
			}
			o.appendTaskLog(task.ID, fmt.Sprintf("agent turn %d: %s", turn, c.Tool+" "+c.Path))
			if done {
				a.persist()
				o.appendTaskLog(task.ID, fmt.Sprintf("%s — completed via %s after %d turn(s), %d file(s) written at %s",
					obs, chain[provider].ProviderName(), turn, len(a.written), time.Now().UTC().Format(time.RFC3339)))
				if err := o.repo.UpdateStatus(o.dbCtx(), task.ID, domain.StatusCompleted); err != nil {
					log.Printf("orchestrator: update status for task %s: %v", task.ID, err)
				}
				o.emit(task.ID, domain.StatusCompleted)
				return true
			}
			results = append(results, fmt.Sprintf("[%s %s]\n%s", c.Tool, c.Path, obs))
		}
		msgs = append(msgs, domain.Message{Role: domain.RoleUser, Content: strings.Join(results, "\n\n")})
	}
	return fail(fmt.Sprintf("agent stopped after %d turns without finishing", limit))
}

// edit replaces the single occurrence of c.Old in c.Path with c.New. Zero or
// several matches are refused so a vague edit never lands in the wrong place.
func (a *agentRun) edit(c agentCall) string {
	if c.Path == "" || c.Old == "" {
		return "edit_file: path and old are required"
	}
	text, err := a.o.fileWriter.ReadContextFiles(a.task.ProjectPath, []string{c.Path})
	if err != nil {
		return "edit_file failed: " + err.Error()
	}
	// ReadContextFiles frames the content with a header line and a trailing newline.
	if rest, ok := strings.CutPrefix(text, "// --- "+c.Path+" ---\n"); ok {
		text = strings.TrimSuffix(rest, "\n")
	}
	switch n := strings.Count(text, c.Old); {
	case n == 0:
		return "edit_file: old text not found in " + c.Path
	case n > 1:
		return fmt.Sprintf("edit_file: old text occurs %d times in %s; include more context so it is unique", n, c.Path)
	}
	updated := strings.Replace(text, c.Old, c.New, 1)
	if v := policyWriteViolation(a.policy, c.Path, updated, a.wroteTest); v != "" {
		return v
	}
	if err := a.o.fileWriter.WriteCodeToFile(a.task.ProjectPath, c.Path, updated); err != nil {
		return "edit_file failed: " + err.Error()
	}
	a.noteWrite(c.Path)
	return fmt.Sprintf("edited %s (%d bytes)", c.Path, len(updated))
}
