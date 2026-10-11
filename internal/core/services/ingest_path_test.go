package services_test

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

const goodMD = "# T\nA sufficiently long preamble paragraph here.\n## Section\nThis body is comfortably longer than twenty characters.\n"

// project returns a fresh project directory containing CLAUDE.md.
func project(t *testing.T) (dir, md string) {
	t.Helper()
	dir = t.TempDir()
	md = filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(md, []byte(goodMD), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir, md
}

func ingest(t *testing.T, project, file string) (int, error) {
	t.Helper()
	b := services.NewBrainService(&mockKnowledgeRepo{}, nil)
	return b.IngestFromFile(bg, project, file)
}

func wantRefused(t *testing.T, err error, fragment string) {
	t.Helper()
	if !errors.Is(err, domain.ErrPathNotAllowed) {
		t.Fatalf("want ErrPathNotAllowed, got %v", err)
	}
	if fragment != "" && !strings.Contains(err.Error(), fragment) {
		t.Fatalf("error %q should mention %q", err, fragment)
	}
}

func TestIngestPath_AcceptsAFileInsideTheProject(t *testing.T) {
	dir, md := project(t)
	if n, err := ingest(t, dir, md); err != nil || n == 0 {
		t.Fatalf("absolute path inside project: n=%d err=%v", n, err)
	}
	if n, err := ingest(t, dir, "CLAUDE.md"); err != nil || n == 0 {
		t.Fatalf("relative path is taken from the project: n=%d err=%v", n, err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "a.md"), []byte(goodMD), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ingest(t, dir, "docs/../docs/a.md"); err != nil {
		t.Fatalf("a path that stays inside after cleaning must work: %v", err)
	}
}

func TestIngestPath_RejectsTraversalOutOfTheProject(t *testing.T) {
	dir, _ := project(t)
	secret := filepath.Join(filepath.Dir(dir), "outside-secret.md")
	if err := os.WriteFile(secret, []byte(goodMD), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(secret) })

	_, err := ingest(t, dir, "../outside-secret.md")
	wantRefused(t, err, "outside the project")
	_, err = ingest(t, dir, filepath.Join("docs", "..", "..", "outside-secret.md"))
	wantRefused(t, err, "outside the project")
}

func TestIngestPath_RejectsAnAbsolutePathOutsideTheProject(t *testing.T) {
	dir, _ := project(t)
	otherDir, other := project(t)
	_ = otherDir
	_, err := ingest(t, dir, other)
	wantRefused(t, err, "outside the project")
}

func TestIngestPath_RejectsASiblingDirectorySharingTheProjectPrefix(t *testing.T) {
	parent := t.TempDir()
	app := filepath.Join(parent, "app")
	sibling := filepath.Join(parent, "app-secrets")
	for _, d := range []string{app, sibling} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	file := filepath.Join(sibling, "keys.md")
	if err := os.WriteFile(file, []byte(goodMD), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := ingest(t, app, file)
	wantRefused(t, err, "outside the project")
}

func TestIngestPath_RejectsASymlinkThatEscapesTheProject(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation needs privileges on Windows")
	}
	dir, _ := project(t)
	_, outside := project(t)
	link := filepath.Join(dir, "innocent.md")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	_, err := ingest(t, dir, "innocent.md")
	wantRefused(t, err, "outside the project")

	// A symlinked project directory is resolved too: files are judged by their real location.
	realDir, realMD := project(t)
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(realDir, alias); err != nil {
		t.Fatal(err)
	}
	if n, err := ingest(t, alias, realMD); err != nil || n == 0 {
		t.Fatalf("a file inside a symlinked project must be accepted: n=%d err=%v", n, err)
	}
}

func TestIngestPath_RejectsNonRegularFilesAndOversizedFiles(t *testing.T) {
	dir, _ := project(t)
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o750); err != nil {
		t.Fatal(err)
	}
	_, err := ingest(t, dir, "sub")
	wantRefused(t, err, "not a regular file")

	big := filepath.Join(dir, "big.md")
	if err := os.WriteFile(big, []byte(strings.Repeat("x", services.MaxIngestBytes+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = ingest(t, dir, big)
	wantRefused(t, err, "larger than")

	exact := filepath.Join(dir, "exact.md")
	if err := os.WriteFile(exact, []byte("# T\n"+strings.Repeat("y", services.MaxIngestBytes-4)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ingest(t, dir, exact); errors.Is(err, domain.ErrPathNotAllowed) {
		t.Fatalf("a file of exactly MaxIngestBytes must be allowed: %v", err)
	}
}

func TestIngestPath_RejectsCredentialLocationsEvenInsideTheProject(t *testing.T) {
	dir := t.TempDir()
	for rel, name := range map[string]string{
		".ssh":     "config.md",
		".aws":     "credentials.md",
		".gnupg":   "notes.md",
		".kube":    "config.md",
		".git":     "HEAD.md",
		"":         ".env",
		"sub":      ".env.production",
		"sub/more": ".netrc",
	} {
		d := filepath.Join(dir, rel)
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, name), []byte(goodMD), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := ingest(t, dir, filepath.Join(rel, name))
		wantRefused(t, err, "credentials location")
	}
}

func TestIngestPath_ProjectPathMustBeAUsableDirectory(t *testing.T) {
	dir, md := project(t)
	_, err := ingest(t, "", md)
	wantRefused(t, err, "projectPath is required")
	_, err = ingest(t, "   ", md)
	wantRefused(t, err, "projectPath is required")
	_, err = ingest(t, dir, "")
	wantRefused(t, err, "filePath is required")
	_, err = ingest(t, filepath.Join(dir, "nope"), md)
	wantRefused(t, err, "project directory is not accessible")
	_, err = ingest(t, md, md) // a file, not a directory
	wantRefused(t, err, "not a directory")
	_, err = ingest(t, string(filepath.Separator), md)
	wantRefused(t, err, "filesystem root")
}

func TestIngestPath_ReportsAMissingFileAsNotExistNotAsAPolicyViolation(t *testing.T) {
	dir, _ := project(t)
	_, err := ingest(t, dir, "missing.md")
	if err == nil || errors.Is(err, domain.ErrPathNotAllowed) || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want a not-exist error, got %v", err)
	}
}

func TestIngestPath_AllowListPermitsSharedDirectoriesOnly(t *testing.T) {
	dir, _ := project(t)
	shared := t.TempDir()
	file := filepath.Join(shared, "template.md")
	if err := os.WriteFile(file, []byte(goodMD), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := ingest(t, dir, file)
	wantRefused(t, err, "outside the project")

	t.Setenv(services.IngestAllowedDirsEnv, strings.Join([]string{"", "/does/not/exist", shared}, string(os.PathListSeparator)))
	if n, err := ingest(t, dir, file); err != nil || n == 0 {
		t.Fatalf("allow-listed directory: n=%d err=%v", n, err)
	}
	// The allow-list does not widen access to anything else.
	_, other := project(t)
	_, err = ingest(t, dir, other)
	wantRefused(t, err, "outside the project")
	// ... and credentials stay off-limits even inside an allowed directory.
	if err := os.MkdirAll(filepath.Join(shared, ".ssh"), 0o750); err != nil {
		t.Fatal(err)
	}
	key := filepath.Join(shared, ".ssh", "id.md")
	_ = os.WriteFile(key, []byte(goodMD), 0o600)
	_, err = ingest(t, dir, key)
	wantRefused(t, err, "credentials location")
}

func TestIngestPath_UnreadableFileIsRefusedNotLeaked(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("permission semantics differ")
	}
	dir, md := project(t)
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locked, "a.md"), []byte(goodMD), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o750) })
	_, err := ingest(t, dir, filepath.Join(locked, "a.md"))
	wantRefused(t, err, "not accessible")
	_ = md
}
