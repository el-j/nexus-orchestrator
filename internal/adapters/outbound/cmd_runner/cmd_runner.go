package cmd_runner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"nexus-orchestrator/internal/core/ports"
)

var _ ports.CommandRunner = (*Runner)(nil)

// Runner executes verification commands within specified directories.
type Runner struct {
	defaultTimeout time.Duration
}

// Option configures Runner.
type Option func(*Runner)

// WithDefaultTimeout sets the default execution timeout when context deadline is not set.
func WithDefaultTimeout(d time.Duration) Option {
	return func(r *Runner) {
		if d > 0 {
			r.defaultTimeout = d
		}
	}
}

// New creates a new Runner instance.
func New(opts ...Option) *Runner {
	r := &Runner{
		defaultTimeout: 2 * time.Minute,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Run executes the command in the specified directory.
// It returns the combined stdout and stderr output, and an error if the process exited with non-zero status.
func (r *Runner) Run(ctx context.Context, dir string, command string) (string, error) {
	if strings.TrimSpace(command) == "" {
		return "", nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok && r.defaultTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.defaultTimeout)
		defer cancel()
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd.exe", "/c", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	if dir != "" {
		cmd.Dir = dir
	}

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	output := strings.TrimSpace(buf.String())
	if err != nil {
		return output, fmt.Errorf("cmd_runner: execute command %q: %w", command, err)
	}
	return output, nil
}
