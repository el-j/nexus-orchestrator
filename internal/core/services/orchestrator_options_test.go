// Package services — white-box tests for functional options.
// These tests live in package services (not services_test) so they can inspect
// unexported fields that are set by the With* option functions.
package services

import (
	"errors"
	"sync"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// --- minimal stubs (cannot reuse the services_test stubs from the same dir) --

type optRepo struct {
	mu    sync.Mutex
	tasks map[string]domain.Task
}

func newOptRepo() *optRepo { return &optRepo{tasks: make(map[string]domain.Task)} }

func (r *optRepo) Save(t domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tasks[t.ID] = t
	return nil
}
func (r *optRepo) GetByID(id string) (domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tasks[id]
	if !ok {
		return domain.Task{}, errors.New("not found")
	}
	return t, nil
}
func (r *optRepo) GetPending() ([]domain.Task, error)               { return nil, nil }
func (r *optRepo) UpdateStatus(_ string, _ domain.TaskStatus) error { return nil }
func (r *optRepo) UpdateLogs(_, _ string) error                     { return nil }
func (r *optRepo) GetByProjectPath(_ string) ([]domain.Task, error) { return nil, nil }
func (r *optRepo) GetByProjectPathAndStatus(_ string, _ ...domain.TaskStatus) ([]domain.Task, error) {
	return nil, nil
}
func (r *optRepo) Update(t domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tasks[t.ID] = t
	return nil
}
func (r *optRepo) GetAll() ([]domain.Task, error) { return nil, nil }

type optWriter struct{}

func (w *optWriter) WriteCodeToFile(_, _, _ string) error                  { return nil }
func (w *optWriter) ReadContextFiles(_ string, _ []string) (string, error) { return "", nil }

// compile-time interface checks
var _ ports.TaskRepository = (*optRepo)(nil)
var _ ports.FileWriter = (*optWriter)(nil)

// --- option tests -------------------------------------------------------------

func TestOrchestratorOptions_WithMaxRetries(t *testing.T) {
	svc := NewOrchestrator(NewDiscoveryService(), newOptRepo(), &optWriter{}, nil, WithMaxRetries(5))
	defer svc.Stop()
	if svc.maxRetries != 5 {
		t.Fatalf("expected maxRetries=5, got %d", svc.maxRetries)
	}
}

func TestOrchestratorOptions_WithMaxResponseTokens(t *testing.T) {
	svc := NewOrchestrator(NewDiscoveryService(), newOptRepo(), &optWriter{}, nil, WithMaxResponseTokens(1024))
	defer svc.Stop()
	if svc.maxResponseTokens != 1024 {
		t.Fatalf("expected maxResponseTokens=1024, got %d", svc.maxResponseTokens)
	}
}

func TestOrchestratorOptions_Defaults(t *testing.T) {
	svc := NewOrchestrator(NewDiscoveryService(), newOptRepo(), &optWriter{}, nil)
	defer svc.Stop()
	if svc.maxRetries != 3 {
		t.Errorf("default maxRetries: want 3, got %d", svc.maxRetries)
	}
	if svc.maxResponseTokens != 512 {
		t.Errorf("default maxResponseTokens: want 512, got %d", svc.maxResponseTokens)
	}
}
