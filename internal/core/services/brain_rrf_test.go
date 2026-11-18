package services_test

import (
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

func TestFuseRRF_Empty(t *testing.T) {
	res := services.FuseRRF(nil, nil, 10)
	if len(res) != 0 {
		t.Errorf("expected empty slice, got %d", len(res))
	}
}

func TestFuseRRF_OneSourceEmpty(t *testing.T) {
	lex := []domain.ProjectKnowledge{
		{ID: "1", Topic: "Lex1"},
		{ID: "2", Topic: "Lex2"},
	}
	vec := []domain.ProjectKnowledge{
		{ID: "3", Topic: "Vec1"},
		{ID: "4", Topic: "Vec2"},
	}

	// Lexical only with maxResults
	res := services.FuseRRF(lex, nil, 1)
	if len(res) != 1 || res[0].ID != "1" {
		t.Errorf("expected top 1 from lexical, got %+v", res)
	}

	// Lexical only without maxResults limit
	resAll := services.FuseRRF(lex, nil, 0)
	if len(resAll) != 2 {
		t.Errorf("expected 2 from lexical, got %d", len(resAll))
	}

	// Vector only with maxResults
	resVec := services.FuseRRF(nil, vec, 1)
	if len(resVec) != 1 || resVec[0].ID != "3" {
		t.Errorf("expected top 1 from vector, got %+v", resVec)
	}

	// Vector only without maxResults limit
	resVecAll := services.FuseRRF(nil, vec, 0)
	if len(resVecAll) != 2 {
		t.Errorf("expected 2 from vector, got %d", len(resVecAll))
	}
}

func TestFuseRRF_HybridFusion(t *testing.T) {
	// Item A is #1 in lexical, #2 in vector -> highest RRF score
	itemA := domain.ProjectKnowledge{ID: "A", Topic: "CommonTop"}
	// Item B is #2 in lexical only
	itemB := domain.ProjectKnowledge{ID: "B", Topic: "LexOnly"}
	// Item C is #1 in vector only
	itemC := domain.ProjectKnowledge{ID: "C", Topic: "VecOnly"}

	lexical := []domain.ProjectKnowledge{itemA, itemB}
	vector := []domain.ProjectKnowledge{itemC, itemA}

	// Fused score for A: 1/(60+1) + 1/(60+2) = 1/61 + 1/62 ≈ 0.01639 + 0.01613 = 0.03252
	// Fused score for C: 1/(60+1) = 1/61 ≈ 0.01639
	// Fused score for B: 1/(60+2) = 1/62 ≈ 0.01613
	// Expected ranking: A, C, B
	fused := services.FuseRRF(lexical, vector, 10)
	if len(fused) != 3 {
		t.Fatalf("expected 3 fused items, got %d", len(fused))
	}
	if fused[0].ID != "A" {
		t.Errorf("expected #1 to be A, got %s", fused[0].ID)
	}
	if fused[1].ID != "C" {
		t.Errorf("expected #2 to be C, got %s", fused[1].ID)
	}
	if fused[2].ID != "B" {
		t.Errorf("expected #3 to be B, got %s", fused[2].ID)
	}

	// Truncated to maxResults
	fusedTrunc := services.FuseRRF(lexical, vector, 2)
	if len(fusedTrunc) != 2 || fusedTrunc[0].ID != "A" || fusedTrunc[1].ID != "C" {
		t.Errorf("expected top 2 [A, C], got %+v", fusedTrunc)
	}

	// Fallback key when ID is empty
	itemNoID1 := domain.ProjectKnowledge{ProjectPath: "/p", Kind: domain.KnowledgeArchitecture, Topic: "Arch"}
	itemNoID2 := domain.ProjectKnowledge{ProjectPath: "/p", Kind: domain.KnowledgeArchitecture, Topic: "Arch"}
	fusedNoID := services.FuseRRF([]domain.ProjectKnowledge{itemNoID1}, []domain.ProjectKnowledge{itemNoID2}, 5)
	if len(fusedNoID) != 1 {
		t.Errorf("expected 1 deduplicated item for empty ID, got %d", len(fusedNoID))
	}
}
