package services_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

type memRepo struct {
	mu    sync.Mutex
	tasks map[string]domain.Task
}

func newMemRepo() *memRepo { return &memRepo{tasks: make(map[string]domain.Task)} }

func (r *memRepo) Save(_ context.Context, t domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tasks[t.ID] = t
	return nil
}

func (r *memRepo) GetByID(_ context.Context, id string) (domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tasks[id]
	if !ok {
		return domain.Task{}, domain.ErrNotFound
	}
	return t, nil
}

func (r *memRepo) GetPending(context.Context) ([]domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.Task
	for _, t := range r.tasks {
		if t.Status == domain.StatusQueued || t.Status == domain.StatusProcessing || t.Status == domain.StatusWaiting {
			out = append(out, t)
		}
	}
	return out, nil
}

func (r *memRepo) ClaimNextQueued(context.Context) (domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var (
		claimed domain.Task
		found   bool
	)
	for _, task := range r.tasks {
		if task.Status != domain.StatusQueued {
			continue
		}
		if !found || task.CreatedAt.Before(claimed.CreatedAt) {
			claimed = task
			found = true
		}
	}
	if !found {
		return domain.Task{}, domain.ErrNotFound
	}
	claimed.Status = domain.StatusProcessing
	claimed.UpdatedAt = time.Now()
	r.tasks[claimed.ID] = claimed
	return claimed, nil
}

func (r *memRepo) UpdateStatus(_ context.Context, id string, status domain.TaskStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tasks[id]
	if !ok {
		return errors.New("not found")
	}
	t.Status = status
	r.tasks[id] = t
	return nil
}

func (r *memRepo) UpdateStatusIfCurrent(_ context.Context, id string, from, to domain.TaskStatus) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tasks[id]
	if !ok {
		return false, nil
	}
	if t.Status != from {
		return false, nil
	}
	t.Status = to
	t.UpdatedAt = time.Now()
	r.tasks[id] = t
	return true, nil
}

func (r *memRepo) UpdateLogs(_ context.Context, id, logs string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tasks[id]
	if !ok {
		return errors.New("not found")
	}
	t.Logs = logs
	r.tasks[id] = t
	return nil
}

func (r *memRepo) GetByProjectPath(_ context.Context, projectPath string) ([]domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.Task
	for _, t := range r.tasks {
		if t.ProjectPath == projectPath {
			out = append(out, t)
		}
	}
	return out, nil
}

func (r *memRepo) GetByProjectPathAndStatus(_ context.Context, projectPath string, statuses ...domain.TaskStatus) ([]domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	statusSet := make(map[domain.TaskStatus]bool, len(statuses))
	for _, s := range statuses {
		statusSet[s] = true
	}
	var out []domain.Task
	for _, t := range r.tasks {
		if t.ProjectPath == projectPath && statusSet[t.Status] {
			out = append(out, t)
		}
	}
	return out, nil
}

func (r *memRepo) Update(_ context.Context, t domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Like the SQL repository, Update does not write the log: UpdateLogs owns it.
	if old, ok := r.tasks[t.ID]; ok {
		t.Logs = old.Logs
	}
	r.tasks[t.ID] = t
	return nil
}

func (r *memRepo) GetAll(context.Context) ([]domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []domain.Task{}
	for _, t := range r.tasks {
		out = append(out, t)
	}
	return out, nil
}

func (r *memRepo) GetTasksBySessionID(_ context.Context, sessionID string) ([]domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.Task
	for _, t := range r.tasks {
		if t.AISessionID == sessionID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (r *memRepo) GetStaleProcessing(_ context.Context, _ time.Duration) ([]domain.Task, error) {
	return nil, nil
}

type noopWriter struct{}

func (w *noopWriter) WriteCodeToFile(_, _, _ string) error { return nil }

func (w *noopWriter) ReadContextFiles(_ string, _ []string) (string, error) { return "", nil }

type memProviderConfigRepo struct {
	mu        sync.Mutex
	configs   map[string]domain.ProviderConfig
	saveCalls int
	saveErr   error
}

func newMemProviderConfigRepo() *memProviderConfigRepo {
	return &memProviderConfigRepo{configs: make(map[string]domain.ProviderConfig)}
}

func (r *memProviderConfigRepo) SaveProviderConfig(_ context.Context, cfg domain.ProviderConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saveCalls++
	if r.saveErr != nil {
		return r.saveErr
	}
	r.configs[cfg.ID] = cfg
	return nil
}

func (r *memProviderConfigRepo) ListProviderConfigs(_ context.Context) ([]domain.ProviderConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.ProviderConfig, 0, len(r.configs))
	for _, cfg := range r.configs {
		out = append(out, cfg)
	}
	return out, nil
}

func (r *memProviderConfigRepo) GetProviderConfig(_ context.Context, id string) (domain.ProviderConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cfg, ok := r.configs[id]
	if !ok {
		return domain.ProviderConfig{}, domain.ErrNotFound
	}
	return cfg, nil
}

func (r *memProviderConfigRepo) DeleteProviderConfig(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.configs, id)
	return nil
}

type memSessionRepo struct {
	mu       sync.Mutex
	sessions map[string][]domain.Message // keyed by projectPath
}

func newMemSessionRepo() *memSessionRepo {
	return &memSessionRepo{sessions: make(map[string][]domain.Message)}
}

