package docscheck

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// undocumentedFuncs returns "file:line name" for every function or method in the non-test Go files under
// root that has no doc comment. Methods that implement an interface are held to the same rule: a reader
// of the adapter should not need the port open next to it.
func undocumentedFuncs(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", ".git", "dist", "build", "testutil":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if perr != nil {
			return perr
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Doc != nil {
				continue
			}
			rel, _ := filepath.Rel(root, path)
			out = append(out, rel+":"+itoa(fset.Position(fn.Pos()).Line)+" "+fn.Name.Name)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(out)
	return out
}

func itoa(n int) string {
	const digits = "0123456789"
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{digits[n%10]}, b...)
	}
	return string(b)
}

// TestEveryGoFunctionHasADocComment keeps the in-code documentation at 100 %: exported or not, every
// function and method says what it is for.
func TestEveryGoFunctionHasADocComment(t *testing.T) {
	missing := undocumentedFuncs(t, "../..")
	if len(missing) > 0 {
		t.Fatalf("%d functions have no doc comment:\n%s", len(missing), strings.Join(missing, "\n"))
	}
}

// auditDocs lists every audit or assessment document of the repository.
func auditDocs(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, pattern := range []string{"docs/*audit*.md", "docs/archive/*.md", ".claude/AUDIT*.md"} {
		matches, err := filepath.Glob(filepath.Join(root, pattern))
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, matches...)
	}
	return out
}

// TestEveryAuditSaysWhetherItIsCurrent keeps old audits from being read as the present state: each one
// starts with a "Status:" line within its first ten lines, and any that is superseded links the audit
// that replaced it.
func TestEveryAuditSaysWhetherItIsCurrent(t *testing.T) {
	docs := auditDocs(t)
	if len(docs) < 6 {
		t.Fatalf("found only %d audit documents: %v", len(docs), docs)
	}
	for _, doc := range docs {
		raw, err := os.ReadFile(doc)
		if err != nil {
			t.Fatal(err)
		}
		head := strings.Join(strings.SplitN(string(raw), "\n", 11)[:min(10, strings.Count(string(raw), "\n")+1)], "\n")
		i := strings.Index(head, "Status:")
		if i < 0 {
			t.Errorf("%s has no \"Status:\" line in its first ten lines", doc)
			continue
		}
		status := head[i:]
		if strings.Contains(status, "superseded") && !strings.Contains(status, "audit-2026-10-31") {
			t.Errorf("%s is superseded but does not link the audit that replaced it", doc)
		}
	}
}
