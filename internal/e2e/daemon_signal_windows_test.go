//go:build e2e && windows

package e2e_test

import "os"

func sigterm() os.Signal { return os.Kill }
