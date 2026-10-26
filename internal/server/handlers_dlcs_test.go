package server

import (
	"net/http"
	"testing"
)

// Hearts of Stone (12503) is a DLC of The Witcher 3 available on PC (6).
const testDLCPath = "/games/1942/platforms/6/dlcs/12503"

// addTestGameWithDLCs adds The Witcher 3 on PC and views its page so its DLCs are fetched and stored.
func addTestGameWithDLCs(t *testing.T, server *Server) {
	t.Helper()
	fake := addTestGame(t, server)
	fake.dlcs = loadIGDBGames(t, "dlcs.json")
	getTestGamePage(t, server)
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
