package services

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// Limits for what the engine puts in front of a model and keeps afterwards.
const (
	// MaxHistoryMessages is how many of the newest session messages a task is sent.
	// Older ones stay stored but cost no tokens.
	MaxHistoryMessages = 20
	// MaxTaskMemories is how many finished-task entries are kept per project.
	MaxTaskMemories = 30
	// TaskMemorySource marks the knowledge entries the engine writes about finished tasks.
	TaskMemorySource = "nexus:task"

	briefingTokens = 800  // upper bound for the project briefing in a prompt
	minBriefingCtx = 4000 // models with a smaller context window get the rules only
)

// briefer is the optional brain capability of producing a briefing without the rules block.
type briefer interface {
	GetProjectBrief(ctx context.Context, projectPath string, maxTokens int) (string, error)
}

// WithBrain gives the engine the project knowledge store: tasks are sent with a
// briefing of the project and finished tasks are remembered (when the project's
// policy asks for it).
func WithBrain(b ports.BrainService) Option {
	return func(o *OrchestratorService) { o.brain = b }
}

// WithBrain sets the project knowledge store after construction (the entry points
// build the brain after the engine).
func (o *OrchestratorService) WithBrain(b ports.BrainService) {
	o.mu.Lock()
	o.brain = b
	o.mu.Unlock()
}

func (o *OrchestratorService) brainService() ports.BrainService {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.brain
}

// applyProjectRules loads the project's policy and returns the task and prompt the
// model should actually get: the policy's verification command as the default, and
// the briefing plus the rules ahead of the instruction. A broken policy is an error.
func (o *OrchestratorService) applyProjectRules(ctx context.Context, task domain.Task, chain []ports.LLMClient, prompt string) (domain.Task, domain.ProjectPolicy, string, error) {
	policy, hasPolicy, err := LoadProjectPolicy(task.ProjectPath)
	if err != nil {
		return task, policy, prompt, err
	}
	if hasPolicy && strings.TrimSpace(task.VerificationCommand) == "" && policy.Verify != "" {
		task.VerificationCommand = policy.Verify
	}

	var parts []string
	brief := o.briefing(ctx, task, chain)
	if brief != "" {
		parts = append(parts, strings.TrimSpace(brief))
	}
	if hasPolicy {
		if rules := renderPolicy(policy); strings.Count(rules, "\n") > 1 {
			parts = append(parts, strings.TrimSpace(rules))
		}
	}
	if focused := o.focusedKnowledge(ctx, task, chain, brief); focused != "" {
		parts = append(parts, strings.TrimSpace(focused))
	}
	if len(parts) == 0 {
		return task, policy, prompt, nil
	}
	return task, policy, strings.Join(parts, "\n\n") + "\n\n## Task\n" + prompt, nil
}

// focusedKnowledge queries the project brain for knowledge entries matching the task instruction
// that are not already covered by the static briefing.
func (o *OrchestratorService) focusedKnowledge(ctx context.Context, task domain.Task, chain []ports.LLMClient, brief string) string {
	b := o.brainService()
	if b == nil || strings.TrimSpace(task.ProjectPath) == "" || strings.TrimSpace(task.Instruction) == "" {
		return ""
	}
	if len(chain) > 0 {
		if limit := chain[0].ContextLimit(ctx); limit > 0 && limit < minBriefingCtx {
			return ""
		}
	}
	resp, err := b.GetFocusedContext(ctx, domain.ContextQuery{
		ProjectPath: task.ProjectPath,
		Question:    task.Instruction,
		MaxTokens:   400,
	})
	if err != nil || len(resp.Sections) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, s := range resp.Sections {
		if brief != "" && strings.Contains(brief, strings.TrimSpace(s.Content)) {
			continue
		}
		if sb.Len() == 0 {
			sb.WriteString("## Relevant Project Knowledge\n")
		}
		fmt.Fprintf(&sb, "- **%s (%s):** %s\n", s.Topic, s.Kind, strings.TrimSpace(s.Content))
	}
	return sb.String()
}

