//go:build !windows

package sys_scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

// fakeBinary writes an executable file called name into dir.
func fakeBinary(t *testing.T, dir, name string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestProbeCLI_PrefersTheRealBinaryOverATerminalShim(t *testing.T) {
	root := t.TempDir()
	shim := fakeBinary(t, filepath.Join(root, ".cmuxterm", "cmux-cli-shims", "SESSION-1"), "tool")
	real := fakeBinary(t, filepath.Join(root, "real-bin"), "tool")
	t.Setenv("PATH", filepath.Dir(shim)+string(os.PathListSeparator)+filepath.Dir(real))

	got, err := newTestScanner().probeCLI(context.Background(), cliTarget{binary: "tool", name: "Tool", kind: domain.ProviderKindCLI})
	if err != nil || len(got) != 1 {
		t.Fatalf("got %v, %v", got, err)
	}
	if got[0].CLIPath != real {
		t.Errorf("CLIPath = %q, want the real binary %q (a shim directory is per terminal session)", got[0].CLIPath, real)
	}
}

func TestProbeCLI_FallsBackToTheShimWhenItIsTheOnlyCopy(t *testing.T) {
	shim := fakeBinary(t, filepath.Join(t.TempDir(), "cmux-cli-shims", "S"), "tool")
	t.Setenv("PATH", filepath.Dir(shim))
	got, _ := newTestScanner().probeCLI(context.Background(), cliTarget{binary: "tool", name: "Tool"})
	if len(got) != 1 || got[0].CLIPath != shim {
		t.Errorf("only a shim exists, it must still be found: %v", got)
	}
}
