package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"nexus-orchestrator/internal/core/domain"
)

// projectDir checks that projectPath is an absolute path of an existing directory.
func projectDir(projectPath string) error {
	if !filepath.IsAbs(projectPath) {
		return fmt.Errorf("policy: projectPath must be an absolute path, got %q", projectPath)
	}
	info, err := os.Stat(projectPath)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("policy: %q is not a directory", projectPath)
	}
	return nil
}

// GetProjectPolicy returns the project's .nexus/policy.json; the bool is false when the project has none.
func (o *OrchestratorService) GetProjectPolicy(_ context.Context, projectPath string) (domain.ProjectPolicy, bool, error) {
	if err := projectDir(projectPath); err != nil {
		return domain.ProjectPolicy{}, false, err
	}
	return LoadProjectPolicy(projectPath)
}

// SetProjectPolicy validates p and writes it as the project's .nexus/policy.json, replacing the file
// (a symbolic link at that place is replaced, never followed). An invalid policy is refused and the
// file stays as it was. The engine reads the file on every task, so the change applies to the next one.
func (o *OrchestratorService) SetProjectPolicy(_ context.Context, projectPath string, p domain.ProjectPolicy) error {
	if err := projectDir(projectPath); err != nil {
		return err
	}
	if err := p.Validate(); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(p, "", "  ") // a struct of strings, numbers and slices always encodes
	return writePolicyFile(filepath.Join(projectPath, filepath.FromSlash(PolicyFile)), append(data, '\n'))
}

// writePolicyFile puts data at dest, creating .nexus when needed. The file is written beside dest
// and moved into place, so a reader never sees half a policy and a symbolic link at dest is replaced,
// never followed.
func writePolicyFile(dest string, data []byte) error {
	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("policy: %w", err)
	}
	tmp := filepath.Join(dir, ".policy-"+uuid.NewString()+".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("policy: %w", err)
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("policy: %w", err)
	}
	return nil
}
