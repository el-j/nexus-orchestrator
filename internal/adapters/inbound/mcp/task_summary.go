package mcp

import (
	"nexus-orchestrator/internal/core/domain"
)

// taskSummary provides a token-efficient, compact view of a task.
// It omits bulky payloads (like large multi-KB logs, context files, and verification traces)
// to save context window tokens when external LLMs call listing tools.
type taskSummary struct {
	ID                 string             `json:"id"`
	Status             domain.TaskStatus  `json:"status"`
	TargetFile         string             `json:"targetFile,omitempty"`
	InstructionSnippet string             `json:"instructionSnippet,omitempty"`
	Role               string             `json:"role,omitempty"`
	Priority           int                `json:"priority,omitempty"`
	Command            domain.CommandType `json:"command,omitempty"`
	HasLogs            bool               `json:"hasLogs"`
	LogsPreview        string             `json:"logsPreview,omitempty"`
}

// toTaskSummary converts a domain.Task into a token-efficient taskSummary.
func toTaskSummary(t domain.Task) taskSummary {
	snip := t.Instruction
	if len(snip) > 120 {
		snip = snip[:117] + "..."
	}
	var preview string
	hasLogs := t.Logs != ""
	if hasLogs {
		if len(t.Logs) > 80 {
			preview = t.Logs[:77] + "..."
		} else {
			preview = t.Logs
		}
	}
	return taskSummary{
		ID:                 t.ID,
		Status:             t.Status,
		TargetFile:         t.TargetFile,
		InstructionSnippet: snip,
		Role:               t.Role,
		Priority:           t.Priority,
		Command:            t.Command,
		HasLogs:            hasLogs,
		LogsPreview:        preview,
	}
}

// toTaskSummaries converts a slice of domain.Task into taskSummary items.
func toTaskSummaries(tasks []domain.Task) []taskSummary {
	out := make([]taskSummary, len(tasks))
	for i, t := range tasks {
		out[i] = toTaskSummary(t)
	}
	return out
}

// applyTaskLimit clamps items to limit if limit is positive.
func applyTaskLimit[T any](items []T, limit int) []T {
	if limit > 0 && limit < len(items) {
		return items[:limit]
	}
	return items
}
