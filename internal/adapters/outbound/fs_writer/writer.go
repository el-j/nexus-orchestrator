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

// WriteCodeToFile writes the generated code to <projectPath>/<targetFile>,
// creating any missing parent directories automatically.
func (w *Writer) WriteCodeToFile(projectPath, targetFile, code string) error {
	fullPath := filepath.Join(projectPath, targetFile)
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
		fullPath := filepath.Join(projectPath, f)
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
