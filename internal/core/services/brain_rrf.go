package services

import (
	"sort"

	"nexus-orchestrator/internal/core/domain"
)

// rrfConstant is the smoothing constant k in the standard Reciprocal Rank Fusion formula:
// RRF(d) = sum(1 / (k + rank)). Standard value in literature is 60.
const rrfConstant = 60.0

type rrfCandidate struct {
	entry domain.ProjectKnowledge
	score float64
}

// FuseRRF merges lexical (BM25) and vector search results using Reciprocal Rank Fusion.
// It returns unique entries sorted by their fused score up to maxResults.
func FuseRRF(lexical, vector []domain.ProjectKnowledge, maxResults int) []domain.ProjectKnowledge {
	if len(lexical) == 0 && len(vector) == 0 {
		return []domain.ProjectKnowledge{}
	}
	if len(vector) == 0 {
		if maxResults > 0 && len(lexical) > maxResults {
			return lexical[:maxResults]
		}
		return lexical
	}
	if len(lexical) == 0 {
		if maxResults > 0 && len(vector) > maxResults {
			return vector[:maxResults]
		}
		return vector
	}

	scores := make(map[string]float64)
	entries := make(map[string]domain.ProjectKnowledge)

	entryKey := func(k domain.ProjectKnowledge) string {
		if k.ID != "" {
			return k.ID
		}
		return k.ProjectPath + "|" + string(k.Kind) + "|" + k.Topic
	}

	for rank, item := range lexical {
		key := entryKey(item)
		scores[key] += 1.0 / (rrfConstant + float64(rank+1))
		entries[key] = item
	}

	for rank, item := range vector {
		key := entryKey(item)
		scores[key] += 1.0 / (rrfConstant + float64(rank+1))
		if _, exists := entries[key]; !exists {
			entries[key] = item
		}
	}

	candidates := make([]rrfCandidate, 0, len(entries))
	for key, item := range entries {
		candidates = append(candidates, rrfCandidate{
			entry: item,
			score: scores[key],
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	limit := len(candidates)
	if maxResults > 0 && limit > maxResults {
		limit = maxResults
	}

	out := make([]domain.ProjectKnowledge, limit)
	for i := 0; i < limit; i++ {
		out[i] = candidates[i].entry
	}
	return out
}
