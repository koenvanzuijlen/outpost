package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
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
	dlcs          []igdb.Game
	dlcsErr       error
	dlcsCalls     int
}

func (c *fakeIGDB) SearchGames(_ context.Context, _ string) ([]igdb.Game, error) {
	return c.searchResults, c.searchErr
}

func (c *fakeIGDB) GameByID(_ context.Context, _ int) (igdb.Game, error) {
	return c.game, c.gameErr
}

func (c *fakeIGDB) DLCs(_ context.Context, _ int) ([]igdb.Game, error) {
	c.dlcsCalls++
	return c.dlcs, c.dlcsErr
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

// addTestGame adds The Witcher 3 (1942) to the library, owned on PC (6), and returns the fake IGDB client.
func addTestGame(t *testing.T, server *Server) *fakeIGDB {
	t.Helper()
	fake := &fakeIGDB{game: loadIGDBGames(t, "game_by_id.json")[0]}
	server.igdbClient = fake
	w := postForm(t, server, "/library", "igdb_id=1942&platform_id=6")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("add to library status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	return fake
}

// wishlistTestGame adds The Witcher 3 (1942) to the wishlist and returns the fake IGDB client.
func wishlistTestGame(t *testing.T, server *Server) *fakeIGDB {
	t.Helper()
	fake := &fakeIGDB{game: loadIGDBGames(t, "game_by_id.json")[0]}
	server.igdbClient = fake
	w := postForm(t, server, "/wishlist", "igdb_id=1942")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("wishlist status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	return fake
}

// getTestGamePage requests the game page for The Witcher 3 and asserts a 200 response.
func getTestGamePage(t *testing.T, server *Server) *httptest.ResponseRecorder {
	t.Helper()
	return getGamePage(t, server, 1942)
}

// getGamePage requests the game page for igdbID and asserts a 200 response.
func getGamePage(t *testing.T, server *Server, igdbID int64) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/games/"+strconv.FormatInt(igdbID, 10), nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	return w
}

// getTestLibraryGame loads The Witcher 3 from the default user's library.
func getTestLibraryGame(t *testing.T, server *Server) database.LibraryGame {
	t.Helper()
	libraryGame, err := server.store.GetLibraryGame(t.Context(), defaultUserID, 1942)
	if err != nil {
		t.Fatalf("GetLibraryGame() failed: %v", err)
	}
	return libraryGame
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

// assertBodyContains reports an error if the response body does not contain want.
func assertBodyContains(t *testing.T, w *httptest.ResponseRecorder, want string) {
	t.Helper()
	if !strings.Contains(w.Body.String(), want) {
		t.Errorf("body does not contain %q", want)
	}
}

// assertBodyNotContains reports an error if the response body contains unwanted.
func assertBodyNotContains(t *testing.T, w *httptest.ResponseRecorder, unwanted string) {
	t.Helper()
	if strings.Contains(w.Body.String(), unwanted) {
		t.Errorf("body unexpectedly contains %q", unwanted)
	}
}

// assertLocation reports an error if the Location header does not equal want.
func assertLocation(t *testing.T, w *httptest.ResponseRecorder, want string) {
	t.Helper()
	if got := w.Header().Get("Location"); got != want {
		t.Errorf("Location header = %q, want %q", got, want)
	}
}

// derefOrNil returns the value p points to, or nil, so failure messages show a value instead of an address.
func derefOrNil[T any](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}