func (r *memSessionRepo) Save(_ context.Context, s domain.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[s.ProjectPath] = s.Messages
	return nil
}

func (r *memSessionRepo) GetByProjectPath(_ context.Context, p string) (domain.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	msgs, ok := r.sessions[p]
	if !ok {
		return domain.Session{}, domain.ErrNotFound
	}
	return domain.Session{ProjectPath: p, Messages: msgs}, nil
}

func (r *memSessionRepo) AppendMessage(_ context.Context, p string, msg domain.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[p] = append(r.sessions[p], msg)
	return nil
}

// chatTrackingLLM wraps mockLLMClient and records Chat calls.
type chatTrackingLLM struct {
	mu sync.Mutex
	mockLLMClient
	chatCalled int
	lastMsgs   []domain.Message
}

func (c *chatTrackingLLM) Chat(_ context.Context, msgs []domain.Message) (string, error) {
	c.mu.Lock()
	c.chatCalled++
	c.lastMsgs = msgs
	c.mu.Unlock()
	return c.mockLLMClient.Chat(context.Background(), msgs)
}

func waitCompleted(t *testing.T, repo *memRepo, id string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(300 * time.Millisecond)
		saved, _ := repo.GetByID(context.Background(), id)
		if saved.Status == domain.StatusCompleted {
			return
		}
	}
	t.Fatalf("task %s did not reach COMPLETED within %s", id, timeout)
}

// mockScanner is a test double for ports.SystemScanner.
type mockScanner struct {
	results []domain.DiscoveredProvider
	err     error
}

func (m *mockScanner) Scan(_ context.Context) ([]domain.DiscoveredProvider, error) {
	return m.results, m.err
}

// mustPanic calls f and fails the test if f does not panic with a message
// containing wantMsg.
func mustPanic(t *testing.T, wantMsg string, f func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("expected panic containing %q, but no panic occurred", wantMsg)
			return
		}
		msg, ok := r.(string)
		if !ok {
			t.Errorf("expected panic to be a string, got %T: %v", r, r)
			return
		}
		if !strings.Contains(msg, wantMsg) {
			t.Errorf("panic message %q does not contain %q", msg, wantMsg)
		}
	}()
	f()
}

type memAISessionRepo struct {
	mu       sync.Mutex
	sessions map[string]domain.AISession
}

func newMemAISessionRepo() *memAISessionRepo {
	return &memAISessionRepo{sessions: make(map[string]domain.AISession)}
}

func (r *memAISessionRepo) SaveAISession(_ context.Context, s domain.AISession) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[s.ID] = s
	return nil
}

func (r *memAISessionRepo) GetAISessionByID(_ context.Context, id string) (domain.AISession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[id]
	if !ok {
		return domain.AISession{}, domain.ErrNotFound
	}
	return s, nil
}

func (r *memAISessionRepo) GetAISessionByExternalID(_ context.Context, externalID string) (domain.AISession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.sessions {
		if s.ExternalID == externalID {
			return s, nil
		}
	}
	return domain.AISession{}, domain.ErrNotFound
}

func (r *memAISessionRepo) ListAISessions(_ context.Context) ([]domain.AISession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.AISession, 0, len(r.sessions))
	for _, s := range r.sessions {
		out = append(out, s)
	}
	return out, nil
}

func (r *memAISessionRepo) UpdateAISessionStatus(_ context.Context, id string, status domain.AISessionStatus, lastActivity time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[id]
	if !ok {
		return domain.ErrNotFound
	}
	s.Status = status
	s.LastActivity = lastActivity
	r.sessions[id] = s
	return nil
}

func (r *memAISessionRepo) DeleteAISession(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, id)
	return nil
}

func (r *memAISessionRepo) AppendRoutedTaskID(_ context.Context, sessionID string, taskID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[sessionID]
	if !ok {
		return domain.ErrNotFound
	}
	for _, id := range s.RoutedTaskIDs {
		if id == taskID {
			return nil
		}
	}
	s.RoutedTaskIDs = append(s.RoutedTaskIDs, taskID)
	r.sessions[sessionID] = s
	return nil
}

func (r *memAISessionRepo) PurgeDisconnected(_ context.Context, olderThan time.Duration) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cutoff := time.Now().Add(-olderThan)
	n := 0
	for id, s := range r.sessions {
		if s.Status == domain.SessionStatusDisconnected && s.LastActivity.Before(cutoff) {
			delete(r.sessions, id)
			n++
		}
	}
	return n, nil
}

func setupClaimTestOrch(t *testing.T) (*services.OrchestratorService, *memRepo, *memAISessionRepo) {
	t.Helper()
	repo := newMemRepo()
	aiRepo := newMemAISessionRepo()
	discovery := services.NewDiscoveryService()
	orch := services.NewOrchestrator(discovery, repo, &noopWriter{}, nil, services.WithDisableBackgroundWorkers())
	orch.SetAISessionRepo(aiRepo)
	t.Cleanup(orch.Stop)
	return orch, repo, aiRepo
}

func seedActiveSession(t *testing.T, aiRepo *memAISessionRepo, id string) {
	t.Helper()
	err := aiRepo.SaveAISession(context.Background(), domain.AISession{
		ID:        id,
		AgentName: "test-agent",
		Source:    domain.SessionSourceVSCode,
		Status:    domain.SessionStatusActive,
	})
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}
}
