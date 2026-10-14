package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

func writeFile(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadProjectPolicy(t *testing.T) {
	t.Run("absent file is not an error", func(t *testing.T) {
		_, ok, err := LoadProjectPolicy(t.TempDir())
		if ok || err != nil {
			t.Fatalf("ok=%v err=%v, want false,nil", ok, err)
		}
	})
	t.Run("valid file", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, PolicyFile, `{"verify":"make test","maxFileLines":300,"testsFirst":true,"rules":["be small"]}`)
		p, ok, err := LoadProjectPolicy(root)
		if err != nil || !ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
		if p.Verify != "make test" || p.MaxFileLines != 300 || !p.TestsFirst || len(p.Rules) != 1 {
			t.Errorf("unexpected policy %+v", p)
		}
	})
	broken := map[string]string{
		"not json":          `{`,
		"unknown field":     `{"maxFileLine":10}`,
		"invalid value":     `{"maxFileLines":-3}`,
		"trailing nonsense": `{"verify":"x"} oops`,
	}
	for name, body := range broken {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, PolicyFile, body)
			if _, ok, err := LoadProjectPolicy(root); err == nil || ok {
				t.Fatalf("ok=%v err=%v, want an error", ok, err)
			}
		})
	}
	t.Run("directory instead of file", func(t *testing.T) {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, ".nexus", "policy.json"), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, _, err := LoadProjectPolicy(root); err == nil {
			t.Fatal("expected an error")
		}
	})
	t.Run("oversized file", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, PolicyFile, `{"rules":["`+strings.Repeat("a", maxPolicyBytes)+`"]}`)
		if _, _, err := LoadProjectPolicy(root); err == nil {
			t.Fatal("expected an error")
		}
	})
	t.Run("unreadable path component", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, ".nexus", "i am a file, not a directory")
		if _, _, err := LoadProjectPolicy(root); err == nil {
			t.Fatal("expected an error")
		}
	})
}

func TestDefaultPolicyDetectsTheStack(t *testing.T) {
	cases := map[string]struct {
		files []string
		want  string
	}{
		"go":     {[]string{"go.mod"}, "go vet ./... && go test ./..."},
		"node":   {[]string{"package.json"}, "npm test"},
		"rust":   {[]string{"Cargo.toml"}, "cargo test"},
		"python": {[]string{"pyproject.toml"}, "pytest"},
		"pyreq":  {[]string{"requirements.txt"}, "pytest"},
		"mixed":  {[]string{"go.mod", "package.json"}, "go vet ./... && go test ./... && npm test"},
		"none":   {nil, ""},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			for _, f := range c.files {
				writeFile(t, root, f, "x")
			}
			p := DefaultPolicy(root)
			if p.Verify != c.want {
				t.Errorf("Verify = %q, want %q", p.Verify, c.want)
			}
			if err := p.Validate(); err != nil {
				t.Errorf("default policy is invalid: %v", err)
			}
			if !p.TestsFirst || p.MaxFileLines != 500 || !p.Remember || len(p.Rules) == 0 {
				t.Errorf("unexpected defaults %+v", p)
			}
		})
	}
}

func TestEnsureProjectPolicy(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module x")
	created, err := EnsureProjectPolicy(root)
	if err != nil || !created {
		t.Fatalf("created=%v err=%v", created, err)
	}
	if _, ok, err := LoadProjectPolicy(root); !ok || err != nil {
		t.Fatalf("the written policy must load: ok=%v err=%v", ok, err)
	}
	// An existing file, even a customised one, is never overwritten.
	writeFile(t, root, PolicyFile, `{"verify":"mine"}`)
	created, err = EnsureProjectPolicy(root)
	if err != nil || created {
		t.Fatalf("created=%v err=%v, want false,nil", created, err)
	}
	p, _, _ := LoadProjectPolicy(root)
	if p.Verify != "mine" {
		t.Errorf("existing policy was overwritten: %+v", p)
	}
	t.Run("cannot create", func(t *testing.T) {
		blocked := t.TempDir()
		writeFile(t, blocked, ".nexus", "a file")
		if _, err := EnsureProjectPolicy(blocked); err == nil {
			t.Fatal("expected an error when .nexus is a file")
		}
	})
}

func TestRenderPolicyStatesEveryRule(t *testing.T) {
	out := renderPolicy(domain.ProjectPolicy{Verify: "make test", MaxFileLines: 200, TestsFirst: true, Rules: []string{" keep it small "}})
	for _, want := range []string{"## Workflow rules", "`make test`", "200 lines", "Tests first", "- keep it small\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered policy lacks %q:\n%s", want, out)
		}
	}
	bare := renderPolicy(domain.ProjectPolicy{})
	if strings.Contains(bare, "Verification") || strings.Contains(bare, "lines") || strings.Contains(bare, "Tests first") {
		t.Errorf("an empty policy must state no rules:\n%s", bare)
	}
}

func TestCountLines(t *testing.T) {
	for in, want := range map[string]int{"": 0, "a": 1, "a\n": 1, "a\nb": 2, "a\nb\n": 2, "\n\n": 2} {
		if got := countLines(in); got != want {
			t.Errorf("countLines(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestPolicyWriteViolation(t *testing.T) {
	p := domain.ProjectPolicy{TestsFirst: true, MaxFileLines: 3}
	if v := policyWriteViolation(p, "a.go", "x\n", false); !strings.Contains(v, "tests first") {
		t.Errorf("source before any test must be refused, got %q", v)
	}
	if v := policyWriteViolation(p, "a.go", "x\n", true); v != "" {
		t.Errorf("source after a test is allowed, got %q", v)
	}
	if v := policyWriteViolation(p, "a_test.go", strings.Repeat("x\n", 50), false); v != "" {
		t.Errorf("tests are neither ordered nor size-limited, got %q", v)
	}
	if v := policyWriteViolation(p, "README.md", strings.Repeat("x\n", 50), false); v != "" {
		t.Errorf("non-code files are exempt, got %q", v)
	}
	if v := policyWriteViolation(p, "a.go", "1\n2\n3\n4\n", true); !strings.Contains(v, "4 lines, the limit is 3") {
		t.Errorf("an oversized source file must be refused, got %q", v)
	}
	if v := policyWriteViolation(domain.ProjectPolicy{}, "a.go", strings.Repeat("x\n", 9000), false); v != "" {
		t.Errorf("an empty policy allows everything, got %q", v)
	}
}
