package services_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

func TestBrainService_InitProject_AutoDetectsDocsAndReadme(t *testing.T) {
	repo := &mockKnowledgeRepo{}
	b := services.NewBrainService(repo, nil)
	ctx := context.Background()

	tempDir := t.TempDir()

	// 1. Write README.md
	readmePath := filepath.Join(tempDir, "README.md")
	readmeContent := `# Project Title
This is the root readme preamble with enough characters to pass the min length check.

## Architecture & Storage
SQLite with WAL mode is used for embedded persistence and FTS5 indexing.
`
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		t.Fatalf("WriteFile README: %v", err)
	}

	// 2. Write docs/api.md
	docsDir := filepath.Join(tempDir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatalf("MkdirAll docs: %v", err)
	}
	apiDocPath := filepath.Join(docsDir, "api.md")
	apiDocContent := `# API Documentation
This is the documentation for all REST endpoints and JSON-RPC tools.

## REST Endpoints
All endpoints return JSON responses with standard error payloads.
`
	if err := os.WriteFile(apiDocPath, []byte(apiDocContent), 0644); err != nil {
		t.Fatalf("WriteFile api.md: %v", err)
	}

	// 3. Write docs/guides/setup.md (nested subdirectory to exercise scanDocDir recursion)
	guidesDir := filepath.Join(docsDir, "guides")
	if err := os.MkdirAll(guidesDir, 0755); err != nil {
		t.Fatalf("MkdirAll guides: %v", err)
	}
	guidePath := filepath.Join(guidesDir, "setup.md")
	guideContent := `# Setup Guide
Detailed instructions for setting up the developer environment properly.

## Prerequisites
Go 1.24 and Node 20 or later are required.
`
	if err := os.WriteFile(guidePath, []byte(guideContent), 0644); err != nil {
		t.Fatalf("WriteFile setup.md: %v", err)
	}

	// 4. Write docs/archive/old.md, docs/.hidden/secret.md, docs/testdata/dummy.md which must be skipped
	archiveDir := filepath.Join(docsDir, "archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		t.Fatalf("MkdirAll archive: %v", err)
	}
	if err := os.WriteFile(filepath.Join(archiveDir, "old.md"), []byte("## Old\nDeprecated notes."), 0644); err != nil {
		t.Fatalf("WriteFile old.md: %v", err)
	}
	hiddenDir := filepath.Join(docsDir, ".hidden")
	if err := os.MkdirAll(hiddenDir, 0755); err != nil {
		t.Fatalf("MkdirAll hidden: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "secret.md"), []byte("## Secret\nHidden notes."), 0644); err != nil {
		t.Fatalf("WriteFile secret.md: %v", err)
	}

	// InitProject without explicit file should detect README.md, docs/api.md, and docs/guides/setup.md
	if _, err := b.InitProject(ctx, tempDir, ""); err != nil {
		t.Fatalf("InitProject: %v", err)
	}

	var foundReadme, foundDoc, foundGuide, foundArchive, foundHidden bool
	for _, e := range repo.entries {
		if e.Source == "file:README.md" {
			foundReadme = true
		}
		if e.Source == "file:docs/api.md" {
			foundDoc = true
		}
		if e.Source == "file:docs/guides/setup.md" {
			foundGuide = true
		}
		if e.Source == "file:docs/archive/old.md" {
			foundArchive = true
		}
		if e.Source == "file:docs/.hidden/secret.md" {
			foundHidden = true
		}
	}

	if !foundReadme {
		t.Errorf("expected README.md to be ingested, entries: %v", repo.entries)
	}
	if !foundDoc {
		t.Errorf("expected docs/api.md to be ingested, entries: %v", repo.entries)
	}
	if !foundGuide {
		t.Errorf("expected docs/guides/setup.md to be ingested, entries: %v", repo.entries)
	}
	if foundArchive {
		t.Errorf("docs/archive/old.md must be skipped")
	}
	if foundHidden {
		t.Errorf("docs/.hidden/secret.md must be skipped")
	}
}

func TestBrainService_SearchKnowledge_CrossProject(t *testing.T) {
	repo := &mockKnowledgeRepo{
		entries: []domain.ProjectKnowledge{
			{ProjectPath: "/proj/a", Kind: domain.KnowledgeArchitecture, Topic: "Auth", Content: "JWT tokens", TokenCount: 10},
			{ProjectPath: "/proj/b", Kind: domain.KnowledgeConvention, Topic: "Style", Content: "Go style", TokenCount: 10},
		},
	}
	b := services.NewBrainService(repo, nil)
	ctx := context.Background()

	// Calling SearchKnowledge with "*" should pass "*" to repo.SearchFTS and populate ProjectPath
	sections, err := b.SearchKnowledge(ctx, "*", "auth", 10)
	if err != nil {
		t.Fatalf("SearchKnowledge: %v", err)
	}
	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}
	if sections[0].ProjectPath != "/proj/a" || sections[1].ProjectPath != "/proj/b" {
		t.Errorf("expected ProjectPath to be preserved on sections: %+v", sections)
	}
}

