// Package domain contains the pure domain types for nexusOrchestrator.
// This file defines the brain / context-intelligence types.
package domain

import "time"

// KnowledgeKind classifies the type of a ProjectKnowledge entry.
type KnowledgeKind string

const (
	// KnowledgeArchitecture describes structural and design decisions.
	KnowledgeArchitecture KnowledgeKind = "architecture"
	// KnowledgeConvention describes coding rules, style, and patterns.
	KnowledgeConvention KnowledgeKind = "convention"
	// KnowledgeFileMap describes the project file layout and path-to-purpose mapping.
	KnowledgeFileMap KnowledgeKind = "file_map"
	// KnowledgeLearning records lessons, discovered pitfalls, and things to avoid.
	KnowledgeLearning KnowledgeKind = "learning"
	// KnowledgeGlossary defines terms, abbreviations, and domain vocabulary.
	KnowledgeGlossary KnowledgeKind = "glossary"
	// KnowledgeTaskSummary captures the outcome and context of a completed task.
	KnowledgeTaskSummary KnowledgeKind = "task_summary"
)

// ProjectKnowledge is a single structured knowledge entry for a project.
type ProjectKnowledge struct {
	ID             string        `json:"id"`
	ProjectPath    string        `json:"projectPath"`
	Kind           KnowledgeKind `json:"kind"`
	Topic          string        `json:"topic"`
	Content        string        `json:"content"`
	Source         string        `json:"source,omitempty"`
	TokenCount     int           `json:"tokenCount"`
	RelevanceScore float64       `json:"relevanceScore"` // 0.0–1.0, default 0.5
	CreatedAt      time.Time     `json:"createdAt"`
	UpdatedAt      time.Time     `json:"updatedAt"`
}

// ContextQuery is the input to BrainService context assembly methods.
type ContextQuery struct {
	ProjectPath string `json:"projectPath"`
	TaskID      string `json:"taskId,omitempty"`
	Question    string `json:"question,omitempty"`
	FocusArea   string `json:"focusArea,omitempty"`
	MaxTokens   int    `json:"maxTokens"`
}

// ContextSection is a single slice of context assembled from a knowledge entry.
type ContextSection struct {
	Kind    KnowledgeKind `json:"kind"`
	Topic   string        `json:"topic"`
	Content string        `json:"content"`
	Tokens  int           `json:"tokens"`
}

// ContextResponse is the token-budget-constrained context returned to an agent.
type ContextResponse struct {
	ProjectPath    string           `json:"projectPath"`
	Sections       []ContextSection `json:"sections"`
	SuggestedFiles []string         `json:"suggestedFiles,omitempty"`
	TokensUsed     int              `json:"tokensUsed"`
	TokenBudget    int              `json:"tokenBudget"`
	Truncated      bool             `json:"truncated"`
}

// BrainStatus summarises the initialisation state of the brain for a project.
type BrainStatus struct {
	ProjectPath string         `json:"projectPath"`
	Initialized bool           `json:"initialized"`
	EntryCount  int            `json:"entryCount"`
	KindCounts  map[string]int `json:"kindCounts"`
	TotalTokens int            `json:"totalTokens"`
	LastUpdated time.Time      `json:"lastUpdated,omitempty"`
}
