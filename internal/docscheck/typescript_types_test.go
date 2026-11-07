package docscheck

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// anyEscape matches the two ways TypeScript code switches the type checker off for a value.
var anyEscape = regexp.MustCompile(`\bas any\b|:\s*any\b|<any>`)

// TestTypeScriptSourcesHaveNoAnyEscapes keeps the "no type issues" rule: application code in the
// four TypeScript packages states its types instead of casting to any. Tests may still use it.
func TestTypeScriptSourcesHaveNoAnyEscapes(t *testing.T) {
	for _, pkg := range []string{"frontend/src", "docs/src", "vscode-extension/src", "github-action/src"} {
		err := filepath.WalkDir(filepath.Join(root, pkg), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "node_modules" || d.Name() == "wailsjs" || d.Name() == "test" {
					return filepath.SkipDir
				}
				return nil
			}
			name := d.Name()
			if filepath.Ext(name) != ".ts" && filepath.Ext(name) != ".vue" ||
				strings.HasSuffix(name, ".spec.ts") || strings.HasSuffix(name, ".test.ts") || strings.HasSuffix(name, ".d.ts") {
				return nil
			}
			for i, line := range strings.Split(read(t, strings.TrimPrefix(filepath.ToSlash(path), root+"/")), "\n") {
				if anyEscape.MatchString(line) {
					t.Errorf("%s:%d casts to any: %s", path, i+1, strings.TrimSpace(line))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
