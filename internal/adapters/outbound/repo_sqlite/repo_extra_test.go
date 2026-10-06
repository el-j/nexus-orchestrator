package repo_sqlite_test

import (
	"context"
	"errors"
	"testing"
	"time"

	repo_sqlite "nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
)

func TestRepository_GetAll_NewestFirst(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	now := time.Now()
	for i, id := range []string{"old", "mid", "new"} {
		if err := r.Save(newTask(id, domain.StatusQueued, now.Add(time.Duration(i)*time.Second))); err != nil {
			t.Fatal(err)
		}
	}
	all, err := r.GetAll()
	if err != nil || len(all) != 3 || all[0].ID != "new" || all[2].ID != "old" {
		t.Fatalf("GetAll: %v %v", all, err)
	}
}

func TestRepository_ClaimNextQueued_OldestFirstThenEmpty(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	now := time.Now()
	_ = r.Save(newTask("second", domain.StatusQueued, now.Add(time.Second)))
	_ = r.Save(newTask("first", domain.StatusQueued, now))

	got, err := r.ClaimNextQueued()
	if err != nil || got.ID != "first" || got.Status != domain.StatusProcessing {
		t.Fatalf("claim 1: %+v %v", got, err)
	}
	got, err = r.ClaimNextQueued()
	if err != nil || got.ID != "second" {
		t.Fatalf("claim 2: %+v %v", got, err)
	}
	if _, err = r.ClaimNextQueued(); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("claim on empty queue: want ErrNotFound, got %v", err)
	}
}

func TestRepository_UpdateStatusIfCurrent_Preconditions(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	_ = r.Save(newTask("t", domain.StatusQueued, time.Now()))

	ok, err := r.UpdateStatusIfCurrent("t", domain.StatusProcessing, domain.StatusFailed)
	if err != nil || ok {
		t.Fatalf("mismatched precondition must not update: ok=%v err=%v", ok, err)
	}
	ok, err = r.UpdateStatusIfCurrent("t", domain.StatusQueued, domain.StatusProcessing)
	if err != nil || !ok {
		t.Fatalf("matching precondition must update: ok=%v err=%v", ok, err)
	}
}

func TestRepository_GetStaleProcessingAndSessionLookup(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	ctx := context.Background()

	stale := newTask("stale", domain.StatusProcessing, time.Now().Add(-2*time.Hour))
	stale.AISessionID = "sess-1"
	fresh := newTask("fresh", domain.StatusProcessing, time.Now())
	for _, tk := range []domain.Task{stale, fresh} {
		if err := r.Save(tk); err != nil {
			t.Fatal(err)
		}
	}
	got, err := r.GetStaleProcessing(ctx, time.Hour)
	if err != nil || len(got) != 1 || got[0].ID != "stale" {
		t.Fatalf("stale: %v %v", got, err)
	}
	bySess, err := r.GetTasksBySessionID("sess-1")
	if err != nil || len(bySess) != 1 || bySess[0].ID != "stale" {
		t.Fatalf("by session: %v %v", bySess, err)
	}
}

func TestRepository_CorruptRowsSurfaceErrorsOrAreTolerated(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	_ = r.Save(newTask("bad", domain.StatusQueued, time.Now()))

	// Corrupt tags are tolerated (logged, not fatal).
	if _, err := r.DB().Exec(`UPDATE tasks SET tags = '{not json'`); err != nil {
		t.Fatal(err)
	}
	if got, err := r.GetByID("bad"); err != nil || got.Tags != nil {
		t.Fatalf("corrupt tags should be ignored: %+v %v", got, err)
	}

	// Corrupt context_files is an error on every read path.
	if _, err := r.DB().Exec(`UPDATE tasks SET context_files = '{not json'`); err != nil {
		t.Fatal(err)
	}
	reads := map[string]func() error{
		"GetByID":      func() error { _, err := r.GetByID("bad"); return err },
		"GetAll":       func() error { _, err := r.GetAll(); return err },
		"GetPending":   func() error { _, err := r.GetPending(); return err },
		"ByProject":    func() error { _, err := r.GetByProjectPath("/projects/foo"); return err },
		"ByProjStatus": func() error { _, err := r.GetByProjectPathAndStatus("/projects/foo", domain.StatusQueued); return err },
		"ClaimNext":    func() error { _, err := r.ClaimNextQueued(); return err },
	}
	for name, read := range reads {
		if err := read(); err == nil {
			t.Errorf("%s: expected error for corrupt context_files", name)
		}
	}
}

func TestRepository_GetByProjectPathAndStatus_NoStatusesIsEmpty(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	got, err := r.GetByProjectPathAndStatus("/projects/foo")
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v %v", got, err)
	}
}

func TestKnowledgeRepo_UpdateDeleteKindAndProjectLifecycle(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()
	repo := repo_sqlite.NewKnowledgeRepo(r)
	ctx := context.Background()

	entries := []domain.ProjectKnowledge{
		{ProjectPath: "/p", Kind: domain.KnowledgeArchitecture, Topic: "layers", Content: "hexagonal architecture with ports", RelevanceScore: 0.9},
		{ProjectPath: "/p", Kind: domain.KnowledgeLearning, Topic: "tests", Content: "table driven tests are preferred", RelevanceScore: 0.4},
	}
	for _, e := range entries {
		if err := repo.SaveKnowledge(ctx, e); err != nil {
			t.Fatal(err)
		}
	}

	arch, err := repo.GetByProjectAndKind(ctx, "/p", domain.KnowledgeArchitecture)
	if err != nil || len(arch) != 1 || arch[0].Topic != "layers" {
		t.Fatalf("by kind: %v %v", arch, err)
	}

	arch[0].Content = "updated content about layers"
	if err := repo.UpdateKnowledge(ctx, arch[0]); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByID(ctx, arch[0].ID)
	if err != nil || got.Content != "updated content about layers" {
		t.Fatalf("after update: %+v %v", got, err)
	}
	if want := len("updated content about layers") / 4; got.TokenCount != want {
		t.Errorf("update should re-derive TokenCount: got %d want %d", got.TokenCount, want)
	}

	hits, err := repo.SearchFTS(ctx, "/p", "tests", 5)
	if err != nil || len(hits) != 1 {
		t.Fatalf("fts: %v %v", hits, err)
	}

	status, err := repo.GetStatus(ctx, "/p")
	if err != nil || !status.Initialized || status.EntryCount != 2 {
		t.Fatalf("status: %+v %v", status, err)
	}

	if err := repo.DeleteKnowledge(ctx, arch[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteKnowledge(ctx, arch[0].ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("second delete: want ErrNotFound, got %v", err)
	}
	if err := repo.DeleteByProject(ctx, "/p"); err != nil {
		t.Fatal(err)
	}
	if left, _ := repo.GetByProject(ctx, "/p"); len(left) != 0 {
		t.Errorf("project not emptied: %v", left)
	}
}
