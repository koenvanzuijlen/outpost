package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/koenvanzuijlen/outpost/internal/database"
	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
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

// SearchGames returns the configured search results and error.
func (c *fakeIGDB) SearchGames(_ context.Context, _ string) ([]igdb.Game, error) {
	return c.searchResults, c.searchErr
}

// GameByID returns the configured game and error, whatever the ID.
func (c *fakeIGDB) GameByID(_ context.Context, _ int) (igdb.Game, error) {
	return c.game, c.gameErr
}

// DLCs counts the call and returns the configured DLC and error, whatever the ID.
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

// loadIGDBGame loads the first IGDB game from a fixture located in the testdata directory.
func loadIGDBGame(t *testing.T, name string) igdb.Game {
	t.Helper()
	games := loadIGDBGames(t, name)
	if len(games) == 0 {
		t.Fatalf("len(%s) = 0, want at least 1", name)
	}
	return games[0]
}

// newTestServer returns a Server with a fresh database in a temporary directory and no IGDB client. Tests that call
// IGDB set server.igdbClient to a fakeIGDB.
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

	return New(logger, store, nil)
}

// addTestGame adds The Witcher 3 (1942) to the library, owned on PC (6), and returns the fake IGDB client.
func addTestGame(t *testing.T, server *Server) *fakeIGDB {
	t.Helper()
	fake := &fakeIGDB{game: loadIGDBGames(t, "game_by_id.json")[0]}
	server.igdbClient = fake
	w := postForm(t, server, "/games/1942/library", "platform_id=6")
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
	w := postForm(t, server, "/games/1942/wishlist", "")
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
	return getPage(t, server, "/games/"+strconv.FormatInt(igdbID, 10))
}

// getPage requests path and asserts a 200 response.
func getPage(t *testing.T, server *Server, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	server.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, want %d", path, w.Code, http.StatusOK)
	}
	return w
}

// getTestGame returns The Witcher 3 (1942) from the library or wishlist. It fails the test if it isn't there.
func getTestGame(t *testing.T, server *Server) database.GameOverview {
	t.Helper()
	details, err := server.store.GetGame(t.Context(), defaultUserID, 1942)
	if err != nil {
		t.Fatalf("GetGame() failed: %v", err)
	}
	if details.UserGame == nil {
		t.Fatal("GetGame(1942) UserGame = nil, want an entry")
	}
	return details
}

// onlyPlaythrough returns the single playthrough of details. It fails the test if there isn't exactly one.
func onlyPlaythrough(t *testing.T, details database.GameOverview) dbgen.ListPlaythroughsForUserGameIDRow {
	t.Helper()
	if len(details.Playthroughs) != 1 {
		t.Fatalf("len(Playthroughs) = %d, want 1", len(details.Playthroughs))
	}
	return details.Playthroughs[0]
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

// collapseSpace replaces every run of whitespace in s with a single space, so assertions don't depend on template
// formatting.
func collapseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// assertBodyContains reports an error if the response body does not contain want, ignoring whitespace differences.
func assertBodyContains(t *testing.T, w *httptest.ResponseRecorder, want string) {
	t.Helper()
	if !strings.Contains(collapseSpace(w.Body.String()), collapseSpace(want)) {
		t.Errorf("body does not contain %q", want)
	}
}

// assertBodyNotContains reports an error if the response body contains unwanted, ignoring whitespace differences.
func assertBodyNotContains(t *testing.T, w *httptest.ResponseRecorder, unwanted string) {
	t.Helper()
	if strings.Contains(collapseSpace(w.Body.String()), collapseSpace(unwanted)) {
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

// testAddDLCPath adds Hearts of Stone (12503), a DLC of The Witcher 3, to the library separately.
const testAddDLCPath = "/games/12503/library"

// testDLCGame returns Hearts of Stone (12503) as IGDB serves it.
func testDLCGame(t *testing.T) igdb.Game {
	t.Helper()
	dlcs := loadIGDBGames(t, "dlcs.json")
	i := slices.IndexFunc(dlcs, func(g igdb.Game) bool { return g.ID == 12503 })
	if i == -1 {
		t.Fatal("dlcs.json has no game 12503")
	}
	return dlcs[i]
}

// postAddTestDLC posts testAddDLCPath while fake serves Hearts of Stone from IGDB, then restores the previous game.
func postAddTestDLC(t *testing.T, server *Server, fake *fakeIGDB) *httptest.ResponseRecorder {
	t.Helper()
	previous := fake.game
	fake.game = testDLCGame(t)
	defer func() { fake.game = previous }()
	return postForm(t, server, testAddDLCPath, "")
}

// addTestGameWithDLCs adds The Witcher 3 on PC and views its page so its DLCs are fetched and stored.
func addTestGameWithDLCs(t *testing.T, server *Server) *fakeIGDB {
	t.Helper()
	fake := addTestGame(t, server)
	fake.dlcs = loadIGDBGames(t, "dlcs.json")
	getTestGamePage(t, server)
	return fake
}

// startTestPlaythrough adds The Witcher 3 (1942) to the library on PC, starts a playthrough with the given name and
// platform (both may be empty), and returns its ID.
func startTestPlaythrough(t *testing.T, server *Server, name, platformIGDBID string) int64 {
	t.Helper()
	addTestGame(t, server)

	form := url.Values{"name": {name}, "platform": {platformIGDBID}}.Encode()
	w := postForm(t, server, "/games/1942/playthroughs", form)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("start playthrough status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	details := getTestGame(t, server)
	return onlyPlaythrough(t, details).Playthrough.ID
}

// formByAction returns the form of the response body whose action attribute equals action, with whitespace collapsed.
func formByAction(t *testing.T, w *httptest.ResponseRecorder, action string) string {
	t.Helper()
	body := collapseSpace(w.Body.String())
	marker := `action="` + action + `"`
	start := strings.Index(body, marker)
	if start < 0 {
		t.Fatalf("body has no form with %s", marker)
	}
	end := strings.Index(body[start:], "</form>")
	if end < 0 {
		t.Fatalf("form with %s is not closed", marker)
	}
	return body[start : start+end]
}
