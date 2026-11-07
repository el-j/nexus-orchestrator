package repo_sqlite

import (
	"database/sql"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// DB exposes the underlying handle to external (package repo_sqlite_test)
// tests that need to corrupt or age rows. It is compiled only into test binaries.
func (r *Repository) DB() *sql.DB { return r.db }

// MillisOf, NormalizeStatus and ParseSQLiteTime expose the column readers to the external tests.
func MillisOf(v any) (int64, error)              { return millisOf(v) }
func NormalizeStatus(s string) domain.TaskStatus { return normalizeStatus(s) }
func ParseSQLiteTime(s string) time.Time         { return parseSQLiteTime(s) }

// SetOpenDatabase replaces how New opens the handle and returns a function that restores it.
func SetOpenDatabase(f func(driver, dsn string) (*sql.DB, error)) (restore func()) {
	old := openDatabase
	openDatabase = f
	return func() { openDatabase = old }
}
