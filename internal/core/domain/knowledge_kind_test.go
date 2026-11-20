package domain_test

import (
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

func TestKnowledgeKindIsValid(t *testing.T) {
	for _, k := range []domain.KnowledgeKind{domain.KnowledgeArchitecture, domain.KnowledgeConvention, domain.KnowledgeFileMap,
		domain.KnowledgeLearning, domain.KnowledgeGlossary, domain.KnowledgeTaskSummary} {
		if !k.IsValid() {
			t.Errorf("%q is a kind", k)
		}
	}
	for _, k := range []domain.KnowledgeKind{"", "gossip", "Learning"} {
		if k.IsValid() {
			t.Errorf("%q is not a kind", k)
		}
	}
}

func TestModelTierIsValid(t *testing.T) {
	for _, tier := range []domain.ModelTier{domain.ModelTierFrontier, domain.ModelTierStandard, domain.ModelTierLocal} {
		if !tier.IsValid() {
			t.Errorf("%q is a valid model tier", tier)
		}
	}
	for _, tier := range []domain.ModelTier{"", "supercomputer", "unknown", "Frontier"} {
		if tier.IsValid() {
			t.Errorf("%q is not a valid model tier", tier)
		}
	}
}
