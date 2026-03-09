package wailsbind

import (
	"nexus-ai/internal/core/domain"
	"nexus-ai/internal/core/ports"
)

// NexusApp exposes orchestrator methods to the Wails/JS frontend.
// All exported methods are automatically bound to the frontend by Wails.
type NexusApp struct {
	orch ports.Orchestrator
}

// New creates a NexusApp bound to the given orchestrator.
func New(orch ports.Orchestrator) *NexusApp {
	return &NexusApp{orch: orch}
}

// SubmitTask forwards a task submission from the UI to the orchestrator.
func (n *NexusApp) SubmitTask(task domain.Task) (string, error) {
	return n.orch.SubmitTask(task)
}

// GetQueue returns the current task queue for display in the dashboard.
func (n *NexusApp) GetQueue() ([]domain.Task, error) {
	return n.orch.GetQueue()
}

// CancelTask cancels a pending task by ID.
func (n *NexusApp) CancelTask(id string) error {
	return n.orch.CancelTask(id)
}
