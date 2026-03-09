package fs_writer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nexus-ai/internal/adapters/outbound/fs_writer"
)

func TestWriter_WriteAndRead(t *testing.T) {
	dir := t.TempDir()
	w := fs_writer.New()

	code := "package main\n\nfunc main() {}\n"
	if err := w.WriteCodeToFile(dir, "main.go", code); err != nil {
		t.Fatalf("WriteCodeToFile: %v", err)
	}

	// File must exist with correct content
	got, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != code {
		t.Errorf("file content mismatch:\ngot:  %q\nwant: %q", string(got), code)
	}
}

func TestWriter_WriteCodeToFile_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	w := fs_writer.New()

	if err := w.WriteCodeToFile(dir, "sub/dir/file.go", "// hi"); err != nil {
		t.Fatalf("WriteCodeToFile nested: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "sub", "dir", "file.go")); err != nil {
		t.Errorf("expected nested file to exist: %v", err)
	}
}

func TestWriter_ReadContextFiles(t *testing.T) {
	dir := t.TempDir()

	// Write two context files
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.go"), []byte("package b"), 0o644); err != nil {
		t.Fatal(err)
	}

	w := fs_writer.New()
	ctx, err := w.ReadContextFiles(dir, []string{"a.go", "b.go"})
	if err != nil {
		t.Fatalf("ReadContextFiles: %v", err)
	}
	if ctx == "" {
		t.Error("expected non-empty context")
	}
	for _, want := range []string{"a.go", "package a", "b.go", "package b"} {
		if !strings.Contains(ctx, want) {
			t.Errorf("context missing %q", want)
		}
	}
}

func TestWriter_ReadContextFiles_MissingFileReturnsError(t *testing.T) {
	dir := t.TempDir()
	w := fs_writer.New()

	_, err := w.ReadContextFiles(dir, []string{"nonexistent.go"})
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}
