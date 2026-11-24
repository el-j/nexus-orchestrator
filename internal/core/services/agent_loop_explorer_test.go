package services_test

import (
	"errors"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

type explorerWriter struct {
	*memFS
	dirs      map[string][]string
	dirErr    error
	hits      []string
	searchErr error
}

func (e *explorerWriter) ListDir(_, relDir string) ([]string, error) {
	if e.dirErr != nil {
		return nil, e.dirErr
	}
	return e.dirs[relDir], nil
}

func (e *explorerWriter) SearchFiles(_, _ string) ([]string, error) {
	if e.searchErr != nil {
		return nil, e.searchErr
	}
	return e.hits, nil
}

func TestAgentLoop_ListDirAndSearchFiles(t *testing.T) {
	w := &explorerWriter{
		memFS: newMemFS(map[string]string{}),
		dirs:  map[string][]string{"": {"cmd/", "go.mod"}, "empty": {}},
		hits:  []string{"cmd/main.go:10: func main()"},
	}
	llm := &scriptLLM{
		name:  "ExplorerLLM",
		alive: true,
		limit: 100000,
		replies: []string{
			tool(`{"tool":"list_dir","path":""}`) + "\n" +
				tool(`{"tool":"list_dir","path":"empty"}`) + "\n" +
				tool(`{"tool":"search_files","query":"main"}`) + "\n" +
				tool(`{"tool":"search_files","query":""}`),
			tool(`{"tool":"finish","summary":"explored workspace"}`),
		},
	}
	e := newEngine(t, w, nil, nil, llm)
	id := e.queue(domain.Task{ProjectPath: t.TempDir(), Instruction: "explore project", Agentic: true})
	got := e.runOne(id, domain.StatusCompleted)
	if got.Status != domain.StatusCompleted {
		t.Fatalf("status = %s, logs:\n%s", got.Status, got.Logs)
	}
	obs := llm.histories[1][len(llm.histories[1])-1].Content
	for _, want := range []string{"cmd/", "go.mod", "(empty directory)", "cmd/main.go:10: func main()", "query is required"} {
		if !strings.Contains(obs, want) {
			t.Errorf("turn 2 observation missing %q:\n%s", want, obs)
		}
	}
}

func TestAgentLoop_ExplorerErrorsAndNoHits(t *testing.T) {
	w := &explorerWriter{
		memFS:     newMemFS(map[string]string{}),
		dirErr:    errors.New("dir boom"),
		searchErr: errors.New("search boom"),
	}
	llm := &scriptLLM{
		name:  "ErrLLM",
		alive: true,
		limit: 100000,
		replies: []string{
			tool(`{"tool":"list_dir","path":"sub"}`) + "\n" + tool(`{"tool":"search_files","query":"foo"}`),
			tool(`{"tool":"finish","summary":"handled errors"}`),
		},
	}
	e := newEngine(t, w, nil, nil, llm)
	id := e.queue(domain.Task{ProjectPath: t.TempDir(), Instruction: "check err", Agentic: true})
	e.runOne(id, domain.StatusCompleted)
	obs := llm.histories[1][len(llm.histories[1])-1].Content
	if !strings.Contains(obs, "list_dir failed: dir boom") || !strings.Contains(obs, "search_files failed: search boom") {
		t.Fatalf("expected error observations, got:\n%s", obs)
	}

	w2 := &explorerWriter{memFS: newMemFS(map[string]string{})}
	llm2 := &scriptLLM{
		name:  "EmptyLLM",
		alive: true,
		limit: 100000,
		replies: []string{
			tool(`{"tool":"search_files","query":"nothing"}`),
			tool(`{"tool":"finish","summary":"done"}`),
		},
	}
	e2 := newEngine(t, w2, nil, nil, llm2)
	id2 := e2.queue(domain.Task{ProjectPath: t.TempDir(), Instruction: "no hits", Agentic: true})
	e2.runOne(id2, domain.StatusCompleted)
	obs2 := llm2.histories[1][len(llm2.histories[1])-1].Content
	if !strings.Contains(obs2, "no matches found") {
		t.Fatalf("expected no matches found, got:\n%s", obs2)
	}
}

func TestAgentLoop_ExplorerNotAvailable(t *testing.T) {
	llm := &scriptLLM{
		name:  "NoExpLLM",
		alive: true,
		limit: 100000,
		replies: []string{
			tool(`{"tool":"list_dir","path":""}`) + "\n" + tool(`{"tool":"search_files","query":"foo"}`),
			tool(`{"tool":"finish","summary":"done"}`),
		},
	}
	e := newEngine(t, newMemFS(map[string]string{}), nil, nil, llm)
	id := e.queue(domain.Task{ProjectPath: t.TempDir(), Instruction: "no explorer", Agentic: true})
	e.runOne(id, domain.StatusCompleted)
	obs := llm.histories[1][len(llm.histories[1])-1].Content
	if !strings.Contains(obs, "list_dir: workspace explorer is not available") ||
		!strings.Contains(obs, "search_files: workspace explorer is not available") {
		t.Fatalf("expected unavailable messages, got:\n%s", obs)
	}
}
