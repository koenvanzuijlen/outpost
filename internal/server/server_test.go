package server

import (
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/koenvanzuijlen/outpost/internal/database"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	logger := slog.New(slog.DiscardHandler)

	path := filepath.Join(t.TempDir(), "test.db")
	db, err := database.Open(t.Context(), logger, path)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	server, err := New(logger, db)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	return server
}
