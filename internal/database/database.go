// Package database provides functions to open and interact with an SQLite database.
package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Open creates and returns a connection to the SQLite database at the specified path. It also applies any pending migrations.
func Open(ctx context.Context, logger *slog.Logger, path string) (*sql.DB, error) {
	conn, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	err = conn.PingContext(ctx)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	err = migrate(ctx, conn, logger)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}
	return conn, nil
}

// migrate applies any pending migrations to the given database connection.
func migrate(ctx context.Context, db *sql.DB, logger *slog.Logger) error {
	subFs, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("sub filesystem: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, subFs)
	if err != nil {
		return fmt.Errorf("goose provider: %w", err)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	for _, r := range results {
		logger.Info("migration applied", "migration", r.Source.Path)
	}
	return nil
}

// dsn constructs the Data Source Name for the SQLite database at the specified path.
func dsn(path string) string {
	return fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_txlock=immediate", path)
}
