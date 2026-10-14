package domain_test

import (
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

func TestProjectPolicyValidate(t *testing.T) {
	long := strings.Repeat("x", domain.MaxPolicyRuleLen+1)
	manyRules := make([]string, domain.MaxPolicyRules+1)
	for i := range manyRules {
		manyRules[i] = "rule"
	}
	cases := map[string]struct {
		p       domain.ProjectPolicy
		wantErr string
	}{
		"empty is valid":         {domain.ProjectPolicy{}, ""},
		"full is valid":          {domain.ProjectPolicy{Verify: "go test ./...", MaxFileLines: 500, TestsFirst: true, Rules: []string{"a"}}, ""},
		"negative max lines":     {domain.ProjectPolicy{MaxFileLines: -1}, "maxFileLines"},
		"too many rules":         {domain.ProjectPolicy{Rules: manyRules}, "rules"},
		"rule too long":          {domain.ProjectPolicy{Rules: []string{long}}, "rule"},
		"blank rule":             {domain.ProjectPolicy{Rules: []string{"  "}}, "blank"},
		"unsupported version":    {domain.ProjectPolicy{Version: 99}, "version"},
		"verify with newline":    {domain.ProjectPolicy{Verify: "a\nb"}, "verify"},
		"verify too long":        {domain.ProjectPolicy{Verify: strings.Repeat("x", domain.MaxPolicyVerifyLen+1)}, "verify"},
		"version 1 is supported": {domain.ProjectPolicy{Version: 1}, ""},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			err := c.p.Validate()
			switch {
			case c.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case c.wantErr != "" && (err == nil || !strings.Contains(err.Error(), c.wantErr)):
				t.Fatalf("error = %v, want it to mention %q", err, c.wantErr)
			}
		})
	}
}

func TestFileKinds(t *testing.T) {
	tests := map[string]domain.FileKind{
		"internal/x/foo_test.go":     domain.FileTest,
		"src/foo.test.ts":            domain.FileTest,
		"src/foo.spec.tsx":           domain.FileTest,
		"tests/test_api.py":          domain.FileTest,
		"pkg/api_test.py":            domain.FileTest,
		"src/__tests__/a.js":         domain.FileTest,
		"e2e/login.spec.ts":          domain.FileTest,
		"internal/x/foo.go":          domain.FileSource,
		"src/App.vue":                domain.FileSource,
		"src/lib.rs":                 domain.FileSource,
		"README.md":                  domain.FileOther,
		"package.json":               domain.FileOther,
		".nexus/policy.json":         domain.FileOther,
		"Makefile":                   domain.FileOther,
		"docs/guide.html":            domain.FileOther,
		"SRC/UPPER.GO":               domain.FileSource,
		"testdata/fixture.go":        domain.FileOther,
		"internal/testutil/mocks.go": domain.FileSource,
	}
	for path, want := range tests {
		if got := domain.ClassifyFile(path); got != want {
			t.Errorf("ClassifyFile(%q) = %v, want %v", path, got, want)
		}
	}
}
