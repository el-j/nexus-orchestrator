package cli

import (
	"os"
	"path/filepath"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

func TestApplyBudgetsAndLadders_StartFromAnEmptyPolicy(t *testing.T) {
	pol := domain.ProjectPolicy{}
	if err := applyBudgets(&pol, []string{"Claude=5000"}); err != nil || pol.Budgets["Claude"].DailyTokens != 5000 {
		t.Errorf("budgets = %+v, %v", pol.Budgets, err)
	}
	if err := applyLadders(&pol, []string{"coder=haiku,sonnet"}); err != nil || len(pol.Ladder["coder"]) != 2 {
		t.Errorf("ladder = %+v, %v", pol.Ladder, err)
	}
}

func TestFindStdioProxy_PrefersTheCopyNextToThisBinary(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Skip("no executable path")
	}
	if resolved, err := filepath.EvalSymlinks(self); err == nil {
		self = resolved
	}
	proxy := filepath.Join(filepath.Dir(self), "nexus-mcp-stdio")
	if _, err := os.Stat(proxy); err == nil {
		t.Skip("a real proxy sits next to the test binary")
	}
	if err := os.WriteFile(proxy, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Skipf("cannot write next to the test binary: %v", err)
	}
	defer os.Remove(proxy)
	if got := findStdioProxy(); got != proxy {
		t.Errorf("findStdioProxy() = %q, want %q", got, proxy)
	}
}
