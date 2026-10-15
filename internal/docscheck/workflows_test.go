package docscheck

import (
	"os"
	"path/filepath"
	"regexp"
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
