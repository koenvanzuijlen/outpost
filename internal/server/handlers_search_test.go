package server

import (
	"fmt"
	"testing"
)

func TestHandleSearchResults(t *testing.T) {
	server := newTestServer(t)
	first := loadIGDBGame(t, "games_search.json")
	server.igdbClient = &fakeIGDB{searchResults: loadIGDBGames(t, "games_search.json")}

	w := getPage(t, server, "/search?q=witcher")
	assertBodyContains(t, w, fmt.Sprintf(`href="/games/%d"`, first.ID))
}
