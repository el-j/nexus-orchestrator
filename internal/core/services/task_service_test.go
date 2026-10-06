package services_test

import (
	"errors"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

func TestTaskQueries_ScopeByProject(t *testing.T) {
	orch, repo, _ := newFaultyOrch(t)
	for _, tk := range []domain.Task{
		{ID: "a", ProjectPath: "/p1", Status: domain.StatusQueued},
		{ID: "b", ProjectPath: "/p1", Status: domain.StatusCompleted},
		{ID: "c", ProjectPath: "/p2", Status: domain.StatusProcessing},
	} {
		if err := repo.memRepo.Save(tk); err != nil {
			t.Fatal(err)
		}
	}

	q, err := orch.GetQueueForProject("/p1/")
	if err != nil || len(q) != 1 || q[0].ID != "a" {
		t.Errorf("GetQueueForProject (path is cleaned): %v %v", q, err)
	}
	all, err := orch.GetAllTasks()
	if err != nil || len(all) != 3 {
		t.Errorf("GetAllTasks: %v %v", all, err)
	}
	proj, err := orch.GetTasksForProject("/p1")
	if err != nil || len(proj) != 2 {
		t.Errorf("GetTasksForProject: %v %v", proj, err)
	}
}

func TestCancelTask_AllCancellableStatesAndErrors(t *testing.T) {
	for _, from := range []domain.TaskStatus{
		domain.StatusQueued, domain.StatusNoProvider, domain.StatusDraft, domain.StatusBacklog,
	} {
		orch, repo, _ := newFaultyOrch(t)
		seedTask(t, repo, "t", from)
		if err := orch.CancelTask("t"); err != nil {
			t.Fatalf("cancel from %s: %v", from, err)
		}
		if got, _ := repo.GetByID("t"); got.Status != domain.StatusCancelled {
			t.Errorf("from %s: status = %s", from, got.Status)
		}
	}

	orch, repo, _ := newFaultyOrch(t)
	seedTask(t, repo, "done", domain.StatusCompleted)
	if err := orch.CancelTask("done"); err == nil || !strings.Contains(err.Error(), "cannot cancel task with status COMPLETED") {
		t.Errorf("completed task: %v", err)
	}
	if err := orch.CancelTask("ghost"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("unknown task: want ErrNotFound, got %v", err)
	}

	// A repo failure at any of the four transition attempts is surfaced.
	repo.set("UpdateStatusIfCurrent", true)
	if err := orch.CancelTask("done"); !errors.Is(err, errInjected) {
		t.Errorf("first transition error not surfaced: %v", err)
	}
}

func TestCancelTask_FailureAtEachStageIsSurfaced(t *testing.T) {
	// failOnCall makes the Nth UpdateStatusIfCurrent call fail.
	for call := 1; call <= 4; call++ {
		repo := &countingFailRepo{faultyRepo: newFaultyRepo(), failAt: call}
		orch := newOrchOver(t, repo)
		seedTask(t, repo.faultyRepo, "x", domain.StatusCompleted)
		if err := orch.CancelTask("x"); !errors.Is(err, errInjected) {
			t.Errorf("call %d: error not surfaced: %v", call, err)
		}
	}
}

func TestCreateDraft_ValidationAndPersistenceErrors(t *testing.T) {
	orch, repo, _ := newFaultyOrch(t)
	if _, err := orch.CreateDraft(domain.Task{ProjectPath: "/p"}); err == nil || !strings.Contains(err.Error(), "instruction is required") {
		t.Errorf("missing instruction: %v", err)
	}
	if _, err := orch.CreateDraft(domain.Task{Instruction: "x"}); err == nil || !strings.Contains(err.Error(), "project path is required") {
		t.Errorf("missing project: %v", err)
	}
	repo.set("Save", true)
	if _, err := orch.CreateDraft(domain.Task{Instruction: "x", ProjectPath: "/p"}); !errors.Is(err, errInjected) {
		t.Errorf("save failure: %v", err)
	}
}

func TestGetBacklog_AllProjectsAndErrors(t *testing.T) {
	orch, repo, _ := newFaultyOrch(t)
	seedTask(t, repo, "d", domain.StatusDraft)
	seedTask(t, repo, "b", domain.StatusBacklog)
	seedTask(t, repo, "q", domain.StatusQueued)

	all, err := orch.GetBacklog("")
	if err != nil || len(all) != 2 {
		t.Fatalf("all projects: %v %v", all, err)
	}
	scoped, err := orch.GetBacklog("/proj")
	if err != nil || len(scoped) != 2 {
		t.Fatalf("scoped: %v %v", scoped, err)
	}

	repo.set("GetAll", true)
	if _, err := orch.GetBacklog(""); !errors.Is(err, errInjected) {
		t.Errorf("GetAll failure: %v", err)
	}
	repo.set("GetByProjectPathAndStatus", true)
	if _, err := orch.GetBacklog("/proj"); !errors.Is(err, errInjected) {
		t.Errorf("scoped failure: %v", err)
	}
}

func TestPromoteTask_ErrorsAndProviderWarning(t *testing.T) {
	orch, repo, _ := newFaultyOrch(t)

	if _, err := orch.PromoteTask("ghost"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("unknown: %v", err)
	}

	// No provider registered: promotion succeeds with a warning (also exercises
	// the dry-run for an unnamed and for a named provider).
	seedTask(t, repo, "d1", domain.StatusDraft)
	res, err := orch.PromoteTask("d1")
	if err != nil || !res.Promoted || !strings.Contains(res.Warning, "no active provider") {
		t.Fatalf("unnamed provider: %+v %v", res, err)
	}
	if err := repo.memRepo.Save(domain.Task{ID: "d2", ProjectPath: "/proj", Instruction: "i",
		Status: domain.StatusBacklog, ProviderName: "missing"}); err != nil {
		t.Fatal(err)
	}
	res, err = orch.PromoteTask("d2")
	if err != nil || !strings.Contains(res.Warning, `provider "missing" not found`) {
		t.Fatalf("named provider: %+v %v", res, err)
	}

	// Wrong status.
	seedTask(t, repo, "q", domain.StatusProcessing)
	if _, err := orch.PromoteTask("q"); err == nil || !strings.Contains(err.Error(), "cannot promote task with status") {
		t.Errorf("wrong status: %v", err)
	}

	// Repo failure on the conditional update, and a lost race.
	seedTask(t, repo, "d5", domain.StatusDraft)
	repo.set("UpdateStatusIfCurrent", true)
	if _, err := orch.PromoteTask("d5"); !errors.Is(err, errInjected) {
		t.Errorf("update failure: %v", err)
	}
}

func TestPromoteTask_NoWarningWhenProviderIsLive(t *testing.T) {
	live := services.NewDiscoveryService(&mockLLMClient{alive: true, name: "live"})
	orch, repo, _ := newFaultyOrchWith(t, live)
	seedTask(t, repo, "d3", domain.StatusDraft)
	if res, err := orch.PromoteTask("d3"); err != nil || !res.Promoted || res.Warning != "" {
		t.Errorf("unnamed: %+v %v", res, err)
	}
	if err := repo.memRepo.Save(domain.Task{ID: "d4", ProjectPath: "/proj", Instruction: "i",
		Status: domain.StatusDraft, ProviderName: "live"}); err != nil {
		t.Fatal(err)
	}
	if res, err := orch.PromoteTask("d4"); err != nil || res.Warning != "" {
		t.Errorf("named: %+v %v", res, err)
	}
}

func TestPromoteTask_LostRaceReportsStateChange(t *testing.T) {
	repo := &raceRepo{faultyRepo: newFaultyRepo()}
	orch := newOrchOver(t, repo)
	seedTask(t, repo.faultyRepo, "d", domain.StatusDraft)
	if _, err := orch.PromoteTask("d"); err == nil || !strings.Contains(err.Error(), "state changed during promotion") {
		t.Fatalf("got %v", err)
	}
}

func TestUpdateTask_FieldsStatusAndErrors(t *testing.T) {
	orch, repo, _ := newFaultyOrch(t)
	seedTask(t, repo, "t", domain.StatusQueued)

	got, err := orch.UpdateTask("t", domain.Task{
		Instruction: "new", TargetFile: "f.go", ProviderName: "p", ModelID: "m",
		ProviderHint: "h", Priority: 5, Tags: []string{"x"}, Status: domain.StatusBacklog,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Instruction != "new" || got.TargetFile != "f.go" || got.ProviderName != "p" || got.ModelID != "m" ||
		got.ProviderHint != "h" || got.Priority != 5 || len(got.Tags) != 1 || got.Status != domain.StatusBacklog {
		t.Errorf("merged task = %+v", got)
	}

	// Executing/terminal statuses are never set through UpdateTask.
	got, _ = orch.UpdateTask("t", domain.Task{Status: domain.StatusCompleted})
	if got.Status != domain.StatusBacklog {
		t.Errorf("status must not change to COMPLETED, got %s", got.Status)
	}

	if _, err := orch.UpdateTask("t", domain.Task{Status: domain.StatusQueued}); err == nil || !strings.Contains(err.Error(), "use promote task") {
		t.Errorf("queued transition: %v", err)
	}
	if _, err := orch.UpdateTask("ghost", domain.Task{}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("unknown task: %v", err)
	}
	repo.set("Update", true)
	if _, err := orch.UpdateTask("t", domain.Task{Instruction: "z"}); !errors.Is(err, errInjected) {
		t.Errorf("update failure: %v", err)
	}
}
