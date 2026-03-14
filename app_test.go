package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

type mockOrchestrator struct {
	submitTaskCalled   bool
	getTaskCalled      bool
	getQueueCalled     bool
	cancelTaskCalled   bool
	getBacklogCalled   bool
	lastTaskArg        domain.Task
	lastIDArg          string
	lastProjectPathArg string
	errReturn          error
	taskReturn         domain.Task
	tasksReturn        []domain.Task
	idReturn           string
}

func (m *mockOrchestrator) SubmitTask(t domain.Task) (string, error) {
	m.submitTaskCalled = true
	m.lastTaskArg = t
	return m.idReturn, m.errReturn
}

func (m *mockOrchestrator) GetTask(id string) (domain.Task, error) {
	m.getTaskCalled = true
	m.lastIDArg = id
	return m.taskReturn, m.errReturn
}

func (m *mockOrchestrator) GetQueue() ([]domain.Task, error) {
	m.getQueueCalled = true
	return m.tasksReturn, m.errReturn
}

func (m *mockOrchestrator) CancelTask(id string) error {
	m.cancelTaskCalled = true
	m.lastIDArg = id
	return m.errReturn
}

func (m *mockOrchestrator) GetBacklog(p string) ([]domain.Task, error) {
	m.getBacklogCalled = true
	m.lastProjectPathArg = p
	return m.tasksReturn, m.errReturn
}

func (m *mockOrchestrator) CreateDraft(t domain.Task) (string, error) {
	return m.idReturn, m.errReturn
}
func (m *mockOrchestrator) GetProviders() ([]ports.ProviderInfo, error)         { return nil, nil }
func (m *mockOrchestrator) RegisterCloudProvider(_ domain.ProviderConfig) error { return nil }
func (m *mockOrchestrator) RemoveProvider(_ string) error                       { return nil }
func (m *mockOrchestrator) GetProviderModels(_ string) ([]string, error)        { return nil, nil }
func (m *mockOrchestrator) PromoteTask(_ string) error                          { return nil }
func (m *mockOrchestrator) UpdateTask(_ string, _ domain.Task) (domain.Task, error) {
	return domain.Task{}, nil
}
func (m *mockOrchestrator) GetDiscoveredProviders() ([]domain.DiscoveredProvider, error) {
	return nil, nil
}
func (m *mockOrchestrator) TriggerScan(_ context.Context) ([]domain.DiscoveredProvider, error) {
	return nil, nil
}
func (m *mockOrchestrator) PromoteProvider(_ context.Context, _ string) error { return nil }
func (m *mockOrchestrator) AddProviderConfig(_ context.Context, c domain.ProviderConfig) (domain.ProviderConfig, error) {
	return c, nil
}
func (m *mockOrchestrator) UpdateProviderConfig(_ context.Context, c domain.ProviderConfig) (domain.ProviderConfig, error) {
	return c, nil
}
func (m *mockOrchestrator) RemoveProviderConfig(_ context.Context, _ string) error { return nil }
func (m *mockOrchestrator) ListProviderConfigs(_ context.Context) ([]domain.ProviderConfig, error) {
	return nil, nil
}
func (m *mockOrchestrator) RegisterAISession(_ context.Context, s domain.AISession) (domain.AISession, error) {
	return s, nil
}
func (m *mockOrchestrator) ListAISessions(_ context.Context) ([]domain.AISession, error) {
	return nil, nil
}
func (m *mockOrchestrator) DeregisterAISession(_ context.Context, _ string) error { return nil }
func (m *mockOrchestrator) HeartbeatAISession(_ context.Context, _ string) error  { return nil }
func (m *mockOrchestrator) ClaimTask(_ context.Context, _ string, _ string) (domain.Task, error) {
	return domain.Task{}, nil
}
func (m *mockOrchestrator) UpdateTaskStatus(_ context.Context, _ string, _ string, _ domain.TaskStatus, _ string) (domain.Task, error) {
	return domain.Task{}, nil
}
func (m *mockOrchestrator) PurgeDisconnectedSessions(_ context.Context) (int, error) {
	return 0, nil
}
func (m *mockOrchestrator) GetAllTasks() ([]domain.Task, error) {
	return m.tasksReturn, m.errReturn
}
func (m *mockOrchestrator) GetDiscoveredAgents(_ context.Context) ([]domain.DiscoveredAgent, error) {
	return nil, nil
}
func (m *mockOrchestrator) DelegateToNexus(_ context.Context, _ string) (string, error) {
	return "", nil
}

var _ ports.Orchestrator = (*mockOrchestrator)(nil)

