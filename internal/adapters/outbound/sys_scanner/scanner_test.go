package sys_scanner_test

import (
	"context"
	"os/exec"
	"sort"
	"testing"
	"time"

	sys_scanner "nexus-orchestrator/internal/adapters/outbound/sys_scanner"
)

// TestScanner_Scan_ReturnsWithoutCrash verifies that Scan completes without
// panicking even when no AI providers are installed on the system.
func TestScanner_Scan_ReturnsWithoutCrash(t *testing.T) {
	s := sys_scanner.New()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := s.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan returned unexpected error: %v", err)
	}
}

// TestScanner_Scan_CompletesWithinTimeout verifies that all probes finish
// within 10 seconds even if none of the target endpoints are reachable.
func TestScanner_Scan_CompletesWithinTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow scan test in short mode")
	}
	s := sys_scanner.New()
	done := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, err := s.Scan(ctx)
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Scan returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("Scan did not complete within 10 seconds")
	}
}

// TestScanner_Scan_CLIProbe_WithKnownBinary verifies that when a known binary
// (go) is present in PATH, Scan does not crash or return an error.
func TestScanner_Scan_CLIProbe_WithKnownBinary(t *testing.T) {
	path, err := exec.LookPath("go")
	if err != nil || path == "" {
		t.Skip("go binary not in PATH, skipping")
	}
	s := sys_scanner.New()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, scanErr := s.Scan(ctx)
	if scanErr != nil {
		t.Fatalf("Scan returned unexpected error: %v", scanErr)
	}
}

// TestScanner_Scan_ResultsAreSorted verifies that the providers returned by
// Scan are sorted alphabetically by Name.
func TestScanner_Scan_ResultsAreSorted(t *testing.T) {
	s := sys_scanner.New()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	results, err := s.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan returned unexpected error: %v", err)
	}
	if !sort.SliceIsSorted(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	}) {
		names := make([]string, len(results))
		for i, r := range results {
			names[i] = r.Name
		}
		t.Errorf("scan results not sorted by name: %v", names)
	}
}

// TestScanner_Scan_ContextCancel verifies that a pre-cancelled context causes
// Scan to return promptly (within 3 seconds) rather than blocking.
func TestScanner_Scan_ContextCancel(t *testing.T) {
	s := sys_scanner.New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan struct{})
	go func() {
		_, _ = s.Scan(ctx)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Error("Scan did not return promptly after context cancellation")
	}
}
