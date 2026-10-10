package fs_writer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nexus-orchestrator/internal/adapters/outbound/fs_writer"
)

func TestWriteCodeToFile_CreatesNestedDirsAndRejectsLexicalEscapes(t *testing.T) {
	root := t.TempDir()
	w := fs_writer.New()
	if err := w.WriteCodeToFile(root, "a/b/c.go", "package c"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "a", "b", "c.go")); string(b) != "package c" {
		t.Errorf("content %q", b)
	}
	for _, bad := range []string{"../x.go", "a/../../x.go", "/etc/x", filepath.Join(root, "abs.go")} {
		if err := w.WriteCodeToFile(root, bad, "x"); err == nil || !strings.Contains(err.Error(), "path traversal blocked") {
			t.Errorf("%q must be blocked, got %v", bad, err)
		}
	}
	// Overwriting an existing file inside the project is fine.
	if err := w.WriteCodeToFile(root, "a/b/c.go", "v2"); err != nil {
		t.Error(err)
	}
}

func TestSymlinksCannotEscapeTheProjectRoot(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("TOP SECRET"), 0o600); err != nil {
		t.Fatal(err)
	}
	// A symlinked file pointing outside.
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "alias.txt")); err != nil {
		t.Fatal(err)
	}
	w := fs_writer.New()

	for _, rel := range []string{"link/evil.go", "link/sub/dir/evil.go", "alias.txt"} {
		if err := w.WriteCodeToFile(root, rel, "pwned"); err == nil || !strings.Contains(err.Error(), "symlink") {
			t.Errorf("write %q must be blocked: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(outside, "evil.go")); err == nil {
		t.Fatal("a file was written outside the project root")
	}
	if b, _ := os.ReadFile(filepath.Join(outside, "secret.txt")); string(b) != "TOP SECRET" {
		t.Fatalf("an outside file was modified: %q", b)
	}
	// Reading context through a symlink that leaves the project is blocked too.
	if _, err := w.ReadContextFiles(root, []string{"alias.txt"}); err == nil {
		t.Error("context read through an escaping symlink must be blocked")
	}
	if _, err := w.ReadContextFiles(root, []string{"link/secret.txt"}); err == nil {
		t.Error("context read through an escaping directory symlink must be blocked")
	}
}

func TestSymlinksInsideTheProjectAreAllowed(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "real"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "real"), filepath.Join(root, "alias")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	w := fs_writer.New()
	if err := w.WriteCodeToFile(root, "alias/ok.go", "fine"); err != nil {
		t.Fatalf("an in-project symlink is legitimate: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "real", "ok.go")); string(b) != "fine" {
		t.Errorf("content %q", b)
	}
}

func TestProjectRootItselfMayBeASymlink(t *testing.T) {
	real := t.TempDir()
	link := filepath.Join(t.TempDir(), "proj")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := fs_writer.New().WriteCodeToFile(link, "x.go", "ok"); err != nil {
		t.Fatalf("macOS /tmp and similar roots are symlinks: %v", err)
	}
}

func TestReadContextFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("A"), 0o600); err != nil {
		t.Fatal(err)
	}
	w := fs_writer.New()
	got, err := w.ReadContextFiles(root, []string{"a.go"})
	if err != nil || got != "// --- a.go ---\nA\n" {
		t.Errorf("%q %v", got, err)
	}
	if _, err := w.ReadContextFiles(root, []string{"missing.go"}); err == nil {
		t.Error("missing file")
	}
	if _, err := w.ReadContextFiles(root, []string{"../escape"}); err == nil {
		t.Error("traversal")
	}
	if got, err := w.ReadContextFiles(root, nil); err != nil || got != "" {
		t.Errorf("no files: %q %v", got, err)
	}
}

func TestWriteFailureSurfaces(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "file"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	// "file" is a regular file, so it cannot be a directory component.
	if err := fs_writer.New().WriteCodeToFile(root, "file/child.go", "x"); err == nil {
		t.Error("mkdir under a regular file must fail")
	}
	// A directory where the file should be.
	if err := os.MkdirAll(filepath.Join(root, "dir"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := fs_writer.New().WriteCodeToFile(root, "dir", "x"); err == nil {
		t.Error("writing onto a directory must fail")
	}
}
