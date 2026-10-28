package server

import (
	"net/http"
	"testing"
)

// Hearts of Stone (12503) is a DLC of The Witcher 3 available on PC (6).
const (
	testDLCPath    = "/games/1942/platforms/6/dlcs/12503"
	testAddDLCPath = "/games/1942/dlcs/12503/library"
)

// addTestGameWithDLCs adds The Witcher 3 on PC and views its page so its DLCs are fetched and stored.
func addTestGameWithDLCs(t *testing.T, server *Server) *fakeIGDB {
	t.Helper()
	fake := addTestGame(t, server)
	fake.dlcs = loadIGDBGames(t, "dlcs.json")
	getTestGamePage(t, server)
	return fake
}

func TestHandleSetDLCOwnedStatus(t *testing.T) {
	server := newTestServer(t)
	addTestGameWithDLCs(t, server)

	w := postForm(t, server, testDLCPath, "owned=true")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	assertLocation(t, w, "/games/1942")

	w = getTestGamePage(t, server)
	assertBodyContains(t, w, `action="`+testDLCPath+`"`)
	assertBodyContains(t, w, `value="false" aria-pressed="true"`)
}

func TestHandleSetDLCOwnedValidation(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
		want int
	}{
		{"invalid owned", testDLCPath, "owned=maybe", http.StatusBadRequest},
		{"non-numeric game id", "/games/abc/platforms/6/dlcs/12503", "owned=true", http.StatusBadRequest},
		{"non-numeric platform id", "/games/1942/platforms/abc/dlcs/12503", "owned=true", http.StatusBadRequest},
		{"non-numeric dlc id", "/games/1942/platforms/6/dlcs/abc", "owned=true", http.StatusBadRequest},
		{"unknown dlc", "/games/1942/platforms/6/dlcs/999999", "owned=true", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			addTestGameWithDLCs(t, server)

			w := postForm(t, server, tt.path, tt.body)
			if w.Code != tt.want {
				t.Errorf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}

func TestHandleAddDLCToLibrary(t *testing.T) {
	server := newTestServer(t)
	fake := addTestGameWithDLCs(t, server)

	w := getTestGamePage(t, server)
	assertBodyContains(t, w, `action="`+testAddDLCPath+`"`)

	w = postForm(t, server, testAddDLCPath, "")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	assertLocation(t, w, "/games/12503")

	callsBefore := fake.dlcsCalls
	w = getGamePage(t, server, 12503)
	assertBodyContains(t, w, `DLC of <a href="/games/1942">`)
	assertBodyContains(t, w, "Not owned on any platform.")
	assertBodyContains(t, w, `Ownership of this DLC is managed on the page of <a href="/games/1942">`)
	if fake.dlcsCalls != callsBefore {
		t.Errorf("dlcsCalls = %d, want %d", fake.dlcsCalls, callsBefore)
	}

	w = getTestGamePage(t, server)
	assertBodyContains(t, w, `<a href="/games/12503">`)
	assertBodyContains(t, w, "</a> – backlog")
	assertBodyNotContains(t, w, `action="`+testAddDLCPath+`"`)
}

func TestHandleAddDLCToLibraryValidation(t *testing.T) {
	tests := []struct {
		name string
		path string
		want int
	}{
		{"non-numeric game id", "/games/abc/dlcs/12503/library", http.StatusBadRequest},
		{"non-numeric dlc id", "/games/1942/dlcs/abc/library", http.StatusBadRequest},
		{"unknown dlc", "/games/1942/dlcs/999999/library", http.StatusNotFound},
		{"game not in library", "/games/1020/dlcs/12503/library", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			addTestGameWithDLCs(t, server)

			w := postForm(t, server, tt.path, "")
			if w.Code != tt.want {
				t.Errorf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}

func TestHandleStartPlaythroughOnDLC(t *testing.T) {
	server := newTestServer(t)
	addTestGameWithDLCs(t, server)

	if w := postForm(t, server, testDLCPath, "owned=true"); w.Code != http.StatusSeeOther {
		t.Fatalf("own status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if w := postForm(t, server, testAddDLCPath, ""); w.Code != http.StatusSeeOther {
		t.Fatalf("track status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w := postForm(t, server, "/games/12503/playthroughs", "platform=6")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("start playthrough status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	assertLocation(t, w, "/games/12503")

	w = getGamePage(t, server, 12503)
	assertBodyContains(t, w, "Platform: PC (Microsoft Windows)")
	assertBodyContains(t, w, `Ownership of this DLC is managed on the page of <a href="/games/1942">`)
	assertBodyNotContains(t, w, "Not owned on any platform.")
}
