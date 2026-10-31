package docscheck

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A workflow that runs a script which no longer exists fails only when it runs. The publish
// workflow did exactly that for weeks, so no release was built. Every ./scripts/ path a
// workflow runs must exist.
func TestWorkflowsOnlyRunScriptsThatExist(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.yml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no workflows found: %v", err)
	}
	ref := regexp.MustCompile(`(?:\./|\s)(scripts/[A-Za-z0-9_.\-/]+\.(?:sh|mjs|js|py))`)
	for _, f := range files {
		b, _ := os.ReadFile(f)
		for _, m := range ref.FindAllStringSubmatch(string(b), -1) {
			if _, err := os.Stat(filepath.Join(root, m[1])); err != nil {
				t.Errorf("%s runs %s, which does not exist", filepath.Base(f), m[1])
			}
		}
	}
}

// The macOS signing step passes an entitlements file to codesign. It used to sit under build/,
// which is not tracked, so the first release with a Developer ID certificate would have failed
// at that step. Every file named after --entitlements must be in the repository.
func TestSigningEntitlementsFileIsTracked(t *testing.T) {
	re := regexp.MustCompile(`--entitlements\s+(\S+)`)
	found := false
	for _, m := range re.FindAllStringSubmatch(read(t, ".github/workflows/publish.yml"), -1) {
		found = true
		if _, err := os.Stat(filepath.Join(root, m[1])); err != nil {
			t.Errorf("publish.yml signs with %s, which is not in the repository", m[1])
		}
		if strings.HasPrefix(m[1], "build/") {
			t.Errorf("%s is under build/, which is ignored by git", m[1])
		}
	}
	if !found {
		t.Skip("no signing step passes entitlements")
	}
}
