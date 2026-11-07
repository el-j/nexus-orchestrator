package repo_sqlite

import (
	"context"
	"database/sql"
	"fmt"
)

// migrate brings the database to the current schema: it creates the tables, adds the columns later
// versions introduced and creates the knowledge tables. Every step can run again without harm.
func migrate(db *sql.DB) error {
	if err := runSteps(db, schemaBase); err != nil {
		return err
	}
	for _, col := range addedColumns {
		// An error here means the column exists already.
		_, _ = db.ExecContext(context.Background(), fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", col.table, col.name, col.def))
	}
	if err := runSteps(db, schemaMore); err != nil {
		return err
	}
	return migrateKnowledge(db)
}

// runSteps executes the steps in order and names the failing one in the error.
func runSteps(db *sql.DB, steps []schemaStep) error {
	for _, st := range steps {
		if _, err := db.ExecContext(context.Background(), st.sql); err != nil {
			return fmt.Errorf("sqlite: migrate %s: %w", st.name, err)
		}
	}
	return nil
}
