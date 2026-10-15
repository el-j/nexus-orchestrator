package repo_sqlite

import "database/sql"

// DB exposes the underlying handle to external (package repo_sqlite_test)
// tests that need to corrupt or age rows. It is compiled only into test binaries.
func (r *Repository) DB() *sql.DB { return r.db }
