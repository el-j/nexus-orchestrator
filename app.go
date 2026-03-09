package main

import (
	"context"
	"fmt"

	"nexus-ai/internal/core/domain"
	"nexus-ai/internal/core/ports"
)

// App is the Wails application struct. Its exported methods are bound to the
// JavaScript frontend.
type App struct {
	ctx          context.Context
	orchestrator ports.Orchestrator
}

// NewApp creates a new App instance.
func NewApp(orch ports.Orchestrator) *App {
	return &App{orchestrator: orch}
}

// startup is called by Wails when the application starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// SubmitTask forwards a task from the frontend to the orchestrator.
func (a *App) SubmitTask(task domain.Task) (string, error) {
	return a.orchestrator.SubmitTask(task)
}

// GetQueue returns all pending tasks for the dashboard.
func (a *App) GetQueue() ([]domain.Task, error) {
	return a.orchestrator.GetQueue()
}

// CancelTask cancels a queued task.
func (a *App) CancelTask(id string) error {
	return a.orchestrator.CancelTask(id)
}

// Greet is the default Wails example method — kept for scaffolding compatibility.
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello, %s! NexusAI is running.", name)
}
