package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/koenvanzuijlen/outpost/internal/igdb"
)

func TestHandleGameValidation(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"zero id", "/games/0"},
		{"negative id", "/games/-1"},
		{"string id", "/games/abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestHandleGameFromIGDB(t *testing.T) {
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: loadIGDBGame(t, "game_by_id.json")}

	w := getTestGamePage(t, server)
	assertBodyContains(t, w, "The Witcher 3: Wild Hunt")
	assertBodyContains(t, w, "Add to library")
	assertBodyNotContains(t, w, `name="status"`)
	assertBodyContains(t, w, "Released on: Nintendo Switch, Nintendo Switch 2, PC (Microsoft Windows), PlayStation 4, PlayStation 5, Xbox One, Xbox Series X|S")
	assertBodyContains(t, w, `<time datetime="2015-05-19">2015</time>`)
}

func TestHandleGameWithoutReleaseDate(t *testing.T) {
	server := newTestServer(t)
	game := loadIGDBGame(t, "game_by_id.json")
	game.FirstReleaseDate = 0
	server.igdbClient = &fakeIGDB{game: game}

	w := getTestGamePage(t, server)
	assertBodyContains(t, w, "The Witcher 3: Wild Hunt")
	assertBodyNotContains(t, w, "<time")
}

func TestHandleGameCachedIGDBError(t *testing.T) {
	server := newTestServer(t)
	fake := &fakeIGDB{game: loadIGDBGame(t, "game_by_id.json")}
	server.igdbClient = fake
	getTestGamePage(t, server)

	fake.gameErr = errors.New("down")
	w := getTestGamePage(t, server)
	assertBodyContains(t, w, "The Witcher 3: Wild Hunt")
}

func TestHandleGameUncachedIGDBError(t *testing.T) {
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{gameErr: errors.New("down")}

	w := httptest.NewRecorder()
	server.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/games/1942", nil))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleGameDLCParentInLibrary(t *testing.T) {
	server := newTestServer(t)
	fake := addTestGame(t, server)
	fake.game = loadIGDBGame(t, "game_by_id_dlc.json")

	w := getGamePage(t, server, 13166)
	assertBodyContains(t, w, `action="/games/13166/library"`)
	assertBodyContains(t, w, "Add to library")
}

func TestHandleGameDLCParentNotInLibrary(t *testing.T) {
	server := newTestServer(t)
	fake := &fakeIGDB{game: loadIGDBGame(t, "game_by_id.json")}
	server.igdbClient = fake
	getTestGamePage(t, server)
	fake.game = loadIGDBGame(t, "game_by_id_dlc.json")

	w := getGamePage(t, server, 13166)
	assertBodyContains(t, w, "The Witcher 3: Wild Hunt - Blood and Wine")
	assertBodyNotContains(t, w, `action="/games/13166/library"`)
	assertBodyNotContains(t, w, "Add to library")
}

