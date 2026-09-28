package database

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"
)

func TestDatabaseConnection(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(ctx, slog.New(slog.DiscardHandler), path)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	var fk int
	err = db.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fk)
	if err != nil {
		t.Fatalf("failed to check foreign keys: %v", err)
	}
	if fk != 1 {
		t.Fatalf("foreign keys not enabled")
	}
}
