package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

var conventionKeywords = []string{
	"invariant", "convention", "pattern", "rule", "architectural decision", "contract",
}

// ExtractSummaryLearning constructs a high-level outcome entry for a completed or failed task.
func ExtractSummaryLearning(task domain.Task, status domain.TaskStatus) domain.ProjectKnowledge {
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

	return domain.ProjectKnowledge{
		ProjectPath:    task.ProjectPath,
		Kind:           domain.KnowledgeLearning,
		Topic:          "Task " + shortID(task.ID),
		Content:        sb.String(),
		Source:         TaskMemorySource,
		RelevanceScore: 0.7,
	}
}

// ExtractConventionEntries extracts verified conventions or invariants from a successful task.
func ExtractConventionEntries(task domain.Task) []domain.ProjectKnowledge {
	instructionLower := strings.ToLower(task.Instruction)
	matched := false
	for _, kw := range conventionKeywords {
		if strings.Contains(instructionLower, kw) {
			matched = true
			break
		}
	}
	if !matched {
		return nil
	}

	topic := "General"
	if task.TargetFile != "" {
		topic = filepath.Base(task.TargetFile)
	} else if task.Role != "" {
		topic = task.Role
	}

	rule := strings.TrimSpace(task.Instruction)
	if !strings.HasPrefix(rule, "* INVARIANT:") {
		rule = "* INVARIANT: " + rule
	}

	return []domain.ProjectKnowledge{
		{
			ProjectPath:    task.ProjectPath,
			Kind:           domain.KnowledgeConvention,
			Topic:          topic,
			Content:        truncateUTF8(rule, 300),
			Source:         "task-learning",
			RelevanceScore: 0.85,
		},
	}
}

// ExtractFileMapEntries creates or reconciles file map entries when a task targets a file.
func ExtractFileMapEntries(task domain.Task) []domain.ProjectKnowledge {
	if task.TargetFile == "" {
		return nil
	}

	purpose := strings.TrimSpace(task.Instruction)
	if purpose == "" {
		purpose = "Project source file"
	}
	content := fmt.Sprintf("%s - %s", task.TargetFile, truncateUTF8(purpose, 150))

	return []domain.ProjectKnowledge{
		{
			ProjectPath:    task.ProjectPath,
			Kind:           domain.KnowledgeFileMap,
			Topic:          filepath.Base(task.TargetFile),
			Content:        content,
			Source:         "task-filemap",
			RelevanceScore: 0.8,
		},
	}
}

// ExtractFailureLearnings records lessons and pitfalls from a failed task to prevent repeated errors.
func ExtractFailureLearnings(task domain.Task) []domain.ProjectKnowledge {
	if task.Logs == "" {
		return nil
	}
	logSnippet := truncateUTF8(lastLine(task.Logs), 200)
	if logSnippet == "" {
		return nil
	}

	target := task.TargetFile
	if target == "" {
		target = shortID(task.ID)
	}

	content := fmt.Sprintf("* AVOID: failure on %s: %s", target, logSnippet)
	return []domain.ProjectKnowledge{
		{
			ProjectPath:    task.ProjectPath,
			Kind:           domain.KnowledgeLearning,
			Topic:          "Pitfall " + shortID(task.ID),
			Content:        content,
			Source:         "task-failure",
			RelevanceScore: 0.75,
		},
	}
}

// DigestTaskOutcome processes a settled task, extracting summaries, conventions, file maps,
// and failure pitfalls, ingesting them directly into the project knowledge repository.
func DigestTaskOutcome(ctx context.Context, brain ports.BrainService, task domain.Task, status domain.TaskStatus) []domain.ProjectKnowledge {
	if brain == nil {
		return nil
	}
	if status != domain.StatusCompleted && status != domain.StatusFailed {
		return nil
	}

	var toIngest []domain.ProjectKnowledge
	toIngest = append(toIngest, ExtractSummaryLearning(task, status))

	switch status {
	case domain.StatusCompleted:
		toIngest = append(toIngest, ExtractConventionEntries(task)...)
		toIngest = append(toIngest, ExtractFileMapEntries(task)...)
	case domain.StatusFailed:
		toIngest = append(toIngest, ExtractFailureLearnings(task)...)
	}

	var ingested []domain.ProjectKnowledge
	for _, entry := range toIngest {
		saved, err := brain.IngestKnowledge(ctx, entry)
		if err == nil {
			ingested = append(ingested, saved)
		}
	}

	return ingested
}