func TestBrainService_GetFocusedContext_CrossProject(t *testing.T) {
	repo := &mockKnowledgeRepo{
		entries: []domain.ProjectKnowledge{
			{ProjectPath: "/proj/a", Kind: domain.KnowledgeLearning, Topic: "Quota", Content: "Cooldowns", TokenCount: 10},
		},
	}
	b := services.NewBrainService(repo, nil)
	ctx := context.Background()

	q := domain.ContextQuery{
		ProjectPath: "*",
		Question:    "how to handle quota",
		MaxTokens:   100,
	}
	resp, err := b.GetFocusedContext(ctx, q)
	if err != nil {
		t.Fatalf("GetFocusedContext: %v", err)
	}
	if resp.ProjectPath != "*" {
		t.Errorf("expected resp.ProjectPath = '*', got %q", resp.ProjectPath)
	}
	if len(resp.Sections) != 1 || resp.Sections[0].ProjectPath != "/proj/a" {
		t.Errorf("expected section with ProjectPath '/proj/a', got %+v", resp.Sections)
	}
}

type mockEmbeddingProvider struct {
	vector []float32
	err    error
}

func (m *mockEmbeddingProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.vector, nil
}

func TestBrainService_HybridSearch_WithEmbedder(t *testing.T) {
	repo := &mockKnowledgeRepo{
		entries: []domain.ProjectKnowledge{
			{ID: "k1", ProjectPath: "/proj/a", Kind: domain.KnowledgeArchitecture, Topic: "JWT", Content: "JWT auth", TokenCount: 10, Embedding: []float32{1.0, 0.0}},
			{ID: "k2", ProjectPath: "/proj/a", Kind: domain.KnowledgeConvention, Topic: "OAuth", Content: "OAuth auth", TokenCount: 10, Embedding: []float32{0.9, 0.1}},
		},
	}
	b := services.NewBrainService(repo, nil)
	embedder := &mockEmbeddingProvider{vector: []float32{1.0, 0.0}}
	b.SetEmbeddingProvider(embedder)
	ctx := context.Background()

	// 1. SearchKnowledge uses hybrid fusion
	sections, err := b.SearchKnowledge(ctx, "/proj/a", "auth", 100)
	if err != nil {
		t.Fatalf("SearchKnowledge: %v", err)
	}
	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}

	// 2. GetFocusedContext uses hybrid fusion
	resp, err := b.GetFocusedContext(ctx, domain.ContextQuery{
		ProjectPath: "/proj/a",
		Question:    "auth",
		MaxTokens:   100,
	})
	if err != nil {
		t.Fatalf("GetFocusedContext: %v", err)
	}
	if len(resp.Sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(resp.Sections))
	}

	// 3. Fallback when embedder errors
	embedder.err = context.Canceled
	sectionsFallback, err := b.SearchKnowledge(ctx, "/proj/a", "auth", 100)
	if err != nil {
		t.Fatalf("SearchKnowledge fallback: %v", err)
	}
	if len(sectionsFallback) != 2 {
		t.Fatalf("expected fallback to BM25 results, got %d", len(sectionsFallback))
	}

	respFallback, err := b.GetFocusedContext(ctx, domain.ContextQuery{
		ProjectPath: "/proj/a",
		Question:    "auth",
		MaxTokens:   100,
	})
	if err != nil {
		t.Fatalf("GetFocusedContext fallback: %v", err)
	}
	if len(respFallback.Sections) != 2 {
		t.Fatalf("expected fallback to BM25 focused context, got %d", len(respFallback.Sections))
	}
}

func TestBrainService_IngestKnowledge_WithEmbedder(t *testing.T) {
	repo := &mockKnowledgeRepo{}
	b := services.NewBrainService(repo, nil)
	embedder := &mockEmbeddingProvider{vector: []float32{0.5, 0.5}}
	b.SetEmbeddingProvider(embedder)
	ctx := context.Background()

	// 1. Ingest with successful embedder generates vector
	k1, err := b.IngestKnowledge(ctx, domain.ProjectKnowledge{
		ProjectPath: "/proj/a",
		Kind:        domain.KnowledgeArchitecture,
		Topic:       "DB",
		Content:     "PostgreSQL and SQLite",
	})
	if err != nil {
		t.Fatalf("IngestKnowledge: %v", err)
	}
	if len(k1.Embedding) != 2 || k1.Embedding[0] != 0.5 {
		t.Errorf("expected embedding to be generated, got %+v", k1.Embedding)
	}

	// 2. Ingest when embedder fails gracefully degrades without error
	embedder.err = context.DeadlineExceeded
	k2, err := b.IngestKnowledge(ctx, domain.ProjectKnowledge{
		ProjectPath: "/proj/a",
		Kind:        domain.KnowledgeConvention,
		Topic:       "CodeStyle",
		Content:     "Go formatting rules",
	})
	if err != nil {
		t.Fatalf("IngestKnowledge when embedder fails: %v", err)
	}
	if len(k2.Embedding) != 0 {
		t.Errorf("expected nil embedding when embedder fails, got %+v", k2.Embedding)
	}
}
