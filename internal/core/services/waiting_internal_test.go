package services

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// A resume timer that was already running when the engine was stopped must do nothing: the engine is
// going away and its repository may be closed under it.
func TestHoldTimer_DoesNotResumeAnythingOnceTheEngineIsStopping(t *testing.T) {
	old := waitGrace
	waitGrace = 0
	defer func() { waitGrace = old }()

	repo, err := repo_sqlite.New(filepath.Join(t.TempDir(), "w.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	llm := &deadStub{name: "p"}
	o := NewOrchestrator(NewDiscoveryService(llm), repo, nopWriter{}, nil, WithDisableBackgroundWorkers())
	task := domain.Task{ID: "w", ProjectPath: "/p", Instruction: "i", Status: domain.StatusProcessing, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.Save(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	o.NoteProviderError(llm, &domain.ProviderError{Provider: "p", Kind: domain.ErrKindRateLimited, Status: 429, ResetAt: time.Now().Add(40 * time.Millisecond), Msg: "limited"})

	if !o.holdUntilReset(task, []ports.LLMClient{llm}) {
		t.Fatal("the task was not held")
	}
	o.waitTimerMu.Lock()               // the timer's callback will wait here when it fires (in about 40 ms)
	time.Sleep(150 * time.Millisecond) // the timer fired and is now blocked on the lock
	close(o.stopCh)                    // the engine starts stopping while the callback waits
	o.waitTimerMu.Unlock()
	time.Sleep(100 * time.Millisecond)
	if got, _ := repo.GetByID(context.Background(), "w"); got.Status != domain.StatusWaiting {
		t.Errorf("status = %s: a timer must not queue tasks of a stopping engine", got.Status)
	}
	o.stopOnce.Do(func() {}) // stopCh is already closed
}

// nopWriter is a file writer that writes and reads nothing.
type nopWriter struct{}

func (nopWriter) WriteCodeToFile(string, string, string) error      { return nil }
func (nopWriter) ReadContextFiles(string, []string) (string, error) { return "", nil }
