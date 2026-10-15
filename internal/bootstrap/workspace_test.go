package bootstrap

import (
	"os"
	"path/filepath"
	"testing"
)

// The daemon watches its working directory for changes only when that is a project. A desktop
// app launched from Finder starts in "/", which is not.
func TestIsProjectDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	project := t.TempDir()
	if isProjectDir(project) {
		t.Error("an empty directory is not a project")
	}
	for _, marker := range []string{".git", ".nexus", "go.mod", "package.json", "Cargo.toml", "pyproject.toml"} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, marker), nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if !isProjectDir(dir) {
			t.Errorf("a directory with %s is a project", marker)
		}
	}
	if isProjectDir(string(filepath.Separator)) {
		t.Error("the filesystem root is never a project")
	}
	// Even with a marker, the home directory is too big to watch.
	if err := os.WriteFile(filepath.Join(home, ".git"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if isProjectDir(home) {
		t.Error("the home directory is never a project")
	}
}
