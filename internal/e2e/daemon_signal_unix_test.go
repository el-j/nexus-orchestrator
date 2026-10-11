//go:build e2e && !windows

package e2e_test

import (
	"os"
	"syscall"
)

func sigterm() os.Signal { return syscall.SIGTERM }
