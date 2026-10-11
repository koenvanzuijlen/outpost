package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/koenvanzuijlen/outpost/internal/database"
	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

// fakeIGDB is a mock implementation of the IGDB client used for testing.
type fakeIGDB struct {
	searchResults []igdb.Game
	searchErr     error
	game          igdb.Game
	gameErr       error
}

func (c *fakeIGDB) SearchGames(_ context.Context, _ string) ([]igdb.Game, error) {
	return c.searchResults, c.searchErr
}

func (c *fakeIGDB) GameByID(_ context.Context, _ int) (igdb.Game, error) {
	return c.game, c.gameErr
}

// loadIGDBGames loads a list of IGDB games from a fixture located in the testdata directory.
func loadIGDBGames(t *testing.T, name string) []igdb.Game {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("../igdb/testdata", name))
	if err != nil {
		t.Fatalf("os.ReadFile() failed: %v", err)
	}
	var games []igdb.Game
	if err := json.Unmarshal(data, &games); err != nil {
		t.Fatalf("json.Unmarshal() failed: %v", err)
	}
	return games
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	logger := slog.New(slog.DiscardHandler)

	path := filepath.Join(t.TempDir(), "test.db")
	store, err := database.Open(t.Context(), logger, path)
	if err != nil {
		t.Fatalf("database.Open() failed: %v", err)
	}

	t.Cleanup(func() {
		_ = store.Close()
	})

	server, err := New(logger, store, nil)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	return server
}

// postForm sends a POST request with the given form body to the specified URL and returns the response recorder.
func postForm(t *testing.T, server *Server, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	return w
}

// derefOrNil returns the value p points to, or nil, so failure messages show a value instead of an address.
func derefOrNil[T any](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}
