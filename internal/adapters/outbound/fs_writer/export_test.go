package fs_writer

// SetAbsPath replaces how paths are made absolute and returns a function that restores it.
func SetAbsPath(f func(string) (string, error)) (restore func()) {
	old := absPath
	absPath = f
	return func() { absPath = old }
}
