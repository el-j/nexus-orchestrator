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
	if absResult != absProject && !strings.HasPrefix(absResult, absProject+string(filepath.Separator)) {
		return "", fmt.Errorf("fs_writer: path traversal blocked: %q escapes project root", rel)
	}
	return absResult, nil
}

// WriteCodeToFile writes the generated code to <projectPath>/<targetFile>,
// creating any missing parent directories automatically.
func (w *Writer) WriteCodeToFile(projectPath, targetFile, code string) error {
	fullPath, err := safePath(projectPath, targetFile)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return fmt.Errorf("fs_writer: mkdir: %w", err)
	}
	if err := os.WriteFile(fullPath, []byte(code), 0o644); err != nil {
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
		sb.WriteString(fmt.Sprintf("// --- %s ---\n", f))
		sb.Write(content)
		sb.WriteString("\n")
	}
	return sb.String(), nil
}
