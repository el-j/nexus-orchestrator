package cli_test

import (
	"errors"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

func TestBrainIngest_DefaultsTheKindAndReportsEveryFailure(t *testing.T) {
	t.Run("a single entry without a kind is a learning", func(t *testing.T) {
		brain := &mockBrainService{}
		out, err := execute(&mockOrchestrator{}, brain, "brain", "ingest", "--project", "/p", "--topic", "t", "--content", "c")
		if err != nil || !strings.Contains(out, "Ingested knowledge entry") {
			t.Fatalf("out=%q err=%v", out, err)
		}
		if brain.ingested.Kind != domain.KnowledgeLearning {
			t.Errorf("kind = %q, want learning", brain.ingested.Kind)
		}
	})
	for name, tc := range map[string]struct {
		brain *mockBrainService
		args  []string
		want  string
	}{
		"the entry cannot be stored": {&mockBrainService{ingestKnowledgeErr: errors.New("disk full")},
			[]string{"brain", "ingest", "--project", "/p", "--topic", "t", "--content", "c"}, "disk full"},
		"neither a file nor content": {&mockBrainService{},
			[]string{"brain", "ingest", "--project", "/p"}, "--file or --content is required"},
		"the file cannot be read": {&mockBrainService{ingestFromFileErr: errors.New("no such file")},
			[]string{"brain", "ingest", "--project", "/p", "--file", "x.md"}, "no such file"},
		"the search fails": {&mockBrainService{searchErr: errors.New("index missing")},
			[]string{"brain", "search", "--project", "/p", "--query", "q"}, "index missing"},
		"the project cannot be initialised": {&mockBrainService{initProjectErr: errors.New("read-only")},
			[]string{"brain", "init", "--project", "/p"}, "read-only"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := execute(&mockOrchestrator{}, tc.brain, tc.args...); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want it to contain %q", err, tc.want)
			}
		})
	}
}
