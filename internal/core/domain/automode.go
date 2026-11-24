package domain

import "time"

// AutoPhase describes the current stage of a Project Auto-Mode run.
type AutoPhase string

const (
	// AutoPhaseSyncing initializes or refreshes the project brain and RAG context.
	AutoPhaseSyncing AutoPhase = "syncing"
	// AutoPhasePlanning decomposes the goal into dependency-ordered tasks or promotes backlog work.
	AutoPhasePlanning AutoPhase = "planning"
	// AutoPhaseExecuting monitors the queued and running tasks until all settle.
	AutoPhaseExecuting AutoPhase = "executing"
	// AutoPhaseVerifying runs the project-wide verification check after task execution.
	AutoPhaseVerifying AutoPhase = "verifying"
	// AutoPhaseRepairing dispatches a targeted repair task when final verification fails.
	AutoPhaseRepairing AutoPhase = "repairing"
	// AutoPhaseLearning digests task outcomes back into the project knowledge base.
	AutoPhaseLearning AutoPhase = "learning"
	// AutoPhaseCompleted indicates the auto-mode run succeeded end to end.
	AutoPhaseCompleted AutoPhase = "completed"
	// AutoPhaseFailed indicates the auto-mode run stopped with unrecovered task or verify failures.
	AutoPhaseFailed AutoPhase = "failed"
	// AutoPhaseStopped indicates the user or daemon shutdown cancelled the auto-mode run.
	AutoPhaseStopped AutoPhase = "stopped"
)

// IsTerminal reports whether the phase is a final state.
func (p AutoPhase) IsTerminal() bool {
	return p == AutoPhaseCompleted || p == AutoPhaseFailed || p == AutoPhaseStopped
}

// AutoModeRequest configures a Project Auto-Mode run.
type AutoModeRequest struct {
	ProjectPath        string `json:"projectPath"`
	Goal               string `json:"goal,omitempty"`
	VerifyCommand      string `json:"verifyCommand,omitempty"`
	MaxTasks           int    `json:"maxTasks,omitempty"`
	MaxAgentTurns      int    `json:"maxAgentTurns,omitempty"`
	MaxRepairAttempts  int    `json:"maxRepairAttempts,omitempty"`
	DelegateToSessions bool   `json:"delegateToSessions,omitempty"`
}

// AutoModeRun captures the live status and outcome of a Project Auto-Mode run.
type AutoModeRun struct {
	ID             string    `json:"id"`
	ProjectPath    string    `json:"projectPath"`
	Goal           string    `json:"goal,omitempty"`
	Phase          AutoPhase `json:"phase"`
	TaskIDs        []string  `json:"taskIds"`
	CompletedTasks int       `json:"completedTasks"`
	FailedTasks    int       `json:"failedTasks"`
	RepairAttempts int       `json:"repairAttempts,omitempty"`
	LearningsSaved int       `json:"learningsSaved,omitempty"`
	Summary        string    `json:"summary,omitempty"`
	Error          string    `json:"error,omitempty"`
	StartedAt      time.Time `json:"startedAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	FinishedAt     time.Time `json:"finishedAt,omitzero"`
}
