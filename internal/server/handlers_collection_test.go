package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	testRemovePCPath  = "/games/1942/platforms/6/delete"
	testRemovePS4Path = "/games/1942/platforms/48/delete"
)

// addTestGamePS4 adds The Witcher 3 on PC and PS4 (48).
func addTestGamePS4(t *testing.T, server *Server) {
	t.Helper()
	addTestGame(t, server)
	if w := postForm(t, server, "/library", "igdb_id=1942&platform_id=48"); w.Code != http.StatusSeeOther {
		t.Fatalf("add to library status = %d, want %d", w.Code, http.StatusSeeOther)
	}
}

func TestHandleRemoveFromCollection(t *testing.T) {
	server := newTestServer(t)
	addTestGamePS4(t, server)

	w := getTestGamePage(t, server)
	assertBodyContains(t, w, `action="`+testRemovePCPath+`"`)

	w = postForm(t, server, testRemovePCPath, "")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	assertLocation(t, w, "/games/1942")

	w = getTestGamePage(t, server)
	assertBodyNotContains(t, w, `action="`+testRemovePCPath+`"`)
	assertBodyContains(t, w, `action="`+testRemovePS4Path+`"`)
}

func TestHandleRemoveOnlyCopy(t *testing.T) {
	server := newTestServer(t)
	addTestGame(t, server)

	if w := postForm(t, server, testRemovePCPath, ""); w.Code != http.StatusSeeOther {
		t.Fatalf("remove status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w := httptest.NewRecorder()
	server.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("library status = %d, want %d", w.Code, http.StatusOK)
	}
	assertBodyContains(t, w, `<a href="/games/1942">`)
	assertBodyContains(t, w, "not owned")

	w = getTestGamePage(t, server)
	assertBodyContains(t, w, "Not owned on any platform.")
}

func TestHandleRemoveFromCollectionTwice(t *testing.T) {
	server := newTestServer(t)
	addTestGamePS4(t, server)

	for i := range 2 {
		w := postForm(t, server, testRemovePCPath, "")
		if w.Code != http.StatusSeeOther {
			t.Errorf("call %d status = %d, want %d", i+1, w.Code, http.StatusSeeOther)
		}
	}
}

func TestHandleRemoveFromCollectionValidation(t *testing.T) {
	tests := []struct {
		name string
		path string
		want int
	}{
		{"non-numeric game id", "/games/abc/platforms/6/delete", http.StatusBadRequest},
		{"non-numeric platform id", "/games/1942/platforms/abc/delete", http.StatusBadRequest},
		{"game not in library", "/games/1020/platforms/6/delete", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			addTestGame(t, server)

			w := postForm(t, server, tt.path, "")
			if w.Code != tt.want {
				t.Errorf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}
