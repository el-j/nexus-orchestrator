package repo_sqlite_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	repo_sqlite "nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
)

// TestAllRepos_ReturnErrorsWhenDatabaseIsClosed drives every public method of
// every repository against a closed database and requires each to surface an
// error instead of panicking or silently succeeding.
func TestAllRepos_ReturnErrorsWhenDatabaseIsClosed(t *testing.T) {
	r := closedRepo(t)
	ctx := context.Background()
	task := domain.Task{ID: "t", ProjectPath: "/p", Status: domain.StatusQueued}

	sessions := repo_sqlite.NewSessionRepo(r)
	ai := repo_sqlite.NewAISessionRepo(r)
	know := repo_sqlite.NewKnowledgeRepo(r)
	agents := repo_sqlite.NewDiscoveredAgentRepo(r)
	providers := repo_sqlite.NewProviderConfigRepo(r)

	calls := map[string]func() error{
		"Repository.Save":         func() error { return r.Save(task) },
		"Repository.GetByID":      func() error { _, err := r.GetByID("t"); return err },
		"Repository.GetPending":   func() error { _, err := r.GetPending(); return err },
		"Repository.ClaimNext":    func() error { _, err := r.ClaimNextQueued(); return err },
		"Repository.UpdateStatus": func() error { return r.UpdateStatus("t", domain.StatusFailed) },
		"Repository.UpdateIfCur": func() error {
			_, err := r.UpdateStatusIfCurrent("t", domain.StatusQueued, domain.StatusFailed)
			return err
		},
		"Repository.UpdateLogs":   func() error { return r.UpdateLogs("t", "x") },
		"Repository.GetAll":       func() error { _, err := r.GetAll(); return err },
		"Repository.ByProject":    func() error { _, err := r.GetByProjectPath("/p"); return err },
		"Repository.ByProjStatus": func() error { _, err := r.GetByProjectPathAndStatus("/p", domain.StatusQueued); return err },
		"Repository.Update":       func() error { return r.Update(task) },
		"Repository.BySessionID":  func() error { _, err := r.GetTasksBySessionID("s"); return err },
		"Repository.StaleProc":    func() error { _, err := r.GetStaleProcessing(ctx, time.Hour); return err },
		"SessionRepo.Save":        func() error { return sessions.Save(domain.Session{ProjectPath: "/p"}) },
		"SessionRepo.Get":         func() error { _, err := sessions.GetByProjectPath("/p"); return err },
		"SessionRepo.Append":      func() error { return sessions.AppendMessage("/p", domain.Message{Role: domain.RoleUser}) },
		"AISession.Save":          func() error { return ai.SaveAISession(ctx, domain.AISession{ID: "a"}) },
		"AISession.GetByID":       func() error { _, err := ai.GetAISessionByID(ctx, "a"); return err },
		"AISession.GetByExt":      func() error { _, err := ai.GetAISessionByExternalID(ctx, "e"); return err },
		"AISession.List":          func() error { _, err := ai.ListAISessions(ctx); return err },
		"AISession.UpdateStatus":  func() error { return ai.UpdateAISessionStatus(ctx, "a", domain.AISessionStatus("x"), time.Now()) },
		"AISession.Delete":        func() error { return ai.DeleteAISession(ctx, "a") },
		"AISession.AppendRouted":  func() error { return ai.AppendRoutedTaskID(ctx, "a", "t") },
		"AISession.PurgeDisc":     func() error { _, err := ai.PurgeDisconnected(ctx, time.Hour); return err },
		"Knowledge.Save":          func() error { return know.SaveKnowledge(ctx, domain.ProjectKnowledge{ID: "k"}) },
		"Knowledge.GetByID":       func() error { _, err := know.GetByID(ctx, "k"); return err },
		"Knowledge.Update":        func() error { return know.UpdateKnowledge(ctx, domain.ProjectKnowledge{ID: "k"}) },
		"Knowledge.Delete":        func() error { return know.DeleteKnowledge(ctx, "k") },
		"Knowledge.GetByProject":  func() error { _, err := know.GetByProject(ctx, "/p"); return err },
		"Knowledge.GetByKind":     func() error { _, err := know.GetByProjectAndKind(ctx, "/p", domain.KnowledgeLearning); return err },
		"Knowledge.SearchFTS":     func() error { _, err := know.SearchFTS(ctx, "/p", "q", 5); return err },
		"Knowledge.GetStatus":     func() error { _, err := know.GetStatus(ctx, "/p"); return err },
		"Knowledge.DeleteProject": func() error { return know.DeleteByProject(ctx, "/p") },
		"Agents.Upsert":           func() error { return agents.UpsertDiscoveredAgent(ctx, domain.DiscoveredAgent{ID: "d"}) },
		"Agents.List":             func() error { _, err := agents.ListDiscoveredAgents(ctx); return err },
		"Providers.Save":          func() error { return providers.SaveProviderConfig(ctx, domain.ProviderConfig{ID: "p"}) },
		"Providers.List":          func() error { _, err := providers.ListProviderConfigs(ctx); return err },
		"Providers.Get":           func() error { _, err := providers.GetProviderConfig(ctx, "p"); return err },
		"Providers.Delete":        func() error { return providers.DeleteProviderConfig(ctx, "p") },
	}
	for name, call := range calls {
		if err := call(); err == nil {
			t.Errorf("%s: expected an error from a closed database, got nil", name)
		}
	}
}

func TestNew_FailsWhenDatabaseCannotBeOpened(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no", "such", "dir", "x.db")
	if _, err := repo_sqlite.New(missing); err == nil {
		t.Fatal("expected error for an unopenable path")
	}
}

func TestNew_FailsWhenMigrationConflicts(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "conflict.db")
	raw, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	// A view named "tasks" makes the migration's index creation fail.
	if _, err := raw.Exec(`CREATE VIEW tasks AS SELECT 1 AS id`); err != nil {
		t.Fatal(err)
	}
	raw.Close()

	if _, err := repo_sqlite.New(dbPath); err == nil {
		t.Fatal("expected migration error")
	}
}

func TestNew_FailsOnNonDatabaseFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "garbage.db")
	if err := os.WriteFile(p, []byte("this is not a sqlite database, just text padding padding padding"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := repo_sqlite.New(p); err == nil {
		t.Fatal("expected error for a non-database file")
	}
}
