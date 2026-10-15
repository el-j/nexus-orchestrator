package services_test

import (
	"os"
	"path/filepath"
	"testing"

	"nexus-orchestrator/internal/core/services"
)

// An agent's working directory is often a folder inside the repository (frontend/, src/). The
// project is the repository, otherwise one project shows up as many.
func TestProjectRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := filepath.Join(home, "code", "app")
	for _, d := range []string{filepath.Join(repo, ".git"), filepath.Join(repo, "web", "src"), filepath.Join(home, "plain", "sub")} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	// a git worktree has a .git file, not a folder
	wt := filepath.Join(home, "code", "wt", "pkg")
	if err := os.MkdirAll(wt, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "code", "wt", ".git"), []byte("gitdir: x"), 0o600); err != nil {
		t.Fatal(err)
	}

	cases := map[string]string{
		repo:                                    repo,
		filepath.Join(repo, "web"):              repo,
		filepath.Join(repo, "web", "src"):       repo,
		filepath.Join(repo, "web", "src") + "/": repo,
		wt:                                      filepath.Join(home, "code", "wt"),
		filepath.Join(home, "plain", "sub"):     filepath.Join(home, "plain", "sub"), // no repository: as it is
		home:                                    home,                                // never above the home directory
		"":                                      "",
		"/does/not/exist":                       "/does/not/exist",
	}
	for in, want := range cases {
		if got := services.ProjectRoot(in); got != want {
			t.Errorf("ProjectRoot(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsScratchDir(t *testing.T) {
	for _, p := range []string{os.TempDir(), filepath.Join(os.TempDir(), "x"), "/tmp/a", "/private/tmp/a", "/var/folders/g_/abc/T", "/private/var/folders/g_/abc/T/x"} {
		if !services.IsScratchDir(p) {
			t.Errorf("%q is a scratch directory", p)
		}
	}
	for _, p := range []string{"", "/Users/me/code/app", "/home/me/tmpfoo", "/tmpfoo"} {
		if services.IsScratchDir(p) {
			t.Errorf("%q is not a scratch directory", p)
		}
	}
}
