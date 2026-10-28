package server

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestHandleStartPlaythrough(t *testing.T) {
	server := newTestServer(t)
	addTestGame(t, server)

	w := postForm(t, server, "/games/1942/playthroughs", "platform=6&name=First+run")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("start playthrough status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	assertLocation(t, w, "/games/1942")

	libraryGame := getTestUserGame(t, server)
	if len(libraryGame.Playthroughs) != 1 {
		t.Fatalf("len(Playthroughs) = %d, want %d", len(libraryGame.Playthroughs), 1)
	}
	playthrough := libraryGame.Playthroughs[0]
	if playthrough.Playthrough.Name == nil || *playthrough.Playthrough.Name != "First run" {
		t.Errorf("name = %v, want %q", derefOrNil(playthrough.Playthrough.Name), "First run")
	}
	if playthrough.PlatformName == nil || *playthrough.PlatformName != "PC (Microsoft Windows)" {
		t.Errorf("platform name = %v, want %q", derefOrNil(playthrough.PlatformName), "PC (Microsoft Windows)")
	}
	if libraryGame.UserGame.Status == nil || *libraryGame.UserGame.Status != "playing" {
		t.Errorf("status = %v, want %q", derefOrNil(libraryGame.UserGame.Status), "playing")
	}
}

func TestHandleStartPlaythroughBlankName(t *testing.T) {
	server := newTestServer(t)
	addTestGame(t, server)

	w := postForm(t, server, "/games/1942/playthroughs", "name=++")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("start playthrough status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	libraryGame := getTestUserGame(t, server)
	if len(libraryGame.Playthroughs) != 1 {
		t.Fatalf("len(Playthroughs) = %d, want %d", len(libraryGame.Playthroughs), 1)
	}
	if name := libraryGame.Playthroughs[0].Playthrough.Name; name != nil {
		t.Errorf("name = %v, want %v", derefOrNil(name), nil)
	}
}

func TestHandleStartPlaythroughValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"non-numeric platform", "platform=abc"},
		{"unknown platform", "platform=48"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			addTestGame(t, server)

			w := postForm(t, server, "/games/1942/playthroughs", tt.body)
			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestHandleStartPlaythroughNotInLibrary(t *testing.T) {
	server := newTestServer(t)
	w := postForm(t, server, "/games/1942/playthroughs", "")
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleStartPlaythroughWishlisted(t *testing.T) {
	server := newTestServer(t)
	wishlistTestGame(t, server)

	w := postForm(t, server, "/games/1942/playthroughs", "")
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// startTestPlaythrough adds game 1942 to the collection, starts a playthrough for it and returns the playthrough ID.
func startTestPlaythrough(t *testing.T, server *Server, name, platformIGDBID string) int64 {
	t.Helper()
	addTestGame(t, server)

	form := url.Values{"name": {name}, "platform": {platformIGDBID}}.Encode()
	w := postForm(t, server, "/games/1942/playthroughs", form)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("start playthrough status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	libraryGame := getTestUserGame(t, server)
	if len(libraryGame.Playthroughs) != 1 {
		t.Fatalf("len(Playthroughs) = %d, want %d", len(libraryGame.Playthroughs), 1)
	}
	return libraryGame.Playthroughs[0].Playthrough.ID
}

func TestHandleSetPlaythroughOutcome(t *testing.T) {
	tests := []string{"completed", "abandoned", "retired"}

	for _, outcome := range tests {
		t.Run(outcome, func(t *testing.T) {
			server := newTestServer(t)
			playthroughID := startTestPlaythrough(t, server, "", "")

			path := "/games/1942/playthroughs/" + strconv.FormatInt(playthroughID, 10) + "/outcome"
			w := postForm(t, server, path, "outcome="+outcome)
			if w.Code != http.StatusSeeOther {
				t.Fatalf("set outcome status = %d, want %d", w.Code, http.StatusSeeOther)
			}
			assertLocation(t, w, "/games/1942")

			libraryGame := getTestUserGame(t, server)
			if libraryGame.UserGame.Status == nil || *libraryGame.UserGame.Status != outcome {
				t.Errorf("status = %v, want %q", derefOrNil(libraryGame.UserGame.Status), outcome)
			}
			if got := libraryGame.Playthroughs[0].Playthrough.Outcome; got != outcome {
				t.Errorf("outcome = %q, want %q", got, outcome)
			}
		})
	}
}

func TestHandleSetPlaythroughOutcomeBackToPlaying(t *testing.T) {
	server := newTestServer(t)
	playthroughID := startTestPlaythrough(t, server, "", "")

	path := "/games/1942/playthroughs/" + strconv.FormatInt(playthroughID, 10) + "/outcome"
	for _, outcome := range []string{"completed", "playing"} {
		w := postForm(t, server, path, "outcome="+outcome)
		if w.Code != http.StatusSeeOther {
			t.Fatalf("set outcome %q status = %d, want %d", outcome, w.Code, http.StatusSeeOther)
		}
	}

	libraryGame := getTestUserGame(t, server)
	if len(libraryGame.Playthroughs) != 1 {
		t.Fatalf("len(Playthroughs) = %d, want %d", len(libraryGame.Playthroughs), 1)
	}
	if got := libraryGame.Playthroughs[0].Playthrough.Outcome; got != "playing" {
		t.Errorf("outcome = %q, want %q", got, "playing")
	}
}

func TestHandleSetPlaythroughOutcomeValidation(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
	}{
		{"invalid outcome", "/games/1942/playthroughs/{id}/outcome", "outcome=invalid"},
		{"non-numeric playthrough id", "/games/1942/playthroughs/abc/outcome", "outcome=completed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			playthroughID := startTestPlaythrough(t, server, "", "")

			path := strings.ReplaceAll(tt.path, "{id}", strconv.FormatInt(playthroughID, 10))
			w := postForm(t, server, path, tt.body)
			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestHandleSetPlaythroughOutcomeUnknownPlaythrough(t *testing.T) {
	server := newTestServer(t)
	playthroughID := startTestPlaythrough(t, server, "", "")

	path := "/games/1942/playthroughs/" + strconv.FormatInt(playthroughID+1, 10) + "/outcome"
	w := postForm(t, server, path, "outcome=completed")
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleUpdatePlaythrough(t *testing.T) {
	server := newTestServer(t)
	playthroughID := startTestPlaythrough(t, server, "", "")

	// Finishing a playthrough requires an outcome other than playing.
	path := "/games/1942/playthroughs/" + strconv.FormatInt(playthroughID, 10)
	w := postForm(t, server, path+"/outcome", "outcome=completed")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("set outcome status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	form := url.Values{
		"platform":    {"6"},
		"name":        {"  Second run  "},
		"started_at":  {"2026-01-02"},
		"finished_at": {"2026-03-04"},
	}.Encode()
	w = postForm(t, server, path, form)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("update status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	assertLocation(t, w, "/games/1942")

	libraryGame := getTestUserGame(t, server)
	if len(libraryGame.Playthroughs) != 1 {
		t.Fatalf("len(Playthroughs) = %d, want %d", len(libraryGame.Playthroughs), 1)
	}
	playthrough := libraryGame.Playthroughs[0]
	if playthrough.PlatformName == nil || *playthrough.PlatformName != "PC (Microsoft Windows)" {
		t.Errorf("platform name = %v, want %q", derefOrNil(playthrough.PlatformName), "PC (Microsoft Windows)")
	}
	if playthrough.Playthrough.Name == nil || *playthrough.Playthrough.Name != "Second run" {
		t.Errorf("name = %v, want %q", derefOrNil(playthrough.Playthrough.Name), "Second run")
	}
	wantStarted := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	if got := playthrough.Playthrough.StartedAt; got == nil || !got.Equal(wantStarted) {
		t.Errorf("started at = %v, want %v", derefOrNil(got), wantStarted)
	}
	wantFinished := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)
	if got := playthrough.Playthrough.FinishedAt; got == nil || !got.Equal(wantFinished) {
		t.Errorf("finished at = %v, want %v", derefOrNil(got), wantFinished)
	}
}

func TestHandleUpdatePlaythroughClearsFields(t *testing.T) {
	server := newTestServer(t)
	playthroughID := startTestPlaythrough(t, server, "First run", "6")

	path := "/games/1942/playthroughs/" + strconv.FormatInt(playthroughID, 10)
	w := postForm(t, server, path, "name=&platform=")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("update status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	libraryGame := getTestUserGame(t, server)
	if len(libraryGame.Playthroughs) != 1 {
		t.Fatalf("len(Playthroughs) = %d, want %d", len(libraryGame.Playthroughs), 1)
	}
	playthrough := libraryGame.Playthroughs[0]
	if playthrough.Playthrough.Name != nil {
		t.Errorf("name = %v, want %v", derefOrNil(playthrough.Playthrough.Name), nil)
	}
	if playthrough.Playthrough.PlatformID != nil {
		t.Errorf("platform id = %v, want %v", derefOrNil(playthrough.Playthrough.PlatformID), nil)
	}
	if playthrough.PlatformName != nil {
		t.Errorf("platform name = %v, want %v", derefOrNil(playthrough.PlatformName), nil)
	}
}

func TestHandleUpdatePlaythroughValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"invalid start date", "started_at=02-01-2026"},
		{"invalid finish date", "finished_at=not-a-date"},
		{"finished on playing playthrough", "started_at=2026-01-02&finished_at=2026-03-04"},
		{"unknown platform", "platform=48"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			playthroughID := startTestPlaythrough(t, server, "", "")

			path := "/games/1942/playthroughs/" + strconv.FormatInt(playthroughID, 10)
			w := postForm(t, server, path, tt.body)
			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestHandleUpdatePlaythroughFinishedBeforeStarted(t *testing.T) {
	server := newTestServer(t)
	playthroughID := startTestPlaythrough(t, server, "", "")

	path := "/games/1942/playthroughs/" + strconv.FormatInt(playthroughID, 10)
	w := postForm(t, server, path+"/outcome", "outcome=completed")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("set outcome status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	w = postForm(t, server, path, "started_at=2026-03-04&finished_at=2026-01-02")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleUpdatePlaythroughUnknownPlaythrough(t *testing.T) {
	server := newTestServer(t)
	playthroughID := startTestPlaythrough(t, server, "", "")

	path := "/games/1942/playthroughs/" + strconv.FormatInt(playthroughID+1, 10)
	w := postForm(t, server, path, "name=Other")
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleDeletePlaythrough(t *testing.T) {
	server := newTestServer(t)
	playthroughID := startTestPlaythrough(t, server, "", "")

	path := "/games/1942/playthroughs/" + strconv.FormatInt(playthroughID, 10) + "/delete"
	w := postForm(t, server, path, "")
	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	assertLocation(t, w, "/games/1942")

	libraryGame := getTestUserGame(t, server)
	if len(libraryGame.Playthroughs) != 0 {
		t.Errorf("playthroughs = %v, want empty", libraryGame.Playthroughs)
	}
}

func TestHandleDeletePlaythroughUnknown(t *testing.T) {
	server := newTestServer(t)
	playthroughID := startTestPlaythrough(t, server, "", "")

	path := "/games/1942/playthroughs/" + strconv.FormatInt(playthroughID+1, 10) + "/delete"
	w := postForm(t, server, path, "")
	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	assertLocation(t, w, "/games/1942")

	libraryGame := getTestUserGame(t, server)
	if len(libraryGame.Playthroughs) != 1 {
		t.Errorf("playthroughs = %v, want 1", libraryGame.Playthroughs)
	}
}

func TestHandleDeletePlaythroughInvalid(t *testing.T) {
	server := newTestServer(t)
	_ = startTestPlaythrough(t, server, "", "")

	path := "/games/1942/playthroughs/abc/delete"
	w := postForm(t, server, path, "")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	libraryGame := getTestUserGame(t, server)
	if len(libraryGame.Playthroughs) != 1 {
		t.Errorf("playthroughs = %v, want 1", libraryGame.Playthroughs)
	}
}
