package repo_sqlite

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"path/filepath"
	"sort"

	"nexus-orchestrator/internal/core/domain"
)

// encodeEmbedding serialises a float32 slice into a little-endian byte slice.
func encodeEmbedding(v []float32) []byte {
	if len(v) == 0 {
		return nil
	}
	buf := make([]byte, len(v)*4)
	for i, f := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

// decodeEmbedding deserialises a little-endian byte slice into a float32 slice.
func decodeEmbedding(b []byte) []float32 {
	if len(b) < 4 {
		return nil
	}
	n := len(b) / 4
	out := make([]float32, n)
	for i := 0; i < n; i++ {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out
}

// cosineSimilarity computes the cosine similarity between two float vectors.
// Returns 0 if either vector is empty, lengths differ, or magnitude is zero.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float32
	for i := 0; i < len(a); i++ {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

type scoredKnowledge struct {
	entry domain.ProjectKnowledge
	score float32
}

// SearchVector finds entries with embeddings closest to target by cosine similarity.
// If projectPath is empty or "*", search runs across all projects.
func (r *KnowledgeRepo) SearchVector(ctx context.Context, projectPath string, target []float32, maxResults int) ([]domain.ProjectKnowledge, error) {
	if len(target) == 0 {
		return []domain.ProjectKnowledge{}, nil
	}
	if maxResults <= 0 {
		maxResults = 20
	}

	var rows *sql.Rows
	var err error

	isCrossProject := projectPath == "" || projectPath == "*" || projectPath == "all"
	if isCrossProject {
		rows, err = r.db.QueryContext(ctx, `
			SELECT id, project_path, kind, topic, content, source,
			       token_count, relevance_score, embedding, created_at, updated_at
			FROM project_knowledge
			WHERE embedding IS NOT NULL`,
		)
	} else {
		rows, err = r.db.QueryContext(ctx, `
			SELECT id, project_path, kind, topic, content, source,
			       token_count, relevance_score, embedding, created_at, updated_at
			FROM project_knowledge
			WHERE embedding IS NOT NULL AND project_path = ?`,
			filepath.Clean(projectPath),
		)
	}
	if err != nil {
		return nil, fmt.Errorf("knowledge_repo: search vector: %w", err)
	}
	defer rows.Close()
	entries, err := collectKnowledge(rows)
	if err != nil {
		return nil, err
	}

	var scored []scoredKnowledge
	for _, k := range entries {
		sim := cosineSimilarity(target, k.Embedding)
		if sim > 0 {
			scored = append(scored, scoredKnowledge{entry: k, score: sim})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	if len(scored) > maxResults {
		scored = scored[:maxResults]
	}

	out := make([]domain.ProjectKnowledge, len(scored))
	for i, s := range scored {
		out[i] = s.entry
	}
	return out, nil
}
