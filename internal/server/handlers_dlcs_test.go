package server

import (
	"net/http"
	"testing"
)

// testDLCPath is the ownership toggle for Hearts of Stone (12503) on The Witcher 3's PC copy (6).
const testDLCPath = "/games/1942/copies/6/dlcs/12503"

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
	assertBodyContains(t, w, "PC (Microsoft Windows) (1 DLC owned)")
}

func TestHandleSetDLCOwnedValidation(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
		want int
	}{
		{"invalid owned", testDLCPath, "owned=maybe", http.StatusBadRequest},
		{"non-numeric game id", "/games/abc/copies/6/dlcs/12503", "owned=true", http.StatusBadRequest},
		{"non-numeric platform id", "/games/1942/copies/abc/dlcs/12503", "owned=true", http.StatusBadRequest},
		{"non-numeric dlc id", "/games/1942/copies/6/dlcs/abc", "owned=true", http.StatusBadRequest},
		{"unknown dlc", "/games/1942/copies/6/dlcs/999999", "owned=true", http.StatusNotFound},
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
