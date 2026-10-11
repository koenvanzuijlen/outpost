package database

import (
	"database/sql"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// Store is the application's access to the database. Get one from Open.
type Store struct {
	db      *sql.DB
	queries *dbgen.Queries
}

// newStore creates a new Store from the given database connection.
func newStore(db *sql.DB) *Store {
	return &Store{
		db:      db,
		queries: dbgen.New(db),
	}
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}
