package services_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

// skipUnlessPermissionsApply skips tests that rely on a file mode stopping the current user.
func skipUnlessPermissionsApply(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("file modes do not stop this user")
	}
}

func TestPolicyFiles_ReportAnUnreadableFileAndAFolderThatCannotBeWritten(t *testing.T) {
	skipUnlessPermissionsApply(t)

	t.Run("a policy file that cannot be read", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, ".nexus", "policy.json")
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(`{}`), 0o000); err != nil {
			t.Fatal(err)
		}
		if _, _, err := services.LoadProjectPolicy(dir); err == nil {
			t.Error("an unreadable policy must be an error, not 'no policy'")
		}
	})
	t.Run("the .nexus folder cannot be created", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Chmod(dir, 0o500); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chmod(dir, 0o700) }()
		if created, err := services.EnsureProjectPolicy(dir); err == nil || created {
			t.Errorf("created=%v err=%v, want a refusal", created, err)
		}
	})
	t.Run("the policy file cannot be written", func(t *testing.T) {
		dir := t.TempDir()
		nexus := filepath.Join(dir, ".nexus")
		if err := os.Mkdir(nexus, 0o500); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chmod(nexus, 0o700) }()
		if created, err := services.EnsureProjectPolicy(dir); err == nil || created {
			t.Errorf("created=%v err=%v, want a refusal", created, err)
		}
	})
}

func TestIngestFromFile_ReportsAFileItCannotRead(t *testing.T) {
	skipUnlessPermissionsApply(t)
	dir, md := project(t)
	if err := os.Chmod(md, 0o000); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(md, 0o600) }()
	if _, err := ingest(t, dir, md); err == nil {
		t.Error("an unreadable file must be an error")
	}
}

// failThenBlockRunner fails the first verification, then holds the second until its context ends.
type failThenBlockRunner struct {
	calls   int
	started chan struct{}
}

func (r *failThenBlockRunner) Run(ctx context.Context, _, _ string) (string, error) {
	r.calls++
	if r.calls == 1 {
		return "FAIL: expected 1", errors.New("exit status 1")
	}
	close(r.started)
	<-ctx.Done()
	return "", ctx.Err()
}

func TestCancelTask_AbortsTheVerificationOfACorrection(t *testing.T) {
	code := "```go\npackage a\n```"
	runner := &failThenBlockRunner{started: make(chan struct{})}
	llm := &scriptLLM{name: "p", alive: true, limit: 100000, replies: []string{code, code}}
	e := newEngine(t, &contextWriter{}, nil, runner, llm)
	id := e.queue(domain.Task{Instruction: "c", TargetFile: "a.go", VerificationCommand: "t"})
	done := runInBackground(e)
	waitFor(t, runner.started, "the corrected code to be verified")
	if err := e.orch.CancelTask(id); err != nil {
		t.Fatal(err)
	}
	waitFor(t, done, "the worker to settle")
	if got := e.status(id).Status; got != domain.StatusCancelled {
		t.Errorf("status = %s, want CANCELLED", got)
	}
}

func TestDiscovery_SkipsADeadProviderWhenLookingForAModelAndLogsAModelListThatFails(t *testing.T) {
	dead := &scriptLLM{name: "dead", alive: false, models: []string{"m"}}
	broken := &scriptLLM{name: "broken", alive: true, modelErr: errInjected}
	good := &scriptLLM{name: "good", alive: true, models: []string{"m"}}
	disc := services.NewDiscoveryService(dead, broken, good)
	found, err := disc.FindForModel("m", "")
	if err != nil || found.ProviderName() != "good" {
		t.Fatalf("FindForModel = %v, %v; want the live provider that lists m", found, err)
	}
	for _, p := range disc.ListProviders() {
		if p.Name == "broken" && (!p.Active || len(p.Models) != 0) {
			t.Errorf("a provider whose model list fails stays active with no models: %+v", p)
		}
	}
}

func TestActivityService_ReportsASessionItCannotRemove(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := filepath.Join(home, "app")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o750); err != nil {
		t.Fatal(err)
	}
	sessions := newFaultyAIRepo()
	_ = sessions.memAISessionRepo.SaveAISession(bg, domain.AISession{ID: "sub", Source: "discovered", AgentName: "claude", ProjectPath: filepath.Join(repo, "sub"), Status: domain.SessionStatusIdle, LastActivity: time.Now()})
	sessions.set("Delete", errInjected)
	services.NewActivityService(&actMemActivityRepo{}, sessions, &mockReader{name: "r"}).PollNow()
	if left, _ := sessions.ListAISessions(bg); len(left) != 1 {
		t.Errorf("a session that cannot be deleted must stay: %+v", left)
	}
}
