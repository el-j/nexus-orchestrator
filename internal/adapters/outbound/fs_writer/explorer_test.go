package fs_writer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriter_ListDir(t *testing.T) {
	dir := t.TempDir()
	w := New()
	if err := w.WriteCodeToFile(dir, "pkg/alpha.go", "package pkg\n"); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteCodeToFile(dir, ".git/HEAD", "ref: refs/heads/main\n"); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteCodeToFile(dir, "README.md", "# Demo\n"); err != nil {
		t.Fatal(err)
	}

	entries, err := w.ListDir(dir, "")
	if err != nil {
		t.Fatalf("ListDir root: %v", err)
	}
	joined := strings.Join(entries, ",")
	if !strings.Contains(joined, "pkg/") || !strings.Contains(joined, "README.md") {
		t.Fatalf("expected pkg/ and README.md, got %v", entries)
	}
	if strings.Contains(joined, ".git") {
		t.Fatalf("expected .git to be skipped, got %v", entries)
	}

	if _, err := w.ListDir(dir, "../escape"); err == nil {
		t.Fatal("expected traversal error")
	}
	if _, err := w.ListDir(dir, "missing-dir"); err == nil {
		t.Fatal("expected error for missing directory")
	}
}

func TestWriter_ListDir_CapsEntries(t *testing.T) {
	dir := t.TempDir()
	w := New()
	for i := 0; i < maxListDirEntries+5; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%03d.txt", i)), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := w.ListDir(dir, ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != maxListDirEntries {
		t.Fatalf("expected %d entries, got %d", maxListDirEntries, len(entries))
	}
}

func TestWriter_SearchFiles(t *testing.T) {
	dir := t.TempDir()
	w := New()
	longLine := "prefix " + strings.Repeat("a", 200) + " TargetSymbol suffix"
	if err := w.WriteCodeToFile(dir, "internal/service.go", "package internal\n// TargetSymbol here\n"+longLine+"\n"); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteCodeToFile(dir, "targetsymbol_doc.md", "docs\n"); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteCodeToFile(dir, "node_modules/skip.js", "TargetSymbol\n"); err != nil {
		t.Fatal(err)
	}
	// Oversized file should be skipped during content scan.
	if err := w.WriteCodeToFile(dir, "huge.txt", strings.Repeat("TargetSymbol\n", 50000)); err != nil {
		t.Fatal(err)
	}

	if _, err := w.SearchFiles(dir, "   "); err == nil {
		t.Fatal("expected error on blank query")
	}
	if _, err := w.SearchFiles(filepath.Join(dir, "nonexistent"), "TargetSymbol"); err == nil {
		t.Fatal("expected error on missing project root")
	}

	hits, err := w.SearchFiles(dir, "targetsymbol")
	if err != nil {
		t.Fatalf("SearchFiles: %v", err)
	}
	joined := strings.Join(hits, "\n")
	if !strings.Contains(joined, "internal/service.go:2:") || !strings.Contains(joined, "targetsymbol_doc.md") {
		t.Fatalf("unexpected hits: %v", hits)
	}
	if !strings.Contains(joined, "...") {
		t.Fatalf("expected truncated long line in hits: %v", hits)
	}
	if strings.Contains(joined, "node_modules") || strings.Contains(joined, "huge.txt") {
		t.Fatalf("expected skipped dirs and huge files to be excluded: %v", hits)
	}
}

func TestWriter_SearchFiles_CapsMatches(t *testing.T) {
	dir := t.TempDir()
	w := New()
	var sb strings.Builder
	for i := 0; i < maxSearchMatches+10; i++ {
		sb.WriteString("needle line\n")
	}
	if err := w.WriteCodeToFile(dir, "many.txt", sb.String()); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteCodeToFile(dir, "zz_needle.txt", "extra\n"); err != nil {
		t.Fatal(err)
	}
	hits, err := w.SearchFiles(dir, "needle")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != maxSearchMatches {
		t.Fatalf("expected %d matches, got %d", maxSearchMatches, len(hits))
	}

	// Cap filename matches directly and tolerate unreadable subdirectories.
	dir2 := t.TempDir()
	unreadable := filepath.Join(dir2, "00_locked")
	if err := os.Mkdir(unreadable, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(unreadable, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(unreadable, 0o700) })
	for i := 0; i < maxSearchMatches+5; i++ {
		if err := os.WriteFile(filepath.Join(dir2, fmt.Sprintf("fname_%03d.txt", i)), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	hits2, err := w.SearchFiles(dir2, "fname_")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits2) != maxSearchMatches {
		t.Fatalf("expected %d filename matches, got %d", maxSearchMatches, len(hits2))
	}
}

func TestScanFileForNeedle_OpenError(t *testing.T) {
	got := scanFileForNeedle("/nonexistent/path/file.go", "file.go", "x", nil)
	if len(got) != 0 {
		t.Fatalf("expected no matches on open error, got %v", got)
	}
}
