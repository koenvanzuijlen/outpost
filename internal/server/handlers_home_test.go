package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestHandleHomeRendersLibraryAndWishlist(t *testing.T) {
	searchResponse := loadIGDBGames(t, "games_search.json")
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: searchResponse[0]}
	w := postForm(t, server, "/games/194662/wishlist", "")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("wishlist status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	server.igdbClient = &fakeIGDB{game: searchResponse[1]}
	w = postForm(t, server, "/games/1942/library", "platform_id=6")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("add to library status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w = getPage(t, server, "/")
	assertBodyContains(t, w, `href="/games/194662"`)
	assertBodyContains(t, w, `href="/games/1942"`)
}

func TestHandleHomeShowsPlayingGames(t *testing.T) {
	server := newTestServer(t)
	addTestGame(t, server)
	if w := postForm(t, server, "/games/1942/details", "status=playing"); w.Code != http.StatusSeeOther {
		t.Fatalf("update game status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "games_search.json")[0]}
	if w := postForm(t, server, "/games/194662/library", "platform_id=6"); w.Code != http.StatusSeeOther {
		t.Fatalf("add to library status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w := getPage(t, server, "/")
	body := collapseSpace(w.Body.String())
	_, rest, ok := strings.Cut(body, "<h2>Playing</h2>")
	if !ok {
		t.Fatalf("body missing playing section:\n%s", body)
	}
	playing, _, ok := strings.Cut(rest, "<h2>Backlog")
	if !ok {
		t.Fatalf("body missing backlog section:\n%s", body)
	}

	if !strings.Contains(playing, `href="/games/1942"`) {
		t.Errorf("playing section missing playing game 1942:\n%s", playing)
	}
	if strings.Contains(playing, `href="/games/194662"`) {
		t.Errorf("playing section contains backlog game 194662:\n%s", playing)
	}
}

func TestHandleHomeLinks(t *testing.T) {
	server := newTestServer(t)
	w := getPage(t, server, "/")

	assertBodyNotContains(t, w, `href="/library?status=playing&sort=started"`)
	assertBodyContains(t, w, `href="/library?status=backlog"`)
	assertBodyContains(t, w, `wishlist games</span>`)
}

func TestHandleHomeRendersUnownedDLC(t *testing.T) {
	server := newTestServer(t)
	fake := addTestGameWithDLCs(t, server)
	if w := postAddTestDLC(t, server, fake); w.Code != http.StatusSeeOther {
		t.Fatalf("add dlc to library status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w := getPage(t, server, "/")
	assertBodyContains(t, w, `href="/games/12503"`)
}