func TestApp_SubmitTask_Delegates(t *testing.T) {
	mock := &mockOrchestrator{idReturn: "task-42"}
	app := NewApp(mock, "127.0.0.1:63987")
	task := domain.Task{
		ProjectPath: "/projects/alpha",
		TargetFile:  "main.go",
		Instruction: "write tests",
	}
	id, err := app.SubmitTask(task)
	if err != nil {
		t.Fatalf("SubmitTask: unexpected error: %v", err)
	}
	if !mock.submitTaskCalled {
		t.Error("expected orchestrator.SubmitTask to be called")
	}
	if id != "task-42" {
		t.Errorf("returned ID: got %q, want %q", id, "task-42")
	}
	if mock.lastTaskArg.ProjectPath != task.ProjectPath {
		t.Errorf("task.ProjectPath: got %q, want %q", mock.lastTaskArg.ProjectPath, task.ProjectPath)
	}
}

func TestApp_GetTask_Delegates(t *testing.T) {
	want := domain.Task{ID: "t-99", Status: domain.StatusQueued, CreatedAt: time.Now()}
	mock := &mockOrchestrator{taskReturn: want}
	app := NewApp(mock, "127.0.0.1:63987")
	got, err := app.GetTask("t-99")
	if err != nil {
		t.Fatalf("GetTask: unexpected error: %v", err)
	}
	if !mock.getTaskCalled {
		t.Error("expected orchestrator.GetTask to be called")
	}
	if mock.lastIDArg != "t-99" {
		t.Errorf("ID argument: got %q, want %q", mock.lastIDArg, "t-99")
	}
	if got.ID != want.ID {
		t.Errorf("returned task ID: got %q, want %q", got.ID, want.ID)
	}
}

func TestApp_GetQueue_Delegates(t *testing.T) {
	tasks := []domain.Task{
		{ID: "q-1", Status: domain.StatusQueued},
		{ID: "q-2", Status: domain.StatusProcessing},
	}
	mock := &mockOrchestrator{tasksReturn: tasks}
	app := NewApp(mock, "127.0.0.1:63987")
	got, err := app.GetQueue()
	if err != nil {
		t.Fatalf("GetQueue: unexpected error: %v", err)
	}
	if !mock.getQueueCalled {
		t.Error("expected orchestrator.GetQueue to be called")
	}
	if len(got) != len(tasks) {
		t.Errorf("queue length: got %d, want %d", len(got), len(tasks))
	}
}

func TestApp_CancelTask_Delegates(t *testing.T) {
	mock := &mockOrchestrator{}
	app := NewApp(mock, "127.0.0.1:63987")
	if err := app.CancelTask("t-cancel-1"); err != nil {
		t.Fatalf("CancelTask: unexpected error: %v", err)
	}
	if !mock.cancelTaskCalled {
		t.Error("expected orchestrator.CancelTask to be called")
	}
	if mock.lastIDArg != "t-cancel-1" {
		t.Errorf("ID argument: got %q, want %q", mock.lastIDArg, "t-cancel-1")
	}
}

func TestApp_GetBacklog_Delegates(t *testing.T) {
	drafts := []domain.Task{{ID: "d-1", Status: domain.StatusDraft}}
	mock := &mockOrchestrator{tasksReturn: drafts}
	app := NewApp(mock, "127.0.0.1:63987")
	got, err := app.GetBacklog("/projects/beta")
	if err != nil {
		t.Fatalf("GetBacklog: unexpected error: %v", err)
	}
	if !mock.getBacklogCalled {
		t.Error("expected orchestrator.GetBacklog to be called")
	}
	if mock.lastProjectPathArg != "/projects/beta" {
		t.Errorf("path: got %q, want %q", mock.lastProjectPathArg, "/projects/beta")
	}
	if len(got) != len(drafts) {
		t.Errorf("backlog length: got %d, want %d", len(got), len(drafts))
	}
}

func TestApp_ErrorPropagation(t *testing.T) {
	sentinel := errors.New("orchestrator offline")
	mock := &mockOrchestrator{errReturn: sentinel}
	app := NewApp(mock, "127.0.0.1:63987")
	if _, err := app.SubmitTask(domain.Task{}); !errors.Is(err, sentinel) {
		t.Errorf("SubmitTask error: got %v, want %v", err, sentinel)
	}
	if _, err := app.GetTask("any"); !errors.Is(err, sentinel) {
		t.Errorf("GetTask error: got %v, want %v", err, sentinel)
	}
	if _, err := app.GetQueue(); !errors.Is(err, sentinel) {
		t.Errorf("GetQueue error: got %v, want %v", err, sentinel)
	}
	if err := app.CancelTask("any"); !errors.Is(err, sentinel) {
		t.Errorf("CancelTask error: got %v, want %v", err, sentinel)
	}
	if _, err := app.GetBacklog("/any"); !errors.Is(err, sentinel) {
		t.Errorf("GetBacklog error: got %v, want %v", err, sentinel)
	}
}
