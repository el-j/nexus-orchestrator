package domain

import "time"

// MessageRole identifies the speaker in a conversation turn.
type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
)

// Message is a single turn in a conversation with an LLM.
// Role must be one of the typed MessageRole constants (RoleUser, RoleAssistant).
type Message struct {
	Role      MessageRole `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

// Session holds the per-project conversation history used to ensure session
// isolation: tasks from the same ProjectPath share history, while tasks from
// different projects never mix context.
//
// ProjectPath must be normalized with filepath.Clean before use as an
// isolation key so that equivalent paths compare equal.
type Session struct {
	ID          string    `json:"id"`
	ProjectPath string    `json:"projectPath"`
	Messages    []Message `json:"messages"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
