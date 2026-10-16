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

func TestPolicy_PlanFilesFoldersMustStayInsideTheProject(t *testing.T) {
	ok := func(dirs ...string) domain.ProjectPolicy {
		return domain.ProjectPolicy{PlanFiles: &domain.PlanFilesPolicy{Plans: dirs}}
	}
	for _, good := range [][]string{{"docs/plans"}, {"a", "b/c"}, {"./docs/tasks/"}, {`docs\plans`}, nil} {
		if err := ok(good...).Validate(); err != nil {
			t.Errorf("%v should be valid: %v", good, err)
		}
	}
	for name, bad := range map[string][]string{
		"blank":     {" "},
		"absolute":  {"/etc"},
		"backslash": {`\share\x`},
		"drive":     {"C:/plans"},
		"parent":    {"../outside"},
		"nested up": {"docs/../../x"},
		"too long":  {strings.Repeat("a", 300)},
		"too many":  make([]string, domain.MaxPlanFolders+1),
	} {
		if bad != nil && name == "too many" {
			for i := range bad {
				bad[i] = "d"
			}
		}
		if err := ok(bad...).Validate(); err == nil {
			t.Errorf("%s: %v must be refused", name, bad)
		}
	}
	// the same rules apply to task folders
	if err := (domain.ProjectPolicy{PlanFiles: &domain.PlanFilesPolicy{Tasks: []string{"../x"}}}).Validate(); err == nil {
		t.Error("a task folder outside the project must be refused")
	}
}

func TestQuotaThresholdPolicy(t *testing.T) {
	if got := (domain.ProjectPolicy{}).QuotaLimit(); got != domain.DefaultQuotaThreshold {
		t.Errorf("unset = %v, want the default %v", got, domain.DefaultQuotaThreshold)
	}
	if got := (domain.ProjectPolicy{QuotaThreshold: 0.75}).QuotaLimit(); got != 0.75 {
		t.Errorf("set = %v", got)
	}
	for _, bad := range []float64{-0.1, 1.01, 7} {
		if err := (domain.ProjectPolicy{QuotaThreshold: bad}).Validate(); err == nil {
			t.Errorf("quotaThreshold %v must be rejected", bad)
		}
	}
	for _, ok := range []float64{0, 0.5, 1} {
		if err := (domain.ProjectPolicy{QuotaThreshold: ok}).Validate(); err != nil {
			t.Errorf("quotaThreshold %v: %v", ok, err)
		}
	}
}
