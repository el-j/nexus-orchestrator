// Package fs_writer implements the FileWriter port, writing LLM-generated
// code to the local filesystem.
package fs_writer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Writer implements ports.FileWriter using the local filesystem.
type Writer struct{}

// New returns a new Writer.
func New() *Writer { return &Writer{} }

// safePath resolves rel inside projectPath and returns the absolute path,
// or an error if the result escapes the project root.
func safePath(projectPath, rel string) (string, error) {
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("fs_writer: path traversal blocked: %q escapes project root", rel)
	}
	absProject, err := filepath.Abs(projectPath)
	if err != nil {
		return "", fmt.Errorf("fs_writer: abs project path: %w", err)
	}
	joined := filepath.Join(absProject, rel)
	absResult, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("fs_writer: abs result path: %w", err)
	}
	if !within(absProject, absResult) {
		return "", fmt.Errorf("fs_writer: path traversal blocked: %q escapes project root", rel)
	}
	// The lexical check cannot see symlinks: "link/x" with link -> /etc would pass
	// it but write outside the project. Resolve the deepest existing ancestor of
	// the target and require the real location to stay inside the real root.
	realProject, err := filepath.EvalSymlinks(absProject)
	if err != nil {
		return "", fmt.Errorf("fs_writer: resolve project path: %w", err)
	}
	realExisting, err := evalDeepestExisting(absResult)
	if err != nil {
		return "", fmt.Errorf("fs_writer: resolve %q: %w", rel, err)
	}
	if !within(realProject, realExisting) {
		return "", fmt.Errorf("fs_writer: path traversal blocked: %q resolves outside the project root via a symlink", rel)
	}
	return absResult, nil
}

// within reports whether path equals root or lies beneath it.
func within(root, path string) bool {
	return path == root || strings.HasPrefix(path, root+string(filepath.Separator))
}

// evalDeepestExisting resolves symlinks in the longest existing prefix of path
// and re-appends the not-yet-existing remainder.
func evalDeepestExisting(path string) (string, error) {
	existing, rest := path, ""
	for {
		if _, err := os.Lstat(existing); err == nil {
			break
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return path, nil // nothing exists (cannot happen for an absolute path on a real FS)
		}
		rest = filepath.Join(filepath.Base(existing), rest)
		existing = parent
	}
	resolved, err := filepath.EvalSymlinks(existing)
	if err != nil {
		return "", err
	}
	return filepath.Join(resolved, rest), nil
}

// WriteCodeToFile writes the generated code to <projectPath>/<targetFile>,
// creating any missing parent directories automatically.
func (w *Writer) WriteCodeToFile(projectPath, targetFile, code string) error {
	fullPath, err := safePath(projectPath, targetFile)
	if err != nil {
		return err
	}
	// 0o640: owner read/write, group read; generated code may contain credentials
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o750); err != nil {
		return fmt.Errorf("fs_writer: mkdir: %w", err)
	}
	if err := os.WriteFile(fullPath, []byte(code), 0o640); err != nil {
		return fmt.Errorf("fs_writer: write file: %w", err)
	}
	return nil
}

// ReadContextFiles reads each file in <files> relative to <projectPath> and
// concatenates their content, separated by a header comment.
func (w *Writer) ReadContextFiles(projectPath string, files []string) (string, error) {
	var sb strings.Builder
	for _, f := range files {
		fullPath, err := safePath(projectPath, f)
		if err != nil {
			return "", err
		}
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return "", fmt.Errorf("fs_writer: read context file %q: %w", f, err)
		}
		fmt.Fprintf(&sb, "// --- %s ---\n", f)
		sb.Write(content)
		sb.WriteString("\n")
	}
	return sb.String(), nil
}
