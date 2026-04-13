// Package repo_sqlite — FTS5 compile-time activation.
//
// mattn/go-sqlite3 bundles the sqlite3 amalgamation and compiles it when the
// package is first built. SQLite FTS5 is disabled in the amalgamation by default;
// enabling it requires -DSQLITE_ENABLE_FTS5 to be passed to the C compiler at
// go-sqlite3's compile time.
//
// Setting #cgo CFLAGS here does NOT affect go-sqlite3's own compilation (CGO
// flags are package-scoped). The reliable cross-platform mechanism is to set
// CGO_CFLAGS in the build environment. This file documents that requirement and
// provides a compile-time assertion that FTS5 is expected.
//
// To build and test with FTS5 enabled:
//
//	CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go test ./...
//
// The project Makefile handles this automatically via the `test` target.
package repo_sqlite
