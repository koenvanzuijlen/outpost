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
		t.Fatalf("database.Open() failed: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	server, err := New(logger, db, nil)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	return server
}
