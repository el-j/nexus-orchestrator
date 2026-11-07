package sys_scanner

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

func scanPlans(t *testing.T, root string) []domain.DiscoveredPlanFile {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	got, err := New().ScanPlanFiles(context.Background(), []string{root})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestScanPlanFiles_IgnoresFoldersNamedLikeFilesAndFindsInstructionFilesByName(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{".claude/tasks/sub.md", ".cursor/rules/sub.mdc", ".github/agents/sub.agent.md"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(root, "notes.prompt.md"), "just words")
	got := scanPlans(t, root)
	if len(got) != 1 || !strings.HasSuffix(got[0].Path, "notes.prompt.md") {
		t.Errorf("found %+v, want only the prompt file", got)
	}
}

func TestScanPlanFiles_SkipsALinkWhoseTargetIsGone(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating links needs privileges on Windows")
	}
	root := t.TempDir()
	// One link where the folder scan sees it and one in a sub-folder only the recursive search reaches.
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o750); err != nil {
		t.Fatal(err)
	}
	for _, link := range []string{"dangling.prompt.md", "docs/dangling.prompt.md"} {
		if err := os.Symlink(filepath.Join(root, "missing.md"), filepath.Join(root, filepath.FromSlash(link))); err != nil {
			t.Fatal(err)
		}
	}
	if got := scanPlans(t, root); len(got) != 0 {
		t.Errorf("a dangling link is not a plan file: %+v", got)
	}
}

func TestScanPlanFiles_ARootThatCannotBeMadeAbsoluteIsSkipped(t *testing.T) {
	old := absPath
	absPath = func(string) (string, error) { return "", errors.New("no working directory") }
	defer func() { absPath = old }()
	if got := scanPlans(t, "relative"); len(got) != 0 {
		t.Errorf("got %+v", got)
	}
}

func TestFindProjectPath_StopsAtTheFilesystemRoot(t *testing.T) {
	root := string(filepath.Separator)
	if got := findProjectPath(root, t.TempDir()); got != root {
		t.Errorf("got %q, want the starting folder when no .git is found on the way up", got)
	}
}

func TestLooksLikePlanMarkdown_HandlesAMissingAndAnOversizedFile(t *testing.T) {
	if looksLikePlanMarkdown(filepath.Join(t.TempDir(), "missing.md")) {
		t.Error("a missing file is not a plan")
	}
	long := filepath.Join(t.TempDir(), "long.md")
	// Only the first 2 KiB are read: the checklist that proves it is a plan sits before the cut.
	writeFile(t, long, "- [ ] one\n- [x] two\n"+strings.Repeat("filler line\n", 400))
	if !looksLikePlanMarkdown(long) {
		t.Error("a long file with a checklist at the top is a plan")
	}
}

func TestMergeByName_JoinsProbesThatFoundTheSameProviderAndSortsByName(t *testing.T) {
	later := time.Now()
	got := mergeByName([][]domain.DiscoveredProvider{
		{{Name: "Zed", Status: domain.DiscoveryStatusInstalled}, {Name: "Ollama", Status: domain.DiscoveryStatusInstalled, LastSeen: later.Add(-time.Hour)}},
		{{Name: "Ollama", Status: domain.DiscoveryStatusReachable, BaseURL: "http://127.0.0.1:11434", LastSeen: later}},
	})
	if len(got) != 2 || got[0].Name != "Ollama" || got[1].Name != "Zed" {
		t.Fatalf("got %+v", got)
	}
	if got[0].Status != domain.DiscoveryStatusReachable || got[0].BaseURL == "" {
		t.Errorf("the reachable sighting must win the merge: %+v", got[0])
	}
}

func TestProbePort_ASilentListenerThatSpeaksNoHTTPIsNotAProvider(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			_ = c.Close() // accepts, then hangs up without a word
		}
	}()
	port := l.Addr().(*net.TCPAddr).Port
	got, err := New().probePort(context.Background(), portTarget{name: "Odd", port: port, endpoint: "/v1/models", kind: domain.ProviderKindLMStudio})
	if err != nil || len(got) != 0 {
		t.Errorf("got %+v err=%v", got, err)
	}
}

func TestRunTasklist_RunsTheWindowsTaskListCommand(t *testing.T) {
	out, err := runTasklist(context.Background())
	if runtime.GOOS == "windows" {
		if err != nil || len(out) == 0 {
			t.Errorf("tasklist: %v", err)
		}
		return
	}
	if err == nil {
		t.Error("tasklist does not exist outside Windows")
	}
}
