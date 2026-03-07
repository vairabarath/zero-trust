package state

import (
	"database/sql"
)

// OpenSQLite opens a SQLite database at path and initialises the schema.
// Prefer Open() which also supports PostgreSQL via DATABASE_URL.
func OpenSQLite(path string) (*sql.DB, error) {
	return openSQLiteDB(path)
}
