package services_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

// One repository must be one project, however deep in it the agent's shell happens to be, and
// a throwaway command in a temp directory is not a project at all.
func TestActivityService_ActivitiesBelongToTheRepositoryNotTheFolder(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := filepath.Join(home, "code", "app")
	for _, d := range []string{filepath.Join(repo, ".git"), filepath.Join(repo, "web", "src")} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	act := func(id, project string) domain.AIActivity {
		return domain.AIActivity{ID: id, AgentName: "claude", ProjectPath: project, Summary: "x", Timestamp: time.Now()}
	}
	repoStore, sessions := &actMemActivityRepo{}, newMemAISessionRepo()
	reader := &mockReader{name: "r", activities: []domain.AIActivity{
		act("a", repo), act("b", filepath.Join(repo, "web")), act("c", filepath.Join(repo, "web", "src")),
		act("d", "/private/tmp/scratch"), act("e", "/private/var/folders/x/T/y"),
	}}
	svc := services.NewActivityService(repoStore, sessions, reader)
	svc.PollNow()

	got, _ := svc.GetRecentActivities(context.Background(), domain.ActivityFilter{})
	if len(got) != 3 {
		t.Fatalf("saved %d activities, want the 3 inside the repository (scratch directories are skipped)", len(got))
	}
	for _, a := range got {
		if a.ProjectPath != repo {
			t.Errorf("activity %s has project %q, want the repository %q", a.ID, a.ProjectPath, repo)
		}
	}
	list, _ := sessions.ListAISessions(context.Background())
	if len(list) != 1 || list[0].ProjectPath != repo {
		t.Errorf("sessions = %+v, want one session on the repository", list)
	}
}

// Sessions recorded before this, at folders inside a repository or in a temp directory, are
// removed; the next activity creates the session again on the repository.
func TestActivityService_TidiesOldDiscoveredSessionsAtSubfoldersAndScratch(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := filepath.Join(home, "app")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "sub"), 0o750); err != nil {
		t.Fatal(err)
	}
	sessions := newMemAISessionRepo()
	ctx := context.Background()
	save := func(id, source, path string) {
		_ = sessions.SaveAISession(ctx, domain.AISession{ID: id, Source: domain.AISessionSource(source), AgentName: "claude", ProjectPath: path, Status: domain.SessionStatusIdle, LastActivity: time.Now()})
	}
	save("root", "discovered", repo)
	save("sub", "discovered", filepath.Join(repo, "sub"))
	save("tmp", "discovered", "/private/var/folders/x/T")
	save("mine", "vscode", filepath.Join(repo, "sub")) // registered by an editor: not ours to remove
	services.NewActivityService(&actMemActivityRepo{}, sessions, &mockReader{name: "r"}).PollNow()

	left, _ := sessions.ListAISessions(ctx)
	ids := map[string]bool{}
	for _, s := range left {
		ids[s.ID] = true
	}
	if !ids["root"] || !ids["mine"] || ids["sub"] || ids["tmp"] {
		t.Errorf("sessions left = %v, want root and mine only", ids)
	}
}
