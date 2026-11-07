package connect

// Tilde and SetRename expose internals to the external tests.
var Tilde = tilde

// SetRename replaces the file move and returns a function that restores it.
func SetRename(f func(oldpath, newpath string) error) (restore func()) {
	old := rename
	rename = f
	return func() { rename = old }
}
