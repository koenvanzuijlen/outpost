package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHandleExportJSON checks that the download contains the library and the wishlist of the user.
func TestHandleExportJSON(t *testing.T) {
	server := newTestServer(t)
	server.igdbClient = &fakeIGDB{game: loadIGDBGames(t, "games_search.json")[0]}
	w := postForm(t, server, "/games/194662/wishlist", "")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("wishlist status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	addTestGame(t, server)

	w = getPage(t, server, "/export.json")

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/json; charset=utf-8")
	}
	disposition := w.Header().Get("Content-Disposition")
	if !strings.HasPrefix(disposition, "attachment; filename=outpost-") || !strings.HasSuffix(disposition, ".json") {
		t.Errorf("Content-Disposition = %q, want prefix %q and suffix %q",
			disposition, "attachment; filename=outpost-", ".json")
	}

	var file exportFile
	err := json.Unmarshal(w.Body.Bytes(), &file)
	if err != nil {
		t.Fatalf("decode export: %v", err)
	}
	if len(file.Library) != 1 {
		t.Fatalf("len(Library) = %d, want %d", len(file.Library), 1)
	}
	if len(file.Wishlist) != 1 {
		t.Fatalf("len(Wishlist) = %d, want %d", len(file.Wishlist), 1)
	}
	if len(file.Library[0].Copies) != 1 {
		t.Fatalf("len(Library[0].Copies) = %d, want %d", len(file.Library[0].Copies), 1)
	}

	if file.FormatVersion != 1 {
		t.Errorf("FormatVersion = %d, want %d", file.FormatVersion, 1)
	}
	if file.OutpostVersion != testVersion {
		t.Errorf("OutpostVersion = %q, want %q", file.OutpostVersion, testVersion)
	}
	if file.Library[0].IGDBID != 1942 {
		t.Errorf("Library[0].IGDBID = %d, want %d", file.Library[0].IGDBID, 1942)
	}
	if file.Library[0].Status != "backlog" {
		t.Errorf("Library[0].Status = %q, want %q", file.Library[0].Status, "backlog")
	}
	if file.Library[0].Copies[0].Platform.IGDBID != 6 {
		t.Errorf("Library[0].Copies[0].Platform.IGDBID = %d, want %d", file.Library[0].Copies[0].Platform.IGDBID, 6)
	}
	if file.Wishlist[0].IGDBID != 194662 {
		t.Errorf("Wishlist[0].IGDBID = %d, want %d", file.Wishlist[0].IGDBID, 194662)
	}
}

// TestHandleExportJSONDatabaseError checks that a database failure gives the 500 page.
func TestHandleExportJSONDatabaseError(t *testing.T) {
	server := newTestServer(t)

	// Simulate a database failure by closing the database connection.
	_ = server.store.Close()

	w := httptest.NewRecorder()
	server.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/export.json", nil))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// TestHandleExportAllJSON checks that the download of all users holds every user with the own data, and no password
// hash.
func TestHandleExportAllJSON(t *testing.T) {
	server := newTestServer(t)
	addTestGame(t, server)
	addSignedInUser(t, server, "anna")

	w := getPage(t, server, "/admin/export.json")

	disposition := w.Header().Get("Content-Disposition")
	if !strings.HasPrefix(disposition, "attachment; filename=outpost-all-users-") {
		t.Errorf("Content-Disposition = %q, want prefix %q", disposition, "attachment; filename=outpost-all-users-")
	}
	if strings.Contains(w.Body.String(), "$argon2id$") {
		t.Errorf("export body contains a password hash")
	}

	var file exportAllFile
	if err := json.Unmarshal(w.Body.Bytes(), &file); err != nil {
		t.Fatalf("decode export: %v", err)
	}
	if len(file.Users) != 2 {
		t.Fatalf("len(Users) = %d, want %d", len(file.Users), 2)
	}

	wants := []struct {
		username    string
		role        string
		libraryLen  int
		wishlistLen int
	}{
		{"admin", "admin", 1, 0},
		{"anna", "user", 0, 0},
	}
	for i, want := range wants {
		got := file.Users[i]
		if got.Username != want.username {
			t.Errorf("Users[%d].Username = %q, want %q", i, got.Username, want.username)
		}
		if got.Role != want.role {
			t.Errorf("Users[%d].Role = %q, want %q", i, got.Role, want.role)
		}
		if len(got.Library) != want.libraryLen {
			t.Errorf("len(Users[%d].Library) = %d, want %d", i, len(got.Library), want.libraryLen)
		}
		if len(got.Wishlist) != want.wishlistLen {
			t.Errorf("len(Users[%d].Wishlist) = %d, want %d", i, len(got.Wishlist), want.wishlistLen)
		}
	}
}

// TestHandleAdminExport checks that the export page links to the download of all users.
func TestHandleAdminExport(t *testing.T) {
	w := getPage(t, newTestServer(t), "/admin/export")

	if w.Code != http.StatusOK {
		t.Fatalf("GET /admin/export status = %d, want %d", w.Code, http.StatusOK)
	}
	assertBodyContains(t, w, `href="/admin/export.json"`)
}

// TestAdminExportForbidden checks that a user who is not an admin gets a 403 on the export routes of the admin.
func TestAdminExportForbidden(t *testing.T) {
	tests := []struct {
		path       string
		wantStatus int
	}{
		{"/admin/export.json", http.StatusForbidden},
		{"/admin/export", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			server := newTestServer(t)
			_, cookie := addSignedInUser(t, server, "anna")

			w := serveWithCookie(t, server, http.MethodGet, tt.path, "", cookie)
			if w.Code != tt.wantStatus {
				t.Errorf("GET %s status = %d, want %d", tt.path, w.Code, tt.wantStatus)
			}
		})
	}
}
