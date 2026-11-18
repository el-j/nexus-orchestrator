package repo_sqlite_test

import (
	"context"
	"math"
	"testing"

	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
)

func TestKnowledgeVector_EncodeDecodeRoundtrip(t *testing.T) {
	vec := []float32{0.1, -0.5, 0.999, 0.0, 123.456}
	r := newTestRepo(t)
	kRepo := repo_sqlite.NewKnowledgeRepo(r)
	ctx := context.Background()

	entry := domain.ProjectKnowledge{
		ID:             "vec-entry-1",
		ProjectPath:    "/proj/vec",
		Kind:           domain.KnowledgeArchitecture,
		Topic:          "VectorStore",
		Content:        "Vector store using cosine similarity",
		Embedding:      vec,
		RelevanceScore: 0.9,
	}
	if err := kRepo.SaveKnowledge(ctx, entry); err != nil {
		t.Fatalf("SaveKnowledge: %v", err)
	}

	got, err := kRepo.GetByID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(got.Embedding) != len(vec) {
		t.Fatalf("expected embedding length %d, got %d", len(vec), len(got.Embedding))
	}
	for i := range vec {
		if math.Abs(float64(got.Embedding[i]-vec[i])) > 1e-6 {
			t.Errorf("dim %d: expected %f, got %f", i, vec[i], got.Embedding[i])
		}
	}

	// Update entry with new embedding
	newVec := []float32{1.0, 2.0}
	got.Embedding = newVec
	if err := kRepo.UpdateKnowledge(ctx, got); err != nil {
		t.Fatalf("UpdateKnowledge: %v", err)
	}
	gotUpdated, err := kRepo.GetByID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("GetByID after update: %v", err)
	}
	if len(gotUpdated.Embedding) != 2 || gotUpdated.Embedding[0] != 1.0 || gotUpdated.Embedding[1] != 2.0 {
		t.Errorf("expected updated embedding, got %+v", gotUpdated.Embedding)
	}
}

func TestKnowledgeVector_SearchVector(t *testing.T) {
	r := newTestRepo(t)
	kRepo := repo_sqlite.NewKnowledgeRepo(r)
	ctx := context.Background()

	// Proj A entries
	itemA1 := domain.ProjectKnowledge{
		ProjectPath: "/proj/a",
		Kind:        domain.KnowledgeArchitecture,
		Topic:       "Auth",
		Content:     "Authentication using JWT",
		Embedding:   []float32{1.0, 0.0, 0.0},
	}
	itemA2 := domain.ProjectKnowledge{
		ProjectPath: "/proj/a",
		Kind:        domain.KnowledgeConvention,
		Topic:       "Style",
		Content:     "Go formatting rules",
		Embedding:   []float32{0.0, 1.0, 0.0},
	}
	// Proj B entry
	itemB1 := domain.ProjectKnowledge{
		ProjectPath: "/proj/b",
		Kind:        domain.KnowledgeArchitecture,
		Topic:       "OAuth",
		Content:     "OAuth login flows",
		Embedding:   []float32{0.9, 0.1, 0.0},
	}
	// Item without embedding
	itemNoEmb := domain.ProjectKnowledge{
		ProjectPath: "/proj/a",
		Kind:        domain.KnowledgeLearning,
		Topic:       "NoVector",
		Content:     "No vector here",
	}

	for _, item := range []domain.ProjectKnowledge{itemA1, itemA2, itemB1, itemNoEmb} {
		if err := kRepo.SaveKnowledge(ctx, item); err != nil {
			t.Fatalf("SaveKnowledge: %v", err)
		}
	}

	// 1. Empty target vector returns empty slice
	res, err := kRepo.SearchVector(ctx, "/proj/a", nil, 10)
	if err != nil {
		t.Fatalf("SearchVector nil: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("expected 0 results for nil target, got %d", len(res))
	}

	// 2. Single project search: target [1.0, 0.0, 0.0] in /proj/a
	res, err = kRepo.SearchVector(ctx, "/proj/a", []float32{1.0, 0.0, 0.0}, 10)
	if err != nil {
		t.Fatalf("SearchVector /proj/a: %v", err)
	}
	if len(res) != 1 || res[0].Topic != "Auth" {
		t.Errorf("expected 1 result (Auth), got %+v", res)
	}

	// 3. Cross-project search: target [1.0, 0.0, 0.0] with "*"
	// Should return itemA1 ([1.0, 0, 0]) first, then itemB1 ([0.9, 0.1, 0]) second
	res, err = kRepo.SearchVector(ctx, "*", []float32{1.0, 0.0, 0.0}, 10)
	if err != nil {
		t.Fatalf("SearchVector *: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 results, got %d", len(res))
	}
	if res[0].Topic != "Auth" || res[1].Topic != "OAuth" {
		t.Errorf("expected [Auth, OAuth], got [%s, %s]", res[0].Topic, res[1].Topic)
	}

	// 4. maxResults constraint
	resLimit, err := kRepo.SearchVector(ctx, "*", []float32{1.0, 0.0, 0.0}, 1)
	if err != nil {
		t.Fatalf("SearchVector limit 1: %v", err)
	}
	if len(resLimit) != 1 || resLimit[0].Topic != "Auth" {
		t.Errorf("expected 1 result (Auth), got %+v", resLimit)
	}

	// 5. Default maxResults when 0 or negative
	resDef, err := kRepo.SearchVector(ctx, "*", []float32{1.0, 0.0, 0.0}, 0)
	if err != nil {
		t.Fatalf("SearchVector default maxResults: %v", err)
	}
	if len(resDef) != 2 {
		t.Errorf("expected 2 results with default limit, got %d", len(resDef))
	}
}
