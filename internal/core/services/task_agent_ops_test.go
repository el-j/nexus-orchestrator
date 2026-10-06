package services_test

import (
	"errors"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

func TestClaimTask_ErrorBranches(t *testing.T) {
	t.Run("no session repo", func(t *testing.T) {
		orch := services.NewOrchestrator(services.NewDiscoveryService(), newMemRepo(), &noopWriter{}, nil,
			services.WithDisableBackgroundWorkers())
		defer orch.Stop()
		if _, err := orch.ClaimTask(bg, "t", "s"); err == nil || !strings.Contains(err.Error(), "no session repo") {
			t.Errorf("got %v", err)
		}
	})

	t.Run("unknown task", func(t *testing.T) {
		orch, _, aiRepo := newFaultyOrch(t)
		seedActiveSession(t, aiRepo, "s")
		if _, err := orch.ClaimTask(bg, "ghost", "s"); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("got %v", err)
		}
	})

	t.Run("project isolation", func(t *testing.T) {
		orch, repo, aiRepo := newFaultyOrch(t)
		if err := aiRepo.SaveAISession(bg, domain.AISession{
			ID: "s", Status: domain.SessionStatusActive, ProjectPath: "/other",
		}); err != nil {
			t.Fatal(err)
		}
		seedTask(t, repo, "t", domain.StatusQueued) // project "/proj"
		if _, err := orch.ClaimTask(bg, "t", "s"); err == nil || !strings.Contains(err.Error(), "does not match task project") {
			t.Errorf("got %v", err)
		}
		if got, _ := repo.GetByID("t"); got.Status != domain.StatusQueued {
			t.Errorf("rejected claim must not mutate the task, status=%s", got.Status)
		}
	})

	t.Run("not queued", func(t *testing.T) {
		orch, repo, aiRepo := newFaultyOrch(t)
		seedActiveSession(t, aiRepo, "s")
		seedTask(t, repo, "t", domain.StatusProcessing)
		if _, err := orch.ClaimTask(bg, "t", "s"); err == nil || !strings.Contains(err.Error(), "is not QUEUED") {
			t.Errorf("got %v", err)
		}
	})

	t.Run("conditional update failure", func(t *testing.T) {
		orch, repo, aiRepo := newFaultyOrch(t)
		seedActiveSession(t, aiRepo, "s")
		seedTask(t, repo, "t", domain.StatusQueued)
		repo.set("UpdateStatusIfCurrent", true)
		if _, err := orch.ClaimTask(bg, "t", "s"); !errors.Is(err, errInjected) {
			t.Errorf("got %v", err)
		}
	})

	t.Run("session binding persist failure", func(t *testing.T) {
		orch, repo, aiRepo := newFaultyOrch(t)
		seedActiveSession(t, aiRepo, "s")
		seedTask(t, repo, "t", domain.StatusQueued)
		repo.set("Update", true)
		if _, err := orch.ClaimTask(bg, "t", "s"); !errors.Is(err, errInjected) {
			t.Errorf("got %v", err)
		}
	})

	t.Run("broadcasts processing event", func(t *testing.T) {
		orch, repo, aiRepo := newFaultyOrch(t)
		b := &capturingBroadcaster{}
		orch.SetBroadcaster(b)
		seedActiveSession(t, aiRepo, "s")
		seedTask(t, repo, "t", domain.StatusQueued)
		if _, err := orch.ClaimTask(bg, "t", "s"); err != nil {
			t.Fatal(err)
		}
		if n := len(b.events); n != 1 || b.events[0].Type != ports.EventTaskProcessing || b.events[0].Status != domain.StatusProcessing {
			t.Errorf("events = %+v", b.events)
		}
	})
}

