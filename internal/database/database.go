// Package database provides functions to open and interact with an SQLite database.
package database

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"

	"github.com/pressly/goose/v3"
	// Import the SQLite driver for its side-effects.
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Open opens the SQLite database at path, applies any pending migrations, and returns a Store for it.
func Open(ctx context.Context, logger *slog.Logger, path string) (*Store, error) {
	migrations, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("sub filesystem: %w", err)
	}

	conn, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	err = conn.PingContext(ctx)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	err = migrate(ctx, conn, migrations, logger, path)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}
	return newStore(conn), nil
}

// migrate applies the pending migrations from migrations to db. If any are pending and the database isn't new (version
// 0), it first backs up the database at path; see backup. A failed backup stops the migration.
func migrate(ctx context.Context, db *sql.DB, migrations fs.FS, logger *slog.Logger, path string) error {
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations)
	if err != nil {
		return fmt.Errorf("goose provider: %w", err)
	}

	dbVersion, err := provider.GetDBVersion(ctx)
	if err != nil {
		return fmt.Errorf("get db version: %w", err)
	}
	hasPending, err := provider.HasPending(ctx)
	if err != nil {
		return fmt.Errorf("check pending migrations: %w", err)
	}

	if hasPending && dbVersion != 0 {
		err = backup(ctx, db, logger, path, dbVersion)
		if err != nil {
			return fmt.Errorf("backup database: %w", err)
		}
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

// backup writes a consistent snapshot of db (including the WAL) to <path>.v<version>.bak with VACUUM INTO. If that
// file already exists it is kept: after a failed migration the database is still at the same version, and the existing
// backup is from before the first attempt.
func backup(ctx context.Context, db *sql.DB, logger *slog.Logger, path string, version int64) error {
	backupPath := fmt.Sprintf("%s.v%d.bak", path, version)

	_, err := os.Stat(backupPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("stat backup file: %w", err)
	}
	if err == nil {
		logger.Info("backup file already exists, reusing", "path", backupPath)
		return nil
	}

	_, err = db.ExecContext(ctx, "VACUUM INTO ?", backupPath)
	if err != nil {
		return fmt.Errorf("vacuum into backup: %w", err)
	}

	logger.Info("database backed up before migration", "path", backupPath, "version", version)

	return nil
}

// dsn constructs the Data Source Name for the SQLite database at the specified path.
func dsn(path string) string {
	return fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_txlock=immediate", path)
}
