package repo_sqlite_test

import (
	"context"
	"errors"
	"testing"

	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
)

func TestKnowledgeRepo_SaveAndGetByID(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()

	kr := repo_sqlite.NewKnowledgeRepo(r)
	ctx := context.Background()

	k := domain.ProjectKnowledge{
		ID:             "kn-1",
		ProjectPath:    "/proj/brain",
		Kind:           domain.KnowledgeArchitecture,
		Topic:          "Hexagonal",
		Content:        "We use hexagonal architecture.",
		Source:         "file:CLAUDE.md",
		TokenCount:     10,
		RelevanceScore: 0.8,
	}

	if err := kr.SaveKnowledge(ctx, k); err != nil {
		t.Fatalf("SaveKnowledge: %v", err)
	}

	got, err := kr.GetByID(ctx, "kn-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.ID != k.ID {
		t.Errorf("ID: got %q, want %q", got.ID, k.ID)
	}
	if got.Topic != k.Topic {
		t.Errorf("Topic: got %q, want %q", got.Topic, k.Topic)
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Errorf("Timestamps not set: created=%v, updated=%v", got.CreatedAt, got.UpdatedAt)
	}
}

func TestKnowledgeRepo_DeleteKnowledge(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()

	kr := repo_sqlite.NewKnowledgeRepo(r)
	ctx := context.Background()

	k := domain.ProjectKnowledge{
		ID:          "kn-delete",
		ProjectPath: "/proj/brain",
		Kind:        domain.KnowledgeConvention,
		Topic:       "Delete Me",
		Content:     "To be deleted",
	}

	_ = kr.SaveKnowledge(ctx, k)

	if err := kr.DeleteKnowledge(ctx, "kn-delete"); err != nil {
		t.Fatalf("DeleteKnowledge: %v", err)
	}

	_, err := kr.GetByID(ctx, "kn-delete")
	if err == nil {
		t.Fatalf("Expected error getting deleted knowledge, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestKnowledgeRepo_GetStatus(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()

	kr := repo_sqlite.NewKnowledgeRepo(r)
	ctx := context.Background()

	entries := []domain.ProjectKnowledge{
		{ID: "s1", ProjectPath: "/proj", Kind: domain.KnowledgeArchitecture, Topic: "A", Content: "A", TokenCount: 10},
		{ID: "s2", ProjectPath: "/proj", Kind: domain.KnowledgeArchitecture, Topic: "B", Content: "B", TokenCount: 20},
		{ID: "s3", ProjectPath: "/proj", Kind: domain.KnowledgeConvention, Topic: "C", Content: "C", TokenCount: 5},
	}
	for _, e := range entries {
		_ = kr.SaveKnowledge(ctx, e)
	}

	status, err := kr.GetStatus(ctx, "/proj")
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	if !status.Initialized {
		t.Errorf("Expected Initialized = true")
	}
	if status.EntryCount != 3 {
		t.Errorf("Expected EntryCount = 3, got %d", status.EntryCount)
	}
	if status.TotalTokens != 35 {
		t.Errorf("Expected TotalTokens = 35, got %d", status.TotalTokens)
	}
	if status.KindCounts[string(domain.KnowledgeArchitecture)] != 2 {
		t.Errorf("Expected 2 architecture entries")
	}
	if status.KindCounts[string(domain.KnowledgeConvention)] != 1 {
		t.Errorf("Expected 1 convention entry")
	}
}

func TestKnowledgeRepo_SearchFTS(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()

	kr := repo_sqlite.NewKnowledgeRepo(r)
	ctx := context.Background()

	entries := []domain.ProjectKnowledge{
		{ID: "fts1", ProjectPath: "/proj", Kind: domain.KnowledgeLearning, Topic: "Go Interface", Content: "Use smaller interfaces."},
		{ID: "fts2", ProjectPath: "/proj", Kind: domain.KnowledgeLearning, Topic: "Database", Content: "SQLite is fast."},
		{ID: "fts3", ProjectPath: "/proj", Kind: domain.KnowledgeLearning, Topic: "Go concurrency", Content: "Use goroutines and channels."},
	}
	for _, e := range entries {
		_ = kr.SaveKnowledge(ctx, e)
	}

	results, err := kr.SearchFTS(ctx, "/proj", "interface", 10)
	if err != nil {
		t.Fatalf("SearchFTS: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result for 'interface', got %d", len(results))
	}
	if results[0].ID != "fts1" {
		t.Errorf("Expected fts1, got %s", results[0].ID)
	}
}

func TestKnowledgeRepo_UpsertDedup(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()

	kr := repo_sqlite.NewKnowledgeRepo(r)
	ctx := context.Background()

	first := domain.ProjectKnowledge{
		ID:          "ud-1",
		ProjectPath: "/proj/dedup",
		Kind:        domain.KnowledgeConvention,
		Topic:       "ErrorHandling",
		Content:     "Always wrap errors with fmt.Errorf.",
		TokenCount:  10,
	}
	if err := kr.SaveKnowledge(ctx, first); err != nil {
		t.Fatalf("first SaveKnowledge: %v", err)
	}

	// Same project+kind+topic, different ID and content — must upsert, not duplicate.
	second := domain.ProjectKnowledge{
		ID:          "ud-2",
		ProjectPath: "/proj/dedup",
		Kind:        domain.KnowledgeConvention,
		Topic:       "ErrorHandling",
		Content:     "Always wrap errors with %w and include context.",
		TokenCount:  15,
	}
	if err := kr.SaveKnowledge(ctx, second); err != nil {
		t.Fatalf("second SaveKnowledge: %v", err)
	}

	entries, err := kr.GetByProject(ctx, "/proj/dedup")
	if err != nil {
		t.Fatalf("GetByProject: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 entry after upsert, got %d", len(entries))
	}
	if entries[0].Content != second.Content {
		t.Errorf("content not updated: got %q, want %q", entries[0].Content, second.Content)
	}
}

func TestKnowledgeRepo_SearchFTS_Ranking(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()

	kr := repo_sqlite.NewKnowledgeRepo(r)
	ctx := context.Background()

	entries := []domain.ProjectKnowledge{
		{
			ID:          "rank-arch",
			ProjectPath: "/proj/rank",
			Kind:        domain.KnowledgeArchitecture,
			Topic:       "architecture hexagonal port adapter",
			Content:     "We use hexagonal architecture with ports and adapters.",
		},
		{
			ID:          "rank-food",
			ProjectPath: "/proj/rank",
			Kind:        domain.KnowledgeLearning,
			Topic:       "cooking recipes pasta",
			Content:     "Recipes for pasta and other cooking ideas.",
		},
	}
	for _, e := range entries {
		if err := kr.SaveKnowledge(ctx, e); err != nil {
			t.Fatalf("SaveKnowledge %q: %v", e.ID, err)
		}
	}

	results, err := kr.SearchFTS(ctx, "/proj/rank", "architecture", 5)
	if err != nil {
		t.Fatalf("SearchFTS: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 result, got 0")
	}
	if results[0].ID != "rank-arch" {
		t.Errorf("expected architecture entry first, got %q", results[0].ID)
	}
}

func TestKnowledgeRepo_SearchFTS_ProjectIsolation(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()

	kr := repo_sqlite.NewKnowledgeRepo(r)
	ctx := context.Background()

	entries := []domain.ProjectKnowledge{
		{
			ID:          "iso-a",
			ProjectPath: "/proj/iso-A",
			Kind:        domain.KnowledgeArchitecture,
			Topic:       "hexagonal design",
			Content:     "Hexagonal architecture keeps domain pure.",
		},
		{
			ID:          "iso-b",
			ProjectPath: "/proj/iso-B",
			Kind:        domain.KnowledgeArchitecture,
			Topic:       "hexagonal design",
			Content:     "Hexagonal boundaries separate concerns.",
		},
	}
	for _, e := range entries {
		if err := kr.SaveKnowledge(ctx, e); err != nil {
			t.Fatalf("SaveKnowledge %q: %v", e.ID, err)
		}
	}

	results, err := kr.SearchFTS(ctx, "/proj/iso-A", "hexagonal", 5)
	if err != nil {
		t.Fatalf("SearchFTS: %v", err)
	}
	for _, res := range results {
		if res.ProjectPath != "/proj/iso-A" {
			t.Errorf("result from wrong project: got %q, want /proj/iso-A", res.ProjectPath)
		}
		if res.ID == "iso-b" {
			t.Errorf("proj-B entry must not appear in proj-A search results")
		}
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 result for proj-A, got 0")
	}
}

func TestKnowledgeRepo_DeleteByProject(t *testing.T) {
	r := newTestRepo(t)
	defer r.Close()

	kr := repo_sqlite.NewKnowledgeRepo(r)
	ctx := context.Background()

	entries := []domain.ProjectKnowledge{
		{ID: "dbp-a1", ProjectPath: "/proj/del-A", Kind: domain.KnowledgeArchitecture, Topic: "One", Content: "First entry for A."},
		{ID: "dbp-a2", ProjectPath: "/proj/del-A", Kind: domain.KnowledgeConvention, Topic: "Two", Content: "Second entry for A."},
		{ID: "dbp-b1", ProjectPath: "/proj/del-B", Kind: domain.KnowledgeLearning, Topic: "Three", Content: "Entry for B."},
	}
	for _, e := range entries {
		if err := kr.SaveKnowledge(ctx, e); err != nil {
			t.Fatalf("SaveKnowledge %q: %v", e.ID, err)
		}
	}

	if err := kr.DeleteByProject(ctx, "/proj/del-A"); err != nil {
		t.Fatalf("DeleteByProject: %v", err)
	}

	aEntries, err := kr.GetByProject(ctx, "/proj/del-A")
	if err != nil {
		t.Fatalf("GetByProject proj-A: %v", err)
	}
	if len(aEntries) != 0 {
		t.Errorf("expected 0 entries for proj-A after deletion, got %d", len(aEntries))
	}

	bEntries, err := kr.GetByProject(ctx, "/proj/del-B")
	if err != nil {
		t.Fatalf("GetByProject proj-B: %v", err)
	}
	if len(bEntries) != 1 {
		t.Fatalf("expected 1 entry for proj-B, got %d", len(bEntries))
	}
	if bEntries[0].ID != "dbp-b1" {
		t.Errorf("expected dbp-b1 for proj-B, got %q", bEntries[0].ID)
	}
}
