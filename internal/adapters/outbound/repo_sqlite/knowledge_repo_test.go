package repo_sqlite_test

import (
	"context"
	"errors"
	"testing"
	"time"

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

	// SQLite triggers need to fire, wait briefly just in case, though they are synchronous.
	time.Sleep(10 * time.Millisecond)

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