// briefing returns the project briefing sized for the first model of the chain, or "".
func (o *OrchestratorService) briefing(ctx context.Context, task domain.Task, chain []ports.LLMClient) string {
	b := o.brainService()
	if b == nil || strings.TrimSpace(task.ProjectPath) == "" {
		return ""
	}
	budget := briefingTokens
	if len(chain) > 0 {
		if limit := chain[0].ContextLimit(ctx); limit > 0 {
			if limit < minBriefingCtx {
				return ""
			}
			budget = min(budget, limit/8)
		}
	}
	var (
		text string
		err  error
	)
	if br, ok := b.(briefer); ok {
		text, err = br.GetProjectBrief(ctx, task.ProjectPath, budget)
	} else {
		text, err = b.GetOnboardingContext(ctx, task.ProjectPath, budget)
	}
	if err != nil {
		log.Printf("orchestrator: briefing for %s: %v", task.ProjectPath, err)
		return ""
	}
	return text
}

// trimHistory keeps the newest max messages and makes sure the slice starts with a user turn.
func trimHistory(msgs []domain.Message, max int) []domain.Message {
	if len(msgs) <= max {
		return msgs
	}
	msgs = msgs[len(msgs)-max:]
	for len(msgs) > 0 && msgs[0].Role != domain.RoleUser {
		msgs = msgs[1:]
	}
	return msgs
}

// recordOutcome stores a short entry about a finished task in the project knowledge,
// so the next session (any agent, any model) knows what was tried and how it ended.
// It only acts when the project's policy sets remember, and never fails the task.
func (o *OrchestratorService) recordOutcome(taskID string, status domain.TaskStatus) {
	if status != domain.StatusCompleted && status != domain.StatusFailed {
		return
	}
	b := o.brainService()
	if b == nil {
		return
	}
	task, err := o.repo.GetByID(o.dbCtx(), taskID)
	if err != nil {
		return
	}
	policy, ok, err := LoadProjectPolicy(task.ProjectPath)
	if err != nil || !ok || !policy.Remember {
		return
	}
	outcome := "Completed"
	if status == domain.StatusFailed {
		outcome = "Failed"
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s: %s", outcome, truncateUTF8(strings.Join(strings.Fields(task.Instruction), " "), 200))
	if task.TargetFile != "" {
		fmt.Fprintf(&sb, " | target %s", task.TargetFile)
	}
	if task.VerificationCommand != "" {
		fmt.Fprintf(&sb, " | verified with `%s`", task.VerificationCommand)
	}
	if status == domain.StatusFailed && task.Logs != "" {
		fmt.Fprintf(&sb, " | last log: %s", truncateUTF8(lastLine(task.Logs), 200))
	}
	ctx := o.dbCtx()
	if _, err := b.IngestKnowledge(ctx, domain.ProjectKnowledge{
		ProjectPath: task.ProjectPath,
		Kind:        domain.KnowledgeLearning,
		Topic:       "Task " + shortID(task.ID),
		Content:     sb.String(),
		Source:      TaskMemorySource,
	}); err != nil {
		log.Printf("orchestrator: remember task %s: %v", task.ID, err)
		return
	}
	o.pruneTaskMemories(ctx, b, task.ProjectPath)
}

// pruneTaskMemories deletes the oldest task entries beyond MaxTaskMemories.
func (o *OrchestratorService) pruneTaskMemories(ctx context.Context, b ports.BrainService, projectPath string) {
	entries, err := b.ListKnowledge(ctx, projectPath, string(domain.KnowledgeLearning))
	if err != nil {
		return
	}
	var mine []domain.ProjectKnowledge
	for _, e := range entries {
		if e.Source == TaskMemorySource {
			mine = append(mine, e)
		}
	}
	if len(mine) <= MaxTaskMemories {
		return
	}
	sort.SliceStable(mine, func(i, j int) bool { return mine[i].CreatedAt.Before(mine[j].CreatedAt) })
	for _, e := range mine[:len(mine)-MaxTaskMemories] {
		if err := b.DeleteKnowledge(ctx, e.ID); err != nil {
			log.Printf("orchestrator: prune task memory %s: %v", e.ID, err)
		}
	}
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func lastLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.LastIndexByte(s, '\n'); i >= 0 {
		return s[i+1:]
	}
	return s
}