func TestUpdateTaskStatus_Branches(t *testing.T) {
	setup := func(t *testing.T, status domain.TaskStatus) (*services.OrchestratorService, *faultyRepo, *capturingBroadcaster) {
		orch, repo, _ := newFaultyOrch(t)
		b := &capturingBroadcaster{}
		orch.SetBroadcaster(b)
		if err := repo.memRepo.Save(domain.Task{ID: "t", ProjectPath: "/proj", Status: status, AISessionID: "s"}); err != nil {
			t.Fatal(err)
		}
		return orch, repo, b
	}

	t.Run("completed with logs broadcasts completion", func(t *testing.T) {
		orch, repo, b := setup(t, domain.StatusProcessing)
		got, err := orch.UpdateTaskStatus(bg, "t", "s", domain.StatusCompleted, "all good")
		if err != nil || got.Status != domain.StatusCompleted || got.Logs != "all good" {
			t.Fatalf("%+v %v", got, err)
		}
		if stored, _ := repo.GetByID("t"); stored.Logs != "all good" {
			t.Errorf("logs not persisted: %+v", stored)
		}
		if len(b.events) != 1 || b.events[0].Type != ports.EventTaskCompleted || b.events[0].Status != domain.StatusCompleted {
			t.Errorf("events = %+v", b.events)
		}
	})

	t.Run("failed without logs broadcasts failure", func(t *testing.T) {
		orch, _, b := setup(t, domain.StatusProcessing)
		if _, err := orch.UpdateTaskStatus(bg, "t", "s", domain.StatusFailed, ""); err != nil {
			t.Fatal(err)
		}
		if len(b.events) != 1 || b.events[0].Type != ports.EventTaskFailed || b.events[0].Status != domain.StatusFailed {
			t.Errorf("events = %+v", b.events)
		}
	})

	t.Run("rejects non-terminal target status", func(t *testing.T) {
		orch, _, _ := setup(t, domain.StatusProcessing)
		if _, err := orch.UpdateTaskStatus(bg, "t", "s", domain.StatusQueued, ""); err == nil || !strings.Contains(err.Error(), "invalid target status") {
			t.Errorf("got %v", err)
		}
	})

	t.Run("unknown task", func(t *testing.T) {
		orch, _, _ := setup(t, domain.StatusProcessing)
		if _, err := orch.UpdateTaskStatus(bg, "ghost", "s", domain.StatusCompleted, ""); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("got %v", err)
		}
	})

	t.Run("wrong owner", func(t *testing.T) {
		orch, _, _ := setup(t, domain.StatusProcessing)
		if _, err := orch.UpdateTaskStatus(bg, "t", "intruder", domain.StatusCompleted, ""); err == nil || !strings.Contains(err.Error(), "does not own task") {
			t.Errorf("got %v", err)
		}
	})

	t.Run("not processing", func(t *testing.T) {
		orch, _, _ := setup(t, domain.StatusQueued)
		if _, err := orch.UpdateTaskStatus(bg, "t", "s", domain.StatusCompleted, ""); err == nil || !strings.Contains(err.Error(), "not PROCESSING") {
			t.Errorf("got %v", err)
		}
	})

	t.Run("status write failure", func(t *testing.T) {
		orch, repo, _ := setup(t, domain.StatusProcessing)
		repo.set("UpdateStatus", true)
		if _, err := orch.UpdateTaskStatus(bg, "t", "s", domain.StatusCompleted, ""); !errors.Is(err, errInjected) {
			t.Errorf("got %v", err)
		}
	})

	t.Run("log write failure is tolerated", func(t *testing.T) {
		orch, repo, _ := setup(t, domain.StatusProcessing)
		repo.set("UpdateLogs", true)
		got, err := orch.UpdateTaskStatus(bg, "t", "s", domain.StatusCompleted, "x")
		if err != nil || got.Status != domain.StatusCompleted {
			t.Errorf("%+v %v", got, err)
		}
	})
}

func TestHeartbeatTask_Branches(t *testing.T) {
	orch, repo, _ := newFaultyOrch(t)
	if err := repo.memRepo.Save(domain.Task{ID: "t", Status: domain.StatusProcessing, AISessionID: "s"}); err != nil {
		t.Fatal(err)
	}
	if err := orch.HeartbeatTask(bg, "t", "s"); err != nil {
		t.Fatalf("happy path: %v", err)
	}
	if err := orch.HeartbeatTask(bg, "ghost", "s"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("unknown: %v", err)
	}
	if err := orch.HeartbeatTask(bg, "t", "other"); err == nil || !strings.Contains(err.Error(), "not claimed by session") {
		t.Errorf("wrong session: %v", err)
	}
	seedTask(t, repo, "q", domain.StatusQueued)
	if err := orch.HeartbeatTask(bg, "q", "s"); err == nil || !strings.Contains(err.Error(), "not processing") {
		t.Errorf("not processing: %v", err)
	}
	repo.set("UpdateStatusIfCurrent", true)
	if err := orch.HeartbeatTask(bg, "t", "s"); !errors.Is(err, errInjected) {
		t.Errorf("repo failure: %v", err)
	}

	raced := newOrchOver(t, &raceRepo{faultyRepo: newFaultyRepo()})
	if err := raced.HeartbeatTask(bg, "t", "s"); err == nil {
		t.Error("unknown task on raceRepo should fail")
	}
}

func TestHeartbeatTask_StatusChangedUnderneath(t *testing.T) {
	inner := newFaultyRepo()
	if err := inner.memRepo.Save(domain.Task{ID: "t", Status: domain.StatusProcessing, AISessionID: "s"}); err != nil {
		t.Fatal(err)
	}
	orch := newOrchOver(t, &raceRepo{faultyRepo: inner})
	if err := orch.HeartbeatTask(bg, "t", "s"); err == nil || !strings.Contains(err.Error(), "status changed") {
		t.Errorf("got %v", err)
	}
}
