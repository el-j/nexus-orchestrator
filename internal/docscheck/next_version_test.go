//go:build !windows

package docscheck

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// repo builds a throwaway git repository holding the release-please manifest and returns a
// function that runs git (or the script) in it.
func repo(t *testing.T, manifest string) (git func(args ...string) string, next func() (string, error)) {
	t.Helper()
	dir := t.TempDir()
	git = func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, ".release-please-manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	git("add", ".")
	git("commit", "-q", "-m", "start")
	next = func() (string, error) {
		script, _ := filepath.Abs(filepath.Join(root, "scripts", "next-version.sh"))
		cmd := exec.Command("bash", script)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "NEXUS_MANIFEST_REF=HEAD")
		out, err := cmd.CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	return git, next
}

func commits(git func(args ...string) string, n int) {
	for i := 0; i < n; i++ {
		git("commit", "-q", "--allow-empty", "-m", "work")
	}
}

// A release number is X.Y.Z. What develop builds are betas of the NEXT minor, numbered by the
// commits since the last stable tag, and never reuse a tag: no "0.10.0-48" nonsense.
func TestNextVersion_BetasOfTheNextMinorCountedFromTheLastStableTag(t *testing.T) {
	git, next := repo(t, `{ ".": "0.9.4" }`)
	git("tag", "v0.9.4")
	commits(git, 3)
	if got, err := next(); err != nil || got != "0.10.0-beta.3" {
		t.Fatalf("got %q, %v; want 0.10.0-beta.3", got, err)
	}

	// an existing tag is never reused
	git("tag", "v0.10.0-beta.3")
	if got, _ := next(); got != "0.10.0-beta.4" {
		t.Errorf("after the tag exists: %q, want 0.10.0-beta.4", got)
	}

	// after the release the base moves on and counting starts again
	if err := os.WriteFile(filepath.Join(gitDir(t, git), ".release-please-manifest.json"), []byte(`{ ".": "0.10.0" }`), 0o600); err != nil {
		t.Fatal(err)
	}
	git("commit", "-q", "-am", "release 0.10.0")
	git("tag", "v0.10.0")
	commits(git, 2)
	if got, _ := next(); got != "0.11.0-beta.2" {
		t.Errorf("after 0.10.0: %q, want 0.11.0-beta.2", got)
	}
}

func gitDir(t *testing.T, git func(args ...string) string) string {
	t.Helper()
	return git("rev-parse", "--show-toplevel")
}

func TestNextVersion_NeverGoesBelowABetaThatExists(t *testing.T) {
	git, next := repo(t, `{ ".": "0.9.4" }`)
	git("tag", "v0.9.4")
	commits(git, 2)
	git("tag", "v0.10.0-beta.175")
	if got, _ := next(); got != "0.10.0-beta.176" {
		t.Errorf("got %q, want 0.10.0-beta.176", got)
	}
}

func TestNextVersion_WithoutAStableTagYetItCountsAllCommits(t *testing.T) {
	git, next := repo(t, `{ ".": "0.9.4" }`)
	commits(git, 4)
	if got, _ := next(); got != "0.10.0-beta.5" {
		t.Errorf("got %q, want 0.10.0-beta.5", got)
	}
}

func TestNextVersion_NeedsAManifestVersion(t *testing.T) {
	_, next := repo(t, `{ "other": "1.0.0" }`)
	if got, err := next(); err == nil || !strings.Contains(got, "no version") {
		t.Errorf("got %q, %v; want an error", got, err)
	}
}

func TestReleaseVersionsAreCleanSemver(t *testing.T) {
	// The publish workflow takes stable versions from release-please only.
	b, err := os.ReadFile(filepath.Join(root, ".release-please-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `".":`) {
		t.Errorf("manifest has no root version: %s", b)
	}
}

// TestWorkflowsDoNotUseGitVersion keeps the numeric-suffix release tags (v0.10.0-48) from coming back.
func TestWorkflowsDoNotUseGitVersion(t *testing.T) {
	files, _ := filepath.Glob("../../.github/workflows/*.yml")
	if len(files) == 0 {
		t.Fatal("no workflows found")
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.ToLower(string(b)), "gitversion") {
			t.Errorf("%s uses GitVersion; versions come from release-please and scripts/next-version.sh", f)
		}
	}
}
