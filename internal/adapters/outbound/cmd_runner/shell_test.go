package cmd_runner

import "testing"

func TestShellFor_PicksTheShellOfTheOperatingSystem(t *testing.T) {
	for goos, want := range map[string][2]string{
		"windows": {"cmd.exe", "/c"}, "linux": {"sh", "-c"}, "darwin": {"sh", "-c"},
	} {
		if shell, flag := shellFor(goos); shell != want[0] || flag != want[1] {
			t.Errorf("%s: %s %s, want %s %s", goos, shell, flag, want[0], want[1])
		}
	}
}
