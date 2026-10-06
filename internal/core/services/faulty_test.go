package services_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

var errInjected = errors.New("injected failure")

// faultyRepo wraps memRepo and fails the named operations with errInjected.
// Keys are method names ("Save", "GetByID", ...). A nil map injects nothing.
type faultyRepo struct {
	*memRepo
	mu    sync.Mutex
	fail  map[string]bool
	stale []domain.Task // returned (once) by GetStaleProcessing
}

func newFaultyRepo(fail ...string) *faultyRepo {
	f := &faultyRepo{memRepo: newMemRepo(), fail: map[string]bool{}}
	for _, name := range fail {
		f.fail[name] = true
	}
	return f
}

// setStale queues tasks for the next GetStaleProcessing call (goroutine-safe).
func (f *faultyRepo) setStale(tasks ...domain.Task) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stale = tasks
}

func (f *faultyRepo) failing(name string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.fail[name]
}

func (f *faultyRepo) set(name string, on bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fail[name] = on
}

func (f *faultyRepo) Save(t domain.Task) error {
	if f.failing("Save") {
		return errInjected
	}
	return f.memRepo.Save(t)
}

func (f *faultyRepo) GetByID(id string) (domain.Task, error) {
	if f.failing("GetByID") {
		return domain.Task{}, errInjected
	}
	return f.memRepo.GetByID(id)
}

func (f *faultyRepo) GetStaleProcessing(ctx context.Context, d time.Duration) ([]domain.Task, error) {
	if f.failing("GetStaleProcessing") {
		return nil, errInjected
	}
	f.mu.Lock()
	out := f.stale
	f.stale = nil
	f.mu.Unlock()
	if out != nil {
		return out, nil
	}
	return f.memRepo.GetStaleProcessing(ctx, d)
}

func (f *faultyRepo) GetPending() ([]domain.Task, error) {
	if f.failing("GetPending") {
		return nil, errInjected
	}
	return f.memRepo.GetPending()
}

func (f *faultyRepo) GetAll() ([]domain.Task, error) {
	if f.failing("GetAll") {
		return nil, errInjected
	}
	return f.memRepo.GetAll()
}

func (f *faultyRepo) GetByProjectPathAndStatus(p string, s ...domain.TaskStatus) ([]domain.Task, error) {
	if f.failing("GetByProjectPathAndStatus") {
		return nil, errInjected
	}
	return f.memRepo.GetByProjectPathAndStatus(p, s...)
}

func (f *faultyRepo) Update(t domain.Task) error {
	if f.failing("Update") {
		return errInjected
	}
	return f.memRepo.Update(t)
}

func (f *faultyRepo) UpdateStatus(id string, s domain.TaskStatus) error {
	if f.failing("UpdateStatus") {
		return errInjected
	}
	return f.memRepo.UpdateStatus(id, s)
}

func (f *faultyRepo) UpdateLogs(id, logs string) error {
	if f.failing("UpdateLogs") {
		return errInjected
	}
	return f.memRepo.UpdateLogs(id, logs)
}

func (f *faultyRepo) UpdateStatusIfCurrent(id string, from, to domain.TaskStatus) (bool, error) {
	if f.failing("UpdateStatusIfCurrent") {
		return false, errInjected
	}
	return f.memRepo.UpdateStatusIfCurrent(id, from, to)
}

// newFaultyOrch builds an orchestrator with no background workers over a
// faultyRepo and an in-memory AI-session repo.
func newFaultyOrch(t *testing.T, fail ...string) (*services.OrchestratorService, *faultyRepo, *memAISessionRepo) {
	t.Helper()
	return newFaultyOrchWith(t, services.NewDiscoveryService(), fail...)
}

// newFaultyOrchWith is newFaultyOrch with a caller-supplied discovery service.
func newFaultyOrchWith(t *testing.T, discovery *services.DiscoveryService, fail ...string) (*services.OrchestratorService, *faultyRepo, *memAISessionRepo) {
	t.Helper()
	repo := newFaultyRepo(fail...)
	aiRepo := newMemAISessionRepo()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil,
		services.WithDisableBackgroundWorkers())
	orch.SetAISessionRepo(aiRepo)
	t.Cleanup(orch.Stop)
	return orch, repo, aiRepo
}

func seedTask(t *testing.T, r *faultyRepo, id string, status domain.TaskStatus) {
	t.Helper()
	if err := r.memRepo.Save(domain.Task{
		ID: id, ProjectPath: "/proj", Instruction: "i", Status: status,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
}

var bg = context.Background()

// countingFailRepo fails the failAt-th UpdateStatusIfCurrent call (1-based).
type countingFailRepo struct {
	*faultyRepo
	failAt int
	calls  int
}

func (c *countingFailRepo) UpdateStatusIfCurrent(id string, from, to domain.TaskStatus) (bool, error) {
	c.calls++
	if c.calls == c.failAt {
		return false, errInjected
	}
	return c.faultyRepo.UpdateStatusIfCurrent(id, from, to)
}

// raceRepo reports that every conditional update lost the race.
type raceRepo struct{ *faultyRepo }

func (r *raceRepo) UpdateStatusIfCurrent(string, domain.TaskStatus, domain.TaskStatus) (bool, error) {
	return false, nil
}

// newOrchOver builds a no-worker orchestrator over any TaskRepository.
func newOrchOver(t *testing.T, repo ports.TaskRepository) *services.OrchestratorService {
	t.Helper()
	orch := services.NewOrchestrator(services.NewDiscoveryService(), repo, &noopWriter{}, nil,
		services.WithDisableBackgroundWorkers())
	orch.SetAISessionRepo(newMemAISessionRepo())
	t.Cleanup(orch.Stop)
	return orch
}