func TestHandleGameNotInIGDB(t *testing.T) {
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{gameErr: igdb.ErrGameNotFound}

	req := httptest.NewRequest(http.MethodGet, "/games/999999", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleGameInLibrary(t *testing.T) {
	server := newTestServer(t)
	addTestGame(t, server)

	w := postForm(t, server, "/games/1942/details", "status=completed&rating=9")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("update game status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w = getTestGamePage(t, server)
	assertBodyContains(t, w, "The Witcher 3: Wild Hunt")
	assertBodyContains(t, w, "PC (Microsoft Windows)")
	assertBodyContains(t, w, `name="status"`)
	assertBodyContains(t, w, "4.5")
}

func TestHandleGameInLibraryWishlisted(t *testing.T) {
	server := newTestServer(t)
	wishlistTestGame(t, server)

	w := getTestGamePage(t, server)
	assertBodyContains(t, w, "The Witcher 3: Wild Hunt")
	assertBodyNotContains(t, w, `name="status"`)
	assertBodyNotContains(t, w, "/playthroughs")
	assertBodyContains(t, w, `action="/games/1942/library"`)
	assertBodyNotContains(t, w, `action="/games/1942/wishlist"`)
	assertBodyContains(t, w, "On your wishlist.")
}

func TestHandleGameNotInLibraryForms(t *testing.T) {
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: loadIGDBGame(t, "game_by_id.json")}

	w := getTestGamePage(t, server)
	assertBodyContains(t, w, `action="/games/1942/library"`)
	assertBodyContains(t, w, `action="/games/1942/wishlist"`)
}

func TestHandleAddWishlistedGameToLibrary(t *testing.T) {
	server := newTestServer(t)
	wishlistTestGame(t, server)

	w := postForm(t, server, "/games/1942/library", "platform_id=6")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("add to library status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	assertLocation(t, w, "/games/1942")

	userGame := getTestGame(t, server).UserGame
	if userGame.Wishlisted {
		t.Error("Wishlisted = true, want false")
	}
	if userGame.Status == nil {
		t.Fatal("Status = nil, want backlog")
	}
	if *userGame.Status != "backlog" {
		t.Errorf("Status = %q, want %q", *userGame.Status, "backlog")
	}
}

func TestHandleGameActivePlaythrough(t *testing.T) {
	server := newTestServer(t)
	startTestPlaythrough(t, server, "Hard mode run", "6")

	details := getTestGame(t, server)
	playthrough := onlyPlaythrough(t, details).Playthrough
	if playthrough.StartedAt == nil {
		t.Fatal("StartedAt = nil, want a date")
	}
	playthroughID := playthrough.ID

	w := getTestGamePage(t, server)
	assertBodyContains(t, w, "Hard mode run")
	assertBodyContains(t, w, "Outcome: in progress")
	assertBodyContains(t, w, `aria-pressed="true">in progress</button>`)
	assertBodyContains(t, w, "Platform: PC (Microsoft Windows)")
	assertBodyContains(t, w, fmt.Sprintf("/games/1942/playthroughs/%d/outcome", playthroughID))
	assertBodyContains(t, w, fmt.Sprintf("/games/1942/playthroughs/%d/delete", playthroughID))
	assertBodyContains(t, w, fmt.Sprintf(`action="/games/1942/playthroughs/%d"`, playthroughID))
	assertBodyContains(t, w, `<option value="6" selected>PC (Microsoft Windows)</option>`)
	assertBodyContains(t, w, fmt.Sprintf(`value="%s"`, playthrough.StartedAt.Format("2006-01-02")))
	assertBodyNotContains(t, w, `name="finished_at"`)
}

func TestHandleGameCompletedPlaythrough(t *testing.T) {
	server := newTestServer(t)
	playthroughID := startTestPlaythrough(t, server, "Hard mode run", "6")

	w := postForm(t, server, fmt.Sprintf("/games/1942/playthroughs/%d/outcome", playthroughID), "outcome=abandoned")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("set outcome status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w = getTestGamePage(t, server)
	assertBodyContains(t, w, "Hard mode run")
	assertBodyContains(t, w, "Platform: PC (Microsoft Windows)")
	assertBodyContains(t, w, "Outcome: abandoned")
	assertBodyContains(t, w, `aria-pressed="true">abandoned</button>`)
	assertBodyContains(t, w, `aria-pressed="false">in progress</button>`)
	assertBodyContains(t, w, fmt.Sprintf("/games/1942/playthroughs/%d/outcome", playthroughID))
	assertBodyContains(t, w, fmt.Sprintf("/games/1942/playthroughs/%d/delete", playthroughID))
	assertBodyContains(t, w, `name="finished_at"`)
}

func TestHandleGameDLCs(t *testing.T) {
	server := newTestServer(t)
	fake := addTestGame(t, server)
	fake.dlcs = loadIGDBGames(t, "dlcs.json")

	w := getTestGamePage(t, server)
	assertBodyContains(t, w, `<span class="game-card__title">Hearts of Stone</span>`)
	assertBodyNotContains(t, w, "couldn't be loaded")
	if fake.dlcsCalls != 1 {
		t.Errorf("dlcsCalls after first view = %d, want %d", fake.dlcsCalls, 1)
	}

	w = getTestGamePage(t, server)
	assertBodyContains(t, w, "Hearts of Stone")
	if fake.dlcsCalls != 1 {
		t.Errorf("dlcsCalls after second view = %d, want %d", fake.dlcsCalls, 1)
	}
}

func TestHandleGameCopyDLCs(t *testing.T) {
	server := newTestServer(t)
	fake := addTestGame(t, server)
	fake.dlcs = loadIGDBGames(t, "dlcs.json")

	w := getTestGamePage(t, server)
	// Hearts of Stone (12503) is available on PC (6), the platform of the test copy.
	assertBodyContains(t, w, `</span> Hearts of Stone <span class="game-copy__year">2015</span></button>`)
	assertBodyNotContains(t, w, "DLC owned")
}

func TestHandleGameDLCsNotInLibrary(t *testing.T) {
	server := newTestServer(t)
	fake := &fakeIGDB{game: loadIGDBGame(t, "game_by_id.json")}
	fake.dlcs = loadIGDBGames(t, "dlcs.json")
	server.igdbClient = fake

	w := getTestGamePage(t, server)
	assertBodyContains(t, w, "<h2>DLC</h2>")
	assertBodyContains(t, w, `<a href="/games/12503" class="game-card">`)
	assertBodyContains(t, w, `<span class="game-card__title">Hearts of Stone</span>`)
}

func TestHandleGameDLCsIncludeCopyDLC(t *testing.T) {
	server := newTestServer(t)
	addTestGameWithDLCs(t, server)

	w := getTestGamePage(t, server)
	if got := strings.Count(w.Body.String(), `class="game-card"`); got != 8 {
		t.Errorf("game cards = %d, want 8", got)
	}
}

func TestHandleGameDLCsError(t *testing.T) {
	server := newTestServer(t)
	fake := addTestGame(t, server)
	fake.dlcsErr = errors.New("igdb unavailable")

	w := getTestGamePage(t, server)
	assertBodyContains(t, w, "couldn't be loaded")
	if fake.dlcsCalls != 1 {
		t.Errorf("dlcsCalls after first view = %d, want %d", fake.dlcsCalls, 1)
	}

	getTestGamePage(t, server)
	if fake.dlcsCalls != 2 {
		t.Errorf("dlcsCalls after second view = %d, want %d", fake.dlcsCalls, 2)
	}
}

func TestHandleGameNoDLCs(t *testing.T) {
	server := newTestServer(t)
	fake := addTestGame(t, server)

	w := getTestGamePage(t, server)
	assertBodyNotContains(t, w, "<h2>DLC</h2>")
	if fake.dlcsCalls != 1 {
		t.Errorf("dlcsCalls after first view = %d, want %d", fake.dlcsCalls, 1)
	}

	getTestGamePage(t, server)
	if fake.dlcsCalls != 1 {
		t.Errorf("dlcsCalls after second view = %d, want %d", fake.dlcsCalls, 1)
	}
}

func TestHandleUpdateGameValidation(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
	}{
		{"bad id", "/games/abc/details", "status=backlog&rating=10"},
		{"no status", "/games/1942/details", "rating=10"},
		{"invalid status", "/games/1942/details", "status=invalid&rating=10"},
		{"low rating", "/games/1942/details", "status=backlog&rating=0"},
		{"high rating", "/games/1942/details", "status=backlog&rating=11"},
		{"invalid rating", "/games/1942/details", "status=backlog&rating=abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			w := postForm(t, server, tt.path, tt.body)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestHandleUpdateGame(t *testing.T) {
	server := newTestServer(t)
	addTestGame(t, server)

	w := postForm(t, server, "/games/1942/details", "status=completed&rating=9")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("update game status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	assertLocation(t, w, "/games/1942")
	details := getTestGame(t, server)
	if details.UserGame.Status == nil || *details.UserGame.Status != "completed" {
		t.Errorf("status = %v, want %q", derefOrNil(details.UserGame.Status), "completed")
	}
	if details.UserGame.Rating == nil || *details.UserGame.Rating != 9 {
		t.Errorf("rating = %v, want %v", derefOrNil(details.UserGame.Rating), 9)
	}
}

func TestHandleUpdateGameNoRating(t *testing.T) {
	server := newTestServer(t)
	addTestGame(t, server)

	w := postForm(t, server, "/games/1942/details", "status=completed&rating=")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("update game status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	assertLocation(t, w, "/games/1942")
	details := getTestGame(t, server)
	if details.UserGame.Status == nil || *details.UserGame.Status != "completed" {
		t.Errorf("status = %v, want %q", derefOrNil(details.UserGame.Status), "completed")
	}
	if details.UserGame.Rating != nil {
		t.Errorf("rating = %v, want %v", derefOrNil(details.UserGame.Rating), nil)
	}
}

func TestHandleUpdateGameNotes(t *testing.T) {
	tests := []struct {
		name  string
		notes string
		want  *string
	}{
		{"trimmed", "  Buy the GOTY edition  ", new("Buy the GOTY edition")},
		{"whitespace only", "   ", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			addTestGame(t, server)

			body := url.Values{"status": {"completed"}, "notes": {tt.notes}}.Encode()
			w := postForm(t, server, "/games/1942/details", body)
			if w.Code != http.StatusSeeOther {
				t.Fatalf("update game status = %d, want %d", w.Code, http.StatusSeeOther)
			}
			details := getTestGame(t, server)
			got := details.UserGame.Notes
			if (got == nil) != (tt.want == nil) || (got != nil && *got != *tt.want) {
				t.Errorf("notes = %v, want %v", derefOrNil(got), derefOrNil(tt.want))
			}
		})
	}
}

func TestHandleUpdateGameNotesLength(t *testing.T) {
	tests := []struct {
		name  string
		notes string
		want  int
	}{
		{"over limit", strings.Repeat("a", maxNotesLength+1), http.StatusBadRequest},
		{"at limit", strings.Repeat("a", maxNotesLength), http.StatusSeeOther},
		{"multibyte at limit", strings.Repeat("é", maxNotesLength), http.StatusSeeOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			addTestGame(t, server)

			body := url.Values{"status": {"completed"}, "notes": {tt.notes}}.Encode()
			w := postForm(t, server, "/games/1942/details", body)
			if w.Code != tt.want {
				t.Errorf("update game status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}

func TestHandleGameNotesPrefill(t *testing.T) {
	server := newTestServer(t)
	addTestGame(t, server)

	body := url.Values{"status": {"completed"}, "notes": {"<b>hi</b>"}}.Encode()
	w := postForm(t, server, "/games/1942/details", body)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("update game status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w = getTestGamePage(t, server)
	assertBodyContains(t, w, "&lt;b&gt;hi&lt;/b&gt;</textarea>")
}

func TestHandleUpdateGameWishlisted(t *testing.T) {
	server := newTestServer(t)
	wishlistTestGame(t, server)

	w := postForm(t, server, "/games/1942/details", "status=completed&rating=9")
	if w.Code != http.StatusNotFound {
		t.Fatalf("update game status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleGamePlaythroughEditFormSelectedPlatform(t *testing.T) {
	server := newTestServer(t)
	addTestGamePS4(t, server)
	if w := postForm(t, server, "/games/1942/playthroughs", "platform=48"); w.Code != http.StatusSeeOther {
		t.Fatalf("start playthrough status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	playthroughID := onlyPlaythrough(t, getTestGame(t, server)).Playthrough.ID

	form := formByAction(t, getTestGamePage(t, server), fmt.Sprintf("/games/1942/playthroughs/%d", playthroughID))

	if got := strings.Count(form, "selected"); got != 1 {
		t.Errorf("selected options = %d, want 1 in %q", got, form)
	}
	if !strings.Contains(form, `<option value="48" selected>`) {
		t.Errorf("PS4 option is not selected in %q", form)
	}
	if strings.Contains(form, `<option value="6" selected>`) || strings.Contains(form, `<option value="" selected>`) {
		t.Errorf("a different option is selected in %q", form)
	}
}

func TestHandleGamePlaythroughEditFormNoPlatform(t *testing.T) {
	server := newTestServer(t)
	playthroughID := startTestPlaythrough(t, server, "", "")

	form := formByAction(t, getTestGamePage(t, server), fmt.Sprintf("/games/1942/playthroughs/%d", playthroughID))

	if got := strings.Count(form, "selected"); got != 1 {
		t.Errorf("selected options = %d, want 1 in %q", got, form)
	}
	if !strings.Contains(form, `<option value="" selected>No platform</option>`) {
		t.Errorf("No platform option is not selected in %q", form)
	}
}

func TestHandleGamePlaythroughOutcomeAriaPressed(t *testing.T) {
	server := newTestServer(t)
	playthroughID := startTestPlaythrough(t, server, "", "")
	action := fmt.Sprintf("/games/1942/playthroughs/%d/outcome", playthroughID)

	form := formByAction(t, getTestGamePage(t, server), action)
	if got := strings.Count(form, `aria-pressed="true"`); got != 1 {
		t.Errorf("pressed buttons = %d, want 1 in %q", got, form)
	}
	if !strings.Contains(form, `aria-pressed="true">in progress</button>`) {
		t.Errorf("in progress is not pressed in %q", form)
	}

	if w := postForm(t, server, action, "outcome=abandoned"); w.Code != http.StatusSeeOther {
		t.Fatalf("set outcome status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	form = formByAction(t, getTestGamePage(t, server), action)
	if got := strings.Count(form, `aria-pressed="true"`); got != 1 {
		t.Errorf("pressed buttons = %d, want 1 in %q", got, form)
	}
	if !strings.Contains(form, `aria-pressed="true">abandoned</button>`) {
		t.Errorf("abandoned is not pressed in %q", form)
	}
	if !strings.Contains(form, `aria-pressed="false">in progress</button>`) {
		t.Errorf("in progress is still pressed in %q", form)
	}
}
